package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Notwinner0/gvcs/internal/objects"
	"github.com/Notwinner0/gvcs/internal/repo"
)

func CmdCheckout(commitRef, path string) error {
	gitRepo, err := repo.RepoFind(".", true)
	if err != nil {
		return err
	}

	sha, err := objects.ObjectFind(gitRepo, commitRef, "", true)
	if err != nil {
		return err
	}

	obj, err := objects.ObjectRead(gitRepo, sha)
	if err != nil {
		return err
	}

	// If the object is a commit, we grab its tree
	if commit, ok := obj.(*objects.GitCommit); ok {
		treeSHA, ok := commit.Kvlm["tree"]
		if !ok || len(treeSHA) == 0 {
			return errors.New("commit has no tree")
		}
		obj, err = objects.ObjectRead(gitRepo, treeSHA[0])
		if err != nil {
			return err
		}
	}

	tree, ok := obj.(*objects.GitTree)
	if !ok {
		return fmt.Errorf("object %s is not a tree or commit", sha)
	}

	// Verify that path is an empty directory
	if info, err := os.Stat(path); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("not a directory %s", path)
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		if len(entries) > 0 {
			return fmt.Errorf("not empty %s", path)
		}
	} else {
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
	}

	return treeCheckout(gitRepo, tree, path)
}

func treeCheckout(gitRepo *repo.GitRepository, tree *objects.GitTree, path string) error {
	for _, item := range tree.Items {
		obj, err := objects.ObjectRead(gitRepo, item.SHA)
		if err != nil {
			return err
		}
		dest := filepath.Join(path, item.Path)

		switch o := obj.(type) {
		case *objects.GitTree:
			if err := os.Mkdir(dest, 0755); err != nil {
				return err
			}
			if err := treeCheckout(gitRepo, o, dest); err != nil {
				return err
			}
		case *objects.GitBlob:
			// @TODO Support symlinks (identified by mode 120000)
			data, err := o.Serialize()
			if err != nil {
				return err
			}
			if err := os.WriteFile(dest, data, 0644); err != nil {
				return err
			}
		}
	}
	return nil
}
