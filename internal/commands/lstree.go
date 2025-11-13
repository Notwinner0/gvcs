package commands

import (
	"fmt"
	"path/filepath"

	"github.com/Notwinner0/gvcs/internal/objects"
	"github.com/Notwinner0/gvcs/internal/repo"
)

func CmdLsTree(ref string, recursive bool) error {
	gitRepo, err := repo.RepoFind(".", true)
	if err != nil {
		return err
	}
	return lsTree(gitRepo, ref, recursive, "")
}

func lsTree(gitRepo *repo.GitRepository, ref string, recursive bool, prefix string) error {
	sha, err := objects.ObjectFind(gitRepo, ref, "tree", true)
	if err != nil {
		return err
	}
	obj, err := objects.ObjectRead(gitRepo, sha)
	if err != nil {
		return err
	}
	tree, ok := obj.(*objects.GitTree)
	if !ok {
		return fmt.Errorf("object %s is not a tree", sha)
	}

	for _, item := range tree.Items {
		var objType string
		switch item.Mode {
		case "040000":
			objType = "tree"
		case "100644", "100755":
			objType = "blob"
		case "120000":
			objType = "blob" // Symlink
		case "160000":
			objType = "commit" // Submodule
		default:
			return fmt.Errorf("weird tree leaf mode %s", item.Mode)
		}

		fullPath := filepath.Join(prefix, item.Path)

		if recursive && objType == "tree" {
			if err := lsTree(gitRepo, item.SHA, recursive, fullPath); err != nil {
				return err
			}
		} else {
			fmt.Printf("%s %s %s\t%s\n", item.Mode, objType, item.SHA, fullPath)
		}
	}
	return nil
}
