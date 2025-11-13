package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Notwinner0/gvcs/internal/ignore"
	"github.com/Notwinner0/gvcs/internal/index"
	"github.com/Notwinner0/gvcs/internal/objects"
	"github.com/Notwinner0/gvcs/internal/refs"
	"github.com/Notwinner0/gvcs/internal/repo"
)

func CmdStatus() error {
	gitRepo, err := repo.RepoFind(".", true)
	if err != nil {
		return err
	}
	idx, err := index.IndexRead(gitRepo)
	if err != nil {
		return err
	}

	// Part 1: Print current branch
	if err := statusPrintBranch(gitRepo); err != nil {
		return err
	}

	// Part 2: Compare HEAD to index
	if err := statusHeadIndex(gitRepo, idx); err != nil {
		return err
	}

	fmt.Println()

	// Part 3: Compare index to worktree
	if err := statusIndexWorktree(gitRepo, idx); err != nil {
		return err
	}

	return nil
}

func statusPrintBranch(gitRepo *repo.GitRepository) error {
	branch, detached, err := refs.BranchGetActive(gitRepo)
	if err != nil {
		return err
	}
	if detached {
		fmt.Printf("HEAD detached at %s\n", branch)
	} else {
		fmt.Printf("On branch %s\n", branch)
	}
	return nil
}

func statusHeadIndex(gitRepo *repo.GitRepository, idx *index.GitIndex) error {
	fmt.Println("Changes to be committed:")

	headMap, err := objects.TreeToMap(gitRepo, "HEAD", "")
	if err != nil {
		// This can happen in a new repo with no commits
		if strings.Contains(err.Error(), "no such reference") {
			headMap = make(map[string]string)
		} else {
			return err
		}
	}

	indexMap := make(map[string]string)
	for _, entry := range idx.Entries {
		indexMap[entry.Name] = entry.SHA
	}

	// Compare index to head
	for path, sha := range indexMap {
		headSHA, ok := headMap[path]
		if !ok {
			fmt.Printf("  added:    %s\n", path)
		} else if headSHA != sha {
			fmt.Printf("  modified: %s\n", path)
		}
	}

	// Find deleted files
	for path := range headMap {
		if _, ok := indexMap[path]; !ok {
			fmt.Printf("  deleted:  %s\n", path)
		}
	}
	return nil
}

func statusIndexWorktree(gitRepo *repo.GitRepository, idx *index.GitIndex) error {
	fmt.Println("Changes not staged for commit:")

	worktreeFiles := make(map[string]bool)
	err := filepath.Walk(gitRepo.Worktree, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Ignore .git directory and other directories
		if strings.HasPrefix(path, gitRepo.Gitdir) || info.IsDir() {
			return nil
		}
		relPath, _ := filepath.Rel(gitRepo.Worktree, path)
		worktreeFiles[relPath] = true
		return nil
	})
	if err != nil {
		return err
	}

	indexMap := make(map[string]*index.GitIndexEntry)
	for _, e := range idx.Entries {
		indexMap[e.Name] = e
	}

	// Check for modified and deleted files
	for _, entry := range idx.Entries {
		fullPath := filepath.Join(gitRepo.Worktree, entry.Name)
		stat, err := os.Stat(fullPath)
		if os.IsNotExist(err) {
			fmt.Printf("  deleted:  %s\n", entry.Name)
			continue
		}

		// Compare metadata. A simple mtime check is a good start.
		mtime_s := uint32(stat.ModTime().Unix())
		if mtime_s != entry.MTime[0] {
			// Metadata differs, do a full content check
			f, err := os.Open(fullPath)
			if err != nil {
				return err
			}

			// We pass nil for repo so it doesn't write the object
			newSHA, err := objects.ObjectHash(f, "blob", nil)
			f.Close()
			if err != nil {
				return err
			}

			if newSHA != entry.SHA {
				fmt.Printf("  modified: %s\n", entry.Name)
			}
		}
	}

	rules, err := ignore.GitignoreRead(gitRepo)
	if err != nil {
		return err
	}

	fmt.Println("\nUntracked files:")
	for path := range worktreeFiles {
		if _, inIndex := indexMap[path]; !inIndex {
			if !ignore.CheckIgnore(rules, path) {
				fmt.Printf("  %s\n", path)
			}
		}
	}

	return nil
}
