package refs

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Notwinner0/gvcs/internal/repo"
)

// RefResolve follows a ref to its ultimate object SHA.
func RefResolve(gitRepo *repo.GitRepository, ref string) (string, error) {
	path, err := repo.RepoFile(gitRepo, false, ref)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(path)
	// Special case: In a new repo, HEAD points to 'refs/heads/master',
	// but that file doesn't exist yet. This is not an error.
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	content := strings.TrimSpace(string(data))

	if strings.HasPrefix(content, "ref: ") {
		return RefResolve(gitRepo, content[5:])
	}
	return content, nil
}

// RefList recursively collects all refs in a given path.
func RefList(gitRepo *repo.GitRepository, path string) (map[string]interface{}, error) {
	if path == "" {
		path = repo.RepoPath(gitRepo, "refs")
	}

	ret := make(map[string]interface{})
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	// Sort entries for consistent output
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		can := filepath.Join(path, entry.Name())
		if entry.IsDir() {
			sub, err := RefList(gitRepo, can)
			if err != nil {
				return nil, err
			}
			ret[entry.Name()] = sub
		} else {
			// The ref path should be relative to the gitdir
			relPath, err := filepath.Rel(gitRepo.Gitdir, can)
			if err != nil {
				return nil, err
			}
			sha, err := RefResolve(gitRepo, relPath)
			if err != nil {
				return nil, err
			}
			ret[entry.Name()] = sha
		}
	}
	return ret, nil
}

func RefCreate(gitRepo *repo.GitRepository, refName, sha string) error {
	path, err := repo.RepoFile(gitRepo, true, refName)
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(sha+"\n"), 0644)
}

// BranchGetActive reads HEAD to find the current active branch.
func BranchGetActive(gitRepo *repo.GitRepository) (string, bool, error) {
	headFile := repo.RepoPath(gitRepo, "HEAD")
	data, err := os.ReadFile(headFile)
	if err != nil {
		return "", false, err
	}
	content := strings.TrimSpace(string(data))
	if strings.HasPrefix(content, "ref: refs/heads/") {
		return content[16:], false, nil // false means not detached
	}
	return content, true, nil // true means detached HEAD
}

func ShowRef(refs map[string]interface{}, prefix string, withHash bool) {
	for k, v := range refs {
		fullPath := fmt.Sprintf("%s/%s", prefix, k)
		switch val := v.(type) {
		case string:
			if withHash {
				fmt.Printf("%s %s\n", val, fullPath)
			} else {
				fmt.Printf("%s\n", fullPath)
			}
		case map[string]interface{}:
			ShowRef(val, fullPath, withHash)
		}
	}
}
