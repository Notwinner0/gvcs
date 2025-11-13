package commands

import (
	"fmt"
	"time"

	"github.com/Notwinner0/gvcs/internal/index"
	"github.com/Notwinner0/gvcs/internal/repo"
)

func CmdLsFiles(verbose bool) error {
	gitRepo, err := repo.RepoFind(".", true)
	if err != nil {
		return err
	}
	idx, err := index.IndexRead(gitRepo)
	if err != nil {
		return err
	}
	if verbose {
		fmt.Printf("Index file format v%d, containing %d entries.\n", idx.Version, len(idx.Entries))
	}

	for _, e := range idx.Entries {
		fmt.Println(e.Name)
		if verbose {
			fmt.Printf("  Mode: %o\n", e.Mode)
			fmt.Printf("  SHA: %s\n", e.SHA)
			fmt.Printf("  Size: %d\n", e.FSize)
			fmt.Printf("  ctime: %s\n", time.Unix(int64(e.CTime[0]), int64(e.CTime[1])))
			fmt.Printf("  mtime: %s\n", time.Unix(int64(e.MTime[0]), int64(e.MTime[1])))
			// ... add more details if you like
		}
	}
	return nil
}
