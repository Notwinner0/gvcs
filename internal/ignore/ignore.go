package ignore

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/Notwinner0/gvcs/internal/index"
	"github.com/Notwinner0/gvcs/internal/objects"
	"github.com/Notwinner0/gvcs/internal/repo"
)

type gitignoreRule struct {
	Pattern string
	Negate  bool
}

func gitignoreParse(reader io.Reader) []gitignoreRule {
	var rules []gitignoreRule
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		negate := false
		if strings.HasPrefix(line, "!") {
			negate = true
			line = line[1:]
		}

		if strings.HasPrefix(line, `\`) {
			line = line[1:]
		}

		rules = append(rules, gitignoreRule{Pattern: line, Negate: negate})
	}
	return rules
}

type GitIgnore struct {
	Absolute []gitignoreRule
	Scoped   map[string][]gitignoreRule // Key is the directory path
}

func GitignoreRead(gitRepo *repo.GitRepository) (*GitIgnore, error) {
	ignore := &GitIgnore{
		Scoped: make(map[string][]gitignoreRule),
	}

	// Read repo-specific .git/info/exclude
	excludeFile := repo.RepoPath(gitRepo, "info/exclude")
	if f, err := os.Open(excludeFile); err == nil {
		ignore.Absolute = append(ignore.Absolute, gitignoreParse(f)...)
		f.Close()
	}

	// Read global gitignore
	// Simplified: in a real implementation, you'd read git config to find this path.
	if u, err := user.Current(); err == nil {
		globalFile := filepath.Join(u.HomeDir, ".config/git/ignore")
		if f, err := os.Open(globalFile); err == nil {
			ignore.Absolute = append(ignore.Absolute, gitignoreParse(f)...)
			f.Close()
		}
	}

	// Read .gitignore files from the index
	idx, err := index.IndexRead(gitRepo)
	if err != nil {
		return nil, err
	}

	for _, entry := range idx.Entries {
		if filepath.Base(entry.Name) == ".gitignore" {
			obj, err := objects.ObjectRead(gitRepo, entry.SHA)
			if err != nil {
				return nil, err
			}
			blob := obj.(*objects.GitBlob)
			data, _ := blob.Serialize()
			reader := bytes.NewReader(data)
			rules := gitignoreParse(reader)
			dirName := filepath.Dir(entry.Name)
			// For root .gitignore, dirname is "."
			if dirName == "." {
				dirName = ""
			}
			ignore.Scoped[dirName] = rules
		}
	}

	return ignore, nil
}

// CheckIgnore checks if a path should be ignored.
func CheckIgnore(rules *GitIgnore, path string) bool {
	// A path is ignored if it matches a pattern, unless it also matches
	// a later negation pattern.
	ignored := false

	// Absolute rules (global, info/exclude) have the lowest precedence.
	for _, rule := range rules.Absolute {
		// filepath.Match handles glob patterns like *.c
		if matched, _ := filepath.Match(rule.Pattern, path); matched {
			ignored = !rule.Negate
		}
	}

	// Scoped rules (.gitignore files) have higher precedence.
	// We check from the root down to the file's directory.
	dir := ""
	parts := strings.Split(path, string(os.PathSeparator))
	for i := 0; i < len(parts)-1; i++ {
		if scopedRules, ok := rules.Scoped[dir]; ok {
			for _, rule := range scopedRules {
				// We need to match the pattern against the full path.
				// Git's behavior here is complex. For simplicity, we'll
				// assume patterns in subdirectories are relative to that dir.
				// A proper implementation is much more involved.
				fullPattern := filepath.Join(dir, rule.Pattern)
				if matched, _ := filepath.Match(fullPattern, path); matched {
					ignored = !rule.Negate
				}
			}
		}
		dir = filepath.Join(dir, parts[i])
	}

	return ignored
}
