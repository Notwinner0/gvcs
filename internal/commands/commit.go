package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Notwinner0/gvcs/internal/index"
	"github.com/Notwinner0/gvcs/internal/objects"
	"github.com/Notwinner0/gvcs/internal/refs"
	"github.com/Notwinner0/gvcs/internal/repo"
)

func commitCreate(gitRepo *repo.GitRepository, tree, parent, message string) (string, error) {
	commit := &objects.GitCommit{
		Kvlm: make(map[string][]string),
	}
	commit.Kvlm["tree"] = []string{tree}
	if parent != "" {
		commit.Kvlm["parent"] = []string{parent}
	}

	// Get author name and email from git config (mirror libwyag - no fallbacks)
	var authorName, authorEmail string

	// Try repo config first
	if name, email := parseUserConfig(gitRepo.Gitdir, "config"); name != "" && email != "" {
		authorName, authorEmail = name, email
	} else {
		// Try global config
		homeDir, _ := os.UserHomeDir()
		globalPath := filepath.Join(homeDir, ".gitconfig")
		if name, email := parseUserConfig(filepath.Dir(globalPath), filepath.Base(globalPath)); name != "" && email != "" {
			authorName, authorEmail = name, email
		}
	}

	if authorName == "" || authorEmail == "" {
		return "", errors.New("user name and email not configured")
	}
	author := fmt.Sprintf("%s <%s>", authorName, authorEmail)

	timestamp := time.Now().Format("15:04:05 2006 -0700")

	commit.Kvlm["author"] = []string{fmt.Sprintf("%s %d %s", author, time.Now().Unix(), timestamp[len(timestamp)-5:])}
	commit.Kvlm["committer"] = commit.Kvlm["author"]
	commit.Message = message

	return objects.ObjectWrite(commit, gitRepo)
}

// treeFromIndex builds a tree object from the current index.
func treeFromIndex(gitRepo *repo.GitRepository, idx *index.GitIndex) (string, error) {
	// Group entries by directory
	dirEntries := make(map[string][]objects.GitTreeLeaf)
	for _, entry := range idx.Entries {
		dir := filepath.Dir(entry.Name)
		if dir == "." {
			dir = ""
		}
		leaf := objects.GitTreeLeaf{
			// Convert index mode to tree mode string
			Mode: fmt.Sprintf("%o", entry.Mode),
			Path: filepath.Base(entry.Name),
			SHA:  entry.SHA,
		}
		dirEntries[dir] = append(dirEntries[dir], leaf)
	}

	// Build trees from the bottom up
	var dirs []string
	for k := range dirEntries {
		dirs = append(dirs, k)
	}
	// Sort by path length, descending, to process deepest first
	sort.Slice(dirs, func(i, j int) bool {
		return len(dirs[i]) > len(dirs[j])
	})

	treeSHAs := make(map[string]string) // map of dir path to its tree's SHA

	for _, dir := range dirs {
		tree := &objects.GitTree{Items: dirEntries[dir]}

		// Add subtrees that we've already built
		for subDir, sha := range treeSHAs {
			if filepath.Dir(subDir) == dir {
				leaf := objects.GitTreeLeaf{
					Mode: "040000",
					Path: filepath.Base(subDir),
					SHA:  sha,
				}
				tree.Items = append(tree.Items, leaf)
			}
		}

		sha, err := objects.ObjectWrite(tree, gitRepo)
		if err != nil {
			return "", err
		}
		treeSHAs[dir] = sha
	}
	return treeSHAs[""], nil
}

func CmdCommit(message string) error {
	gitRepo, err := repo.RepoFind(".", true)
	if err != nil {
		return err
	}
	idx, err := index.IndexRead(gitRepo)
	if err != nil {
		return err
	}

	// Create trees
	treeSHA, err := treeFromIndex(gitRepo, idx)
	if err != nil {
		return err
	}

	// Get parent commit (mirror libwyag behavior)
	parent, err := objects.ObjectFind(gitRepo, "HEAD", "", true)
	if err != nil {
		// For initial commit, parent is empty
		parent = ""
	}

	// Trim message and add newline (mirror libwyag)
	message = strings.TrimSpace(message) + "\n"

	// Create the commit object
	commitSHA, err := commitCreate(gitRepo, treeSHA, parent, message)
	if err != nil {
		return err
	}

	// Update HEAD
	branch, detached, err := refs.BranchGetActive(gitRepo)
	if err != nil {
		return err
	}
	var refToUpdate string
	if detached {
		refToUpdate = "HEAD"
	} else {
		refToUpdate = "refs/heads/" + branch
	}

	return refs.RefCreate(gitRepo, refToUpdate, commitSHA)
}

// parseUserConfig parses user name and email from a Git config file
func parseUserConfig(dir, filename string) (name, email string) {
	configPath := filepath.Join(dir, filename)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", ""
	}

	lines := strings.Split(string(data), "\n")
	inUserSection := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section := line[1 : len(line)-1]
			inUserSection = (section == "user")
			continue
		}

		if inUserSection {
			if strings.Contains(line, "=") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					switch key {
					case "name":
						name = value
					case "email":
						email = value
					}
				}
			}
		}
	}

	return name, email
}
