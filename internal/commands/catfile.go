package commands

import (
	"os"

	"github.com/Notwinner0/gvcs/internal/objects"
	"github.com/Notwinner0/gvcs/internal/repo"
)

// CmdCatFile is the handler for the cat-file command.
func CmdCatFile(objType, object string) error {
	gitRepo, err := repo.RepoFind(".", true)
	if err != nil {
		return err
	}
	return catFile(gitRepo, object, objType)
}

func catFile(gitRepo *repo.GitRepository, objName, fmt string) error {
	// ObjectFind is a placeholder for now
	sha, err := objects.ObjectFind(gitRepo, objName, fmt, true)
	if err != nil {
		return err
	}
	obj, err := objects.ObjectRead(gitRepo, sha)
	if err != nil {
		return err
	}

	data, err := obj.Serialize()
	if err != nil {
		return err
	}

	_, err = os.Stdout.Write(data)
	return err
}
