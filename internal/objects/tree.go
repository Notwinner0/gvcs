package objects

import (
	"bytes"
	"encoding/hex"
	"errors"
	"path/filepath"
	"sort"

	"github.com/Notwinner0/gvcs/internal/repo"
)

// GitTree represents a tree object.
type GitTree struct {
	Items []GitTreeLeaf
}

func (t *GitTree) Type() string {
	return "tree"
}

func (t *GitTree) Deserialize(data []byte) error {
	var err error
	t.Items, err = treeParse(data)
	return err
}

func (t *GitTree) Serialize() ([]byte, error) {
	return treeSerialize(t)
}

// GitTreeLeaf represents a single entry in a tree object.
type GitTreeLeaf struct {
	Mode string
	Path string
	SHA  string // Stored as a hex string
}

// treeParse parses the raw data of a tree object.
func treeParse(raw []byte) ([]GitTreeLeaf, error) {
	var leaves []GitTreeLeaf
	pos := 0
	for pos < len(raw) {
		// Find the space after the mode
		space := bytes.IndexByte(raw[pos:], ' ')
		if space == -1 {
			return nil, errors.New("invalid tree object: missing space")
		}
		mode := string(raw[pos : pos+space])

		// Find the null terminator of the path
		null := bytes.IndexByte(raw[pos:], '\x00')
		if null == -1 {
			return nil, errors.New("invalid tree object: missing null terminator")
		}
		path := string(raw[pos+space+1 : pos+null])

		// Read the 20-byte SHA
		shaBytes := raw[pos+null+1 : pos+null+21]
		sha := hex.EncodeToString(shaBytes)

		leaves = append(leaves, GitTreeLeaf{Mode: mode, Path: path, SHA: sha})
		pos += null + 21
	}
	return leaves, nil
}

// treeSerialize serializes a GitTree object back to its byte representation.
func treeSerialize(tree *GitTree) ([]byte, error) {
	// Sort the items according to Git's rules
	sort.Slice(tree.Items, func(i, j int) bool {
		pathA := tree.Items[i].Path
		pathB := tree.Items[j].Path

		// Directories are sorted with a trailing slash
		isDirA := tree.Items[i].Mode == "040000"
		isDirB := tree.Items[j].Mode == "040000"

		if isDirA {
			pathA += "/"
		}
		if isDirB {
			pathB += "/"
		}

		return pathA < pathB
	})

	var b bytes.Buffer
	for _, item := range tree.Items {
		// Mode
		b.WriteString(item.Mode)
		b.WriteByte(' ')
		// Path
		b.WriteString(item.Path)
		b.WriteByte('\x00')
		// SHA
		shaBytes, err := hex.DecodeString(item.SHA)
		if err != nil {
			return nil, err
		}
		b.Write(shaBytes)
	}
	return b.Bytes(), nil
}

// TreeToMap recursively reads a tree and flattens it into a map.
func TreeToMap(gitRepo *repo.GitRepository, ref, prefix string) (map[string]string, error) {
	ret := make(map[string]string)
	treeSHA, err := ObjectFind(gitRepo, ref, "tree", true)
	if err != nil {
		return nil, err
	}
	obj, err := ObjectRead(gitRepo, treeSHA)
	if err != nil {
		return nil, err
	}
	tree := obj.(*GitTree)

	for _, leaf := range tree.Items {
		fullPath := filepath.Join(prefix, leaf.Path)
		if leaf.Mode == "040000" { // is a subtree
			subMap, err := TreeToMap(gitRepo, leaf.SHA, fullPath)
			if err != nil {
				return nil, err
			}
			for k, v := range subMap {
				ret[k] = v
			}
		} else { // is a blob
			ret[fullPath] = leaf.SHA
		}
	}
	return ret, nil
}
