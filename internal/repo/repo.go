package repo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bigkevmcd/go-configparser"
)

// GitRepository represents a git repository
type GitRepository struct {
	Worktree string
	Gitdir   string
	Conf     *configparser.ConfigParser
}

// newGitRepository creates a new GitRepository instance
func newGitRepository(path string, force bool) (*GitRepository, error) {
	repo := &GitRepository{
		Worktree: path,
		Gitdir:   filepath.Join(path, ".git"),
	}

	if !force {
		if _, err := os.Stat(repo.Gitdir); os.IsNotExist(err) {
			return nil, fmt.Errorf("not a Git repository %s", path)
		}
	}

	// Read configuration file in .git/config
	cf, err := RepoFile(repo, false, "config")
	if err != nil {
		return nil, err
	}

	// Read the config file and trim spaces from each line before parsing
	data, err := os.ReadFile(cf)
	if err != nil {
		if !force {
			return nil, err
		}
		repo.Conf = configparser.New()
	} else {
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			lines[i] = strings.TrimSpace(line)
		}
		trimmedData := strings.Join(lines, "\n")

		// Create a temporary file with trimmed content
		tmpFile, err := os.CreateTemp("", "gvcs_config_*.ini")
		if err != nil {
			if !force {
				return nil, err
			}
			repo.Conf = configparser.New()
		} else {
			defer os.Remove(tmpFile.Name()) // Clean up temp file
			if _, err := tmpFile.WriteString(trimmedData); err != nil {
				tmpFile.Close()
				if !force {
					return nil, err
				}
				repo.Conf = configparser.New()
			} else {
				tmpFile.Close()
				repo.Conf, err = configparser.NewConfigParserFromFile(tmpFile.Name())
				if err != nil {
					if !force {
						return nil, err
					}
					repo.Conf = configparser.New()
				} else if !force {
					vers, err := repo.Conf.Get("core", "repositoryformatversion")
					if err != nil || vers != "0" {
						return nil, fmt.Errorf("unsupported repositoryformatversion: %s", vers)
					}
				}
			}
		}
	}

	return repo, nil
}

// RepoPath computes a path under the repo's gitdir.
func RepoPath(repo *GitRepository, path ...string) string {
	return filepath.Join(append([]string{repo.Gitdir}, path...)...)
}

// RepoFile returns the path to a file in the gitdir, optionally creating directories.
func RepoFile(repo *GitRepository, mkdir bool, path ...string) (string, error) {
	fullPath := RepoPath(repo, path...)
	if mkdir {
		dir := filepath.Dir(fullPath)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return "", err
			}
		}
	}
	return fullPath, nil
}

// repoDir returns the path to a directory in the gitdir, optionally creating it.
func repoDir(repo *GitRepository, mkdir bool, path ...string) (string, error) {
	pathStr := RepoPath(repo, path...)
	if _, err := os.Stat(pathStr); err == nil {
		// Path exists
		return pathStr, nil
	}

	if mkdir {
		if err := os.MkdirAll(pathStr, 0755); err != nil {
			return "", err
		}
		return pathStr, nil
	}
	return "", os.ErrNotExist
}

// RepoCreate creates a new repository at the given path.
func RepoCreate(path string) (*GitRepository, error) {
	repo, err := newGitRepository(path, true)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(repo.Worktree); !os.IsNotExist(err) {
		if _, err := os.Stat(repo.Gitdir); !os.IsNotExist(err) {
			entries, err := os.ReadDir(repo.Gitdir)
			if err != nil {
				return nil, err
			}
			if len(entries) != 0 {
				return nil, fmt.Errorf("%s is not empty", path)
			}
		}
	} else {
		if err := os.MkdirAll(repo.Worktree, 0755); err != nil {
			return nil, err
		}
	}

	// Create directories
	_, err = repoDir(repo, true, "branches")
	if err != nil {
		return nil, err
	}
	_, err = repoDir(repo, true, "objects")
	if err != nil {
		return nil, err
	}
	_, err = repoDir(repo, true, "refs", "tags")
	if err != nil {
		return nil, err
	}
	_, err = repoDir(repo, true, "refs", "heads")
	if err != nil {
		return nil, err
	}

	// folder hidden on Windows
	hideGitDir(repo.Gitdir)

	// .git/description
	descPath, err := RepoFile(repo, false, "description")
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(descPath, []byte("Unnamed repository; edit this file 'description' to name the repository.\n"), 0644)
	if err != nil {
		return nil, err
	}

	// .git/HEAD
	headPath, err := RepoFile(repo, false, "HEAD")
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(headPath, []byte("ref: refs/heads/master\n"), 0644)
	if err != nil {
		return nil, err
	}

	// .git/config
	configPath, err := RepoFile(repo, false, "config")
	if err != nil {
		return nil, err
	}
	config := repoDefaultConfig()
	err = config.SaveWithDelimiter(configPath, "=")
	if err != nil {
		return nil, err
	}

	return repo, nil
}

// repoDefaultConfig creates a default config parser object.
func repoDefaultConfig() *configparser.ConfigParser {
	config := configparser.New()
	config.AddSection("core")
	config.Set("core", "repositoryformatversion", "0")
	config.Set("core", "filemode", "false")
	config.Set("core", "bare", "false")
	return config
}

// repoFind finds the root of the repository.
func RepoFind(path string, required bool) (*GitRepository, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	gitdir := filepath.Join(absPath, ".git")
	if _, err := os.Stat(gitdir); err == nil {
		return newGitRepository(absPath, false)
	}

	parent := filepath.Dir(absPath)
	if parent == absPath {
		// At the root
		if required {
			return nil, errors.New("no git directory")
		}
		return nil, nil
	}

	return RepoFind(parent, required)
}
