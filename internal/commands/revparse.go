package commands

import (
	"fmt"

	"github.com/Notwinner0/gvcs/internal/objects"
	"github.com/Notwinner0/gvcs/internal/repo"
)

func CmdRevParse(name, objType string) error {
	gitRepo, err := repo.RepoFind(".", true)
	if err != nil {
		return err
	}
	sha, err := objects.ObjectFind(gitRepo, name, objType, true)
	if err != nil {
		return err
	}
	fmt.Println(sha)
	return nil
}
