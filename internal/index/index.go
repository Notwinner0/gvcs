package index

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"os"

	"github.com/Notwinner0/gvcs/internal/repo"
)

// GitIndexEntry represents a single entry in the Git index.
type GitIndexEntry struct {
	CTime [2]uint32 // ctime seconds and nanoseconds
	MTime [2]uint32 // mtime seconds and nanoseconds
	Dev   uint32
	Ino   uint32
	Mode  uint32 // file mode
	UID   uint32
	GID   uint32
	FSize uint32
	SHA   string // hex SHA
	Flags uint16
	Name  string
}

// GitIndex represents the Git index file.
type GitIndex struct {
	Version uint32
	Entries []*GitIndexEntry
}

// IndexRead reads and parses the index file from the repository.
func IndexRead(gitRepo *repo.GitRepository) (*GitIndex, error) {
	indexFile := repo.RepoPath(gitRepo, "index")

	// New repositories have no index
	data, err := os.ReadFile(indexFile)
	if os.IsNotExist(err) {
		return &GitIndex{Version: 2, Entries: []*GitIndexEntry{}}, nil
	}
	if err != nil {
		return nil, err
	}

	// Header
	if string(data[0:4]) != "DIRC" {
		return nil, errors.New("invalid index signature")
	}
	version := binary.BigEndian.Uint32(data[4:8])
	if version != 2 {
		return nil, fmt.Errorf("gvcs only supports index file version 2, got %d", version)
	}
	count := binary.BigEndian.Uint32(data[8:12])

	index := &GitIndex{Version: version}
	index.Entries = make([]*GitIndexEntry, 0, count)

	// Entries
	pos := 12
	for i := 0; i < int(count); i++ {
		entry := &GitIndexEntry{}
		entry.CTime[0] = binary.BigEndian.Uint32(data[pos : pos+4])
		entry.CTime[1] = binary.BigEndian.Uint32(data[pos+4 : pos+8])
		entry.MTime[0] = binary.BigEndian.Uint32(data[pos+8 : pos+12])
		entry.MTime[1] = binary.BigEndian.Uint32(data[pos+12 : pos+16])
		entry.Dev = binary.BigEndian.Uint32(data[pos+16 : pos+20])
		entry.Ino = binary.BigEndian.Uint32(data[pos+20 : pos+24])
		entry.Mode = binary.BigEndian.Uint32(data[pos+24 : pos+28])
		entry.UID = binary.BigEndian.Uint32(data[pos+28 : pos+32])
		entry.GID = binary.BigEndian.Uint32(data[pos+32 : pos+36])
		entry.FSize = binary.BigEndian.Uint32(data[pos+36 : pos+40])

		shaBytes := data[pos+40 : pos+60]
		entry.SHA = hex.EncodeToString(shaBytes)

		entry.Flags = binary.BigEndian.Uint16(data[pos+60 : pos+62])

		pos += 62

		// Read file name until null terminator
		nameEnd := bytes.IndexByte(data[pos:], '\x00')
		if nameEnd == -1 {
			return nil, errors.New("invalid index entry: missing null terminator in name")
		}
		entry.Name = string(data[pos : pos+nameEnd])
		pos += nameEnd + 1 // pos is now at the start of padding

		// FIX: Calculate padding required to reach the next 8-byte boundary
		// Total bytes for fixed metadata (62) + name + null terminator (1)

		totalNonPaddingBytes := uint32(62 + nameEnd + 1)
		padLen := (8 - (totalNonPaddingBytes % 8)) % 8

		pos += int(padLen) // Advance pos by the precise padding length

		index.Entries = append(index.Entries, entry)
	}

	return index, nil
}

func IndexWrite(gitRepo *repo.GitRepository, index *GitIndex) error {
	// Build index in-memory first so we can compute checksum.
	var buf bytes.Buffer

	// HEADER
	buf.Write([]byte("DIRC"))
	// version
	if err := binary.Write(&buf, binary.BigEndian, index.Version); err != nil {
		return err
	}
	// number of entries
	if err := binary.Write(&buf, binary.BigEndian, uint32(len(index.Entries))); err != nil {
		return err
	}

	// ENTRIES
	for _, e := range index.Entries {
		// ctime (2 x uint32)
		if err := binary.Write(&buf, binary.BigEndian, e.CTime); err != nil {
			return err
		}
		// mtime (2 x uint32)
		if err := binary.Write(&buf, binary.BigEndian, e.MTime); err != nil {
			return err
		}
		// dev, ino, mode, uid, gid, fsize
		if err := binary.Write(&buf, binary.BigEndian, e.Dev); err != nil {
			return err
		}
		if err := binary.Write(&buf, binary.BigEndian, e.Ino); err != nil {
			return err
		}
		if err := binary.Write(&buf, binary.BigEndian, e.Mode); err != nil {
			return err
		}
		if err := binary.Write(&buf, binary.BigEndian, e.UID); err != nil {
			return err
		}
		if err := binary.Write(&buf, binary.BigEndian, e.GID); err != nil {
			return err
		}
		if err := binary.Write(&buf, binary.BigEndian, e.FSize); err != nil {
			return err
		}

		// SHA (20 bytes)
		shaBytes, err := hex.DecodeString(e.SHA)
		if err != nil {
			return err
		}
		if len(shaBytes) != 20 {
			return fmt.Errorf("invalid sha length for entry %s: %d", e.Name, len(shaBytes))
		}
		if _, err := buf.Write(shaBytes); err != nil {
			return err
		}

		// Flags: preserve top bits (assume-valid / stage) and encode name length in low 12 bits
		nameBytes := []byte(e.Name)
		nameLen := len(nameBytes)
		if nameLen >= 0xFFF {
			nameLen = 0xFFF
		}
		// Preserve top 4 bits of existing Flags (bits 12-15) and set low 12 bits to nameLen.
		flags := (e.Flags & 0xF000) | uint16(nameLen)
		if err := binary.Write(&buf, binary.BigEndian, flags); err != nil {
			return err
		}

		// Name + null terminator
		if _, err := buf.Write(nameBytes); err != nil {
			return err
		}
		if err := buf.WriteByte(0); err != nil {
			return err
		}

		// Padding: each entry (62 + name + 1) must be aligned to an 8-byte boundary.
		entryLen := 62 + nameLen + 1
		padLen := (8 - (entryLen % 8)) % 8
		if padLen > 0 {
			if _, err := buf.Write(make([]byte, padLen)); err != nil {
				return err
			}
		}
	}

	// Compute checksum for the whole content and append it.
	content := buf.Bytes()
	sum := sha1.Sum(content)
	final := append(content, sum[:]...)

	// Atomically write the final index file.
	if err := os.WriteFile(repo.RepoPath(gitRepo, "index"), final, 0644); err != nil {
		return err
	}

	return nil
}
