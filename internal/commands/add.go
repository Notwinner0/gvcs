package commands

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/Notwinner0/gvcs/internal/index"
	"github.com/Notwinner0/gvcs/internal/objects"
	"github.com/Notwinner0/gvcs/internal/repo"
)

func CmdAdd(paths []string) error {
	gitRepo, err := repo.RepoFind(".", true)
	if err != nil {
		return err
	}
	idx, err := index.IndexRead(gitRepo)
	if err != nil {
		return err
	}

	// Create a map for quick access to existing entries
	indexMap := make(map[string]*index.GitIndexEntry)
	var keptEntries []*index.GitIndexEntry
	for _, entry := range idx.Entries {
		indexMap[entry.Name] = entry
		// Keep all entries that are *not* being added
		isAdding := false
		for _, p := range paths {
			if entry.Name == p {
				isAdding = true
				break
			}
		}
		if !isAdding {
			keptEntries = append(keptEntries, entry)
		}
	}

	// Process files to add
	for _, path := range paths {
		fullPath := filepath.Join(gitRepo.Worktree, path)
		f, err := os.Open(fullPath)
		if err != nil {
			return err
		}

		sha, err := objects.ObjectHash(f, "blob", gitRepo)
		f.Close()
		if err != nil {
			return err
		}

		stat, err := os.Stat(fullPath)
		if err != nil {
			return err
		}

		// Mode for regular file is 100644
		mode := uint32(0100644)

		relPath, err := filepath.Rel(gitRepo.Worktree, fullPath)
		if err != nil {
			return err
		}

		entry := &index.GitIndexEntry{
			CTime: [2]uint32{uint32(stat.ModTime().Unix()), uint32(stat.ModTime().Nanosecond())},
			MTime: [2]uint32{uint32(stat.ModTime().Unix()), uint32(stat.ModTime().Nanosecond())},
			Dev:   0,
			Ino:   0,
			Mode:  mode,
			UID:   0,
			GID:   0,
			FSize: uint32(stat.Size()),
			SHA:   sha,
			Name:  relPath,
		}
		keptEntries = append(keptEntries, entry)
	}

	// Sort entries by name, as Git requires
	sort.Slice(keptEntries, func(i, j int) bool {
		return keptEntries[i].Name < keptEntries[j].Name
	})

	idx.Entries = keptEntries
	return index.IndexWrite(gitRepo, idx)
}
