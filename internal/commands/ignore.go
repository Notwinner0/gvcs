package commands

import (
	"fmt"

	"github.com/Notwinner0/gvcs/internal/ignore"
	"github.com/Notwinner0/gvcs/internal/repo"
)

func CmdCheckIgnore(paths []string) error {
	gitRepo, err := repo.RepoFind(".", true)
	if err != nil {
		return err
	}
	rules, err := ignore.GitignoreRead(gitRepo)
	if err != nil {
		return err
	}
	for _, path := range paths {
		if ignore.CheckIgnore(rules, path) {
			fmt.Println(path)
		}
	}
	return nil
}
