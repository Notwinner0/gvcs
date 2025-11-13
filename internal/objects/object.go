package objects

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/Notwinner0/gvcs/internal/refs"
	"github.com/Notwinner0/gvcs/internal/repo"
)

// GitObject defines the interface for all git objects.
type GitObject interface {
	Serialize() ([]byte, error)
	Deserialize(data []byte) error
	Type() string
}

// ObjectRead reads an object from the repository.
func ObjectRead(gitRepo *repo.GitRepository, sha string) (GitObject, error) {
	path, err := repo.RepoFile(gitRepo, false, "objects", sha[0:2], sha[2:])
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	z, err := zlib.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer z.Close()

	raw, err := io.ReadAll(z)
	if err != nil {
		return nil, err
	}

	// Read object type
	spaceIndex := bytes.IndexByte(raw, ' ')
	if spaceIndex == -1 {
		return nil, errors.New("invalid object format: missing space")
	}
	fmtStr := string(raw[0:spaceIndex])

	// Read and validate object size
	nullIndex := bytes.IndexByte(raw, '\x00')
	if nullIndex == -1 {
		return nil, errors.New("invalid object format: missing null terminator")
	}
	size, err := strconv.Atoi(string(raw[spaceIndex+1 : nullIndex]))
	if err != nil {
		return nil, err
	}
	if size != len(raw)-nullIndex-1 {
		return nil, fmt.Errorf("malformed object %s: bad length", sha)
	}

	var obj GitObject
	switch fmtStr {
	case "commit":
		obj = new(GitCommit)
	case "tree":
		obj = new(GitTree)
	case "tag":
		obj = new(GitTag)
	case "blob":
		obj = new(GitBlob)
	default:
		return nil, fmt.Errorf("unknown type %s for object %s", fmtStr, sha)
	}

	err = obj.Deserialize(raw[nullIndex+1:])
	if err != nil {
		return nil, err
	}

	return obj, nil
}

// ObjectWrite writes a GitObject to the repository.
func ObjectWrite(obj GitObject, gitRepo *repo.GitRepository) (string, error) {
	data, err := obj.Serialize()
	if err != nil {
		return "", err
	}

	// Add header
	header := []byte(fmt.Sprintf("%s %d\x00", obj.Type(), len(data)))
	result := append(header, data...)

	// Compute hash
	h := sha1.New()
	h.Write(result)
	sha := hex.EncodeToString(h.Sum(nil))

	if gitRepo != nil {
		path, err := repo.RepoFile(gitRepo, true, "objects", sha[0:2], sha[2:])
		if err != nil {
			return "", err
		}

		f, err := os.Create(path)
		if err != nil {
			return "", err
		}
		defer f.Close()

		w := zlib.NewWriter(f)
		_, err = w.Write(result)
		if err != nil {
			return "", err
		}
		w.Close()
	}

	return sha, nil
}

// ObjectFind resolves a name and optionally follows it to the desired object type.
func ObjectFind(gitRepo *repo.GitRepository, name, objType string, follow bool) (string, error) {
	shas, err := objectResolve(gitRepo, name)
	if err != nil {
		return "", err
	}

	if len(shas) == 0 {
		return "", fmt.Errorf("no such reference %s", name)
	}
	if len(shas) > 1 {
		return "", fmt.Errorf("ambiguous reference %s: candidates are %v", name, shas)
	}
	sha := shas[0]

	if objType == "" {
		return sha, nil
	}

	// Follow tags and commits if needed
	for {
		obj, err := ObjectRead(gitRepo, sha)
		if err != nil {
			return "", err
		}

		if obj.Type() == objType {
			return sha, nil
		}

		if !follow {
			return "", nil
		}

		switch o := obj.(type) {
		case *GitTag:
			sha = o.Kvlm["object"][0]
		case *GitCommit:
			if objType == "tree" {
				sha = o.Kvlm["tree"][0]
			} else {
				return "", nil
			}
		default:
			return "", nil
		}
	}
}

// ObjectHash hashes an object from a reader.
func ObjectHash(fd io.Reader, objType string, gitRepo *repo.GitRepository) (string, error) {
	data, err := io.ReadAll(fd)
	if err != nil {
		return "", err
	}

	var obj GitObject
	switch objType {
	case "commit":
		obj = new(GitCommit)
		obj.Deserialize(data)
	case "tree":
		obj = new(GitTree)
		obj.Deserialize(data)
	case "tag":
		obj = new(GitTag)
		obj.Deserialize(data)
	case "blob":
		obj = new(GitBlob)
		err = obj.Deserialize(data)
		if err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unknown type %s", objType) // objtype instead of fmt as fmt is format, but it is a package
	}

	return ObjectWrite(obj, gitRepo)
}

// objectResolve resolves a name to a list of candidate object hashes.
func objectResolve(gitRepo *repo.GitRepository, name string) ([]string, error) {
	if name == "" {
		return nil, nil
	}
	if name == "HEAD" {
		sha, err := refs.RefResolve(gitRepo, "HEAD")
		if err != nil {
			return nil, err
		}
		if sha == "" {
			return nil, nil
		}
		return []string{sha}, nil
	}

	var candidates []string
	hashRE := regexp.MustCompile(`^[0-9a-fA-F]{4,40}$/`)

	// Is it a hash?
	if hashRE.MatchString(name) {
		name = strings.ToLower(name)
		prefix := name[0:2]
		path := repo.RepoPath(gitRepo, "objects", prefix)
		if _, err := os.Stat(path); err == nil {
			rem := name[2:]
			files, err := os.ReadDir(path)
			if err != nil {
				return nil, err
			}
			for _, f := range files {
				if strings.HasPrefix(f.Name(), rem) {
					candidates = append(candidates, prefix+f.Name())
				}
			}
		}
	}

	// Try for references
	for _, refPath := range []string{"refs/tags/", "refs/heads/"} {
		if sha, err := refs.RefResolve(gitRepo, refPath+name); err == nil && sha != "" {
			candidates = append(candidates, sha)
		}
	}

	// Remove duplicates
	uniqueCandidates := make(map[string]bool)
	var result []string
	for _, c := range candidates {
		if !uniqueCandidates[c] {
			uniqueCandidates[c] = true
			result = append(result, c)
		}
	}

	return result, nil
}
