package commands

import (
	"github.com/Notwinner0/gvcs/internal/refs"
	"github.com/Notwinner0/gvcs/internal/repo"
)

func CmdShowRef() error {
	gitRepo, err := repo.RepoFind(".", true)
	if err != nil {
		return err
	}
	refList, err := refs.RefList(gitRepo, "")
	if err != nil {
		return err
	}
	refs.ShowRef(refList, "refs", true)
	return nil
}
