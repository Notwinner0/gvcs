package commands

import (
	"os"
	"path/filepath"

	"github.com/Notwinner0/gvcs/internal/index"
	"github.com/Notwinner0/gvcs/internal/repo"
)

func CmdRm(paths []string) error {
	gitRepo, err := repo.RepoFind(".", true)
	if err != nil {
		return err
	}
	return rm(gitRepo, paths)
}

func rm(gitRepo *repo.GitRepository, paths []string) error {
	idx, err := index.IndexRead(gitRepo)
	if err != nil {
		return err
	}

	// Create a set of paths to remove for efficient lookup
	toRemove := make(map[string]bool)
	for _, p := range paths {
		toRemove[p] = true
	}

	var keptEntries []*index.GitIndexEntry
	for _, entry := range idx.Entries {
		if !toRemove[entry.Name] {
			keptEntries = append(keptEntries, entry)
		}
	}

	idx.Entries = keptEntries
	if err := index.IndexWrite(gitRepo, idx); err != nil {
		return err
	}

	// Physically delete the files
	for _, path := range paths {
		if err := os.Remove(filepath.Join(gitRepo.Worktree, path)); err != nil {
			// Don't fail if file is already gone
			if !os.IsNotExist(err) {
				return err
			}
		}
	}

	return nil
}
