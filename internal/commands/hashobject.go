package commands

import (
	"fmt"
	"os"

	"github.com/Notwinner0/gvcs/internal/objects"
	"github.com/Notwinner0/gvcs/internal/repo"
)

func CmdHashObject(write bool, objType, path string) error {
	var gitRepo *repo.GitRepository
	var err error

	if write {
		gitRepo, err = repo.RepoFind(".", true)
		if err != nil {
			return err
		}
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	sha, err := objects.ObjectHash(f, objType, gitRepo)
	if err != nil {
		return err
	}

	fmt.Println(sha)
	return nil
}
