package commands

import (
	"fmt"
	"strings"

	"github.com/Notwinner0/gvcs/internal/objects"
	"github.com/Notwinner0/gvcs/internal/repo"
)

// CmdLog is the handler for the log command.
func CmdLog(commitRef string) error {
	gitRepo, err := repo.RepoFind(".", true)
	if err != nil {
		return err
	}

	fmt.Println("digraph gvcslog{")
	fmt.Println("  node[shape=rect]")

	sha, err := objects.ObjectFind(gitRepo, commitRef, "", true)
	if err != nil {
		return err
	}

	seen := make(map[string]bool)
	err = logGraphviz(gitRepo, sha, seen)
	if err != nil {
		return err
	}

	fmt.Println("}")
	return nil
}

func logGraphviz(gitRepo *repo.GitRepository, sha string, seen map[string]bool) error {
	if seen[sha] {
		return nil
	}
	seen[sha] = true

	obj, err := objects.ObjectRead(gitRepo, sha)
	if err != nil {
		return err
	}
	commit, ok := obj.(*objects.GitCommit)
	if !ok {
		return fmt.Errorf("object %s is not a commit", sha)
	}

	message := strings.TrimSpace(commit.Message)
	// Escape backslashes and quotes for dot format
	message = strings.ReplaceAll(message, "\\", "\\\\")
	message = strings.ReplaceAll(message, "\"", "\\\"")
	if idx := strings.Index(message, "\n"); idx != -1 {
		message = message[:idx] // Keep only the first line
	}

	fmt.Printf("  c_%s [label=\"%s: %s\"]\n", sha, sha[0:7], message)

	if parents, ok := commit.Kvlm["parent"]; ok {
		for _, p := range parents {
			p = strings.TrimSpace(p)
			fmt.Printf("  c_%s -> c_%s;\n", sha, p)
			if err := logGraphviz(gitRepo, p, seen); err != nil {
				return err
			}
		}
	}
	return nil
}
