package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/akamensky/argparse"
	"github.com/bigkevmcd/go-configparser"
)

func main() {
	parser := argparse.NewParser("gvcs", "The versionfull content tracker")
	// Create sub-commands
	initCmd := parser.NewCommand("init", "Initialize a new, empty repository.")
	initPath := initCmd.String("p", "path", &argparse.Options{Required: false, Default: ".", Help: "Where to create the repository."})
	catFileCmd := parser.NewCommand("cat-file", "Provide content of repository objects")
	catFileType := catFileCmd.String("t", "type", &argparse.Options{Required: true, Help: "Specify the type"})
	catFileObject := catFileCmd.String("o", "object", &argparse.Options{Required: true, Help: "The object to display"})
	hashObjectCmd := parser.NewCommand("hash-object", "Compute object ID and optionally creates a blob from a file")
	hashObjectType := hashObjectCmd.String("t", "type", &argparse.Options{Default: "blob", Help: "Specify the type"})
	hashObjectWrite := hashObjectCmd.Flag("w", "write", &argparse.Options{Help: "Actually write the object into the database"})
	hashObjectPath := hashObjectCmd.StringPositional(&argparse.Options{Required: true, Help: "Read object from <file>"})
	logCmd := parser.NewCommand("log", "Display history of a given commit.")
	logCommit := logCmd.String("c", "commit", &argparse.Options{Required: false, Default: "HEAD", Help: "Commit to start at."})
	lsTreeCmd := parser.NewCommand("ls-tree", "Pretty-print a tree object.")
	lsTreeRecursive := lsTreeCmd.Flag("r", "recursive", &argparse.Options{Help: "Recurse into sub-trees"})
	lsTreeObject := lsTreeCmd.StringPositional(&argparse.Options{Required: true, Help: "A tree-ish object."})
	checkoutCmd := parser.NewCommand("checkout", "Checkout a commit inside of a directory.")
	checkoutCommit := checkoutCmd.StringPositional(&argparse.Options{Required: true, Help: "The commit or tree to checkout."})
	checkoutPath := checkoutCmd.StringPositional(&argparse.Options{Required: true, Help: "The EMPTY directory to checkout on."})
	showRefCmd := parser.NewCommand("show-ref", "List references.")
	tagCmd := parser.NewCommand("tag", "List and create tags")
	tagAnnotated := tagCmd.Flag("a", "annotated", &argparse.Options{Help: "Whether to create a tag object"})
	tagName := tagCmd.String("n", "name", &argparse.Options{Help: "The new tag's name"})
	tagObject := tagCmd.String("o", "object", &argparse.Options{Default: "HEAD", Help: "The object the new tag will point to"})
	revParseCmd := parser.NewCommand("rev-parse", "Parse revision (or other objects) identifiers")
	revParseType := revParseCmd.String("t", "type", &argparse.Options{Help: "Specify the expected type"})
	revParseName := revParseCmd.StringPositional(&argparse.Options{Required: true, Help: "The name to parse"})
	lsFilesCmd := parser.NewCommand("ls-files", "List all the staged files")
	lsFilesVerbose := lsFilesCmd.Flag("v", "verbose", &argparse.Options{Help: "Show everything."})
	statusCmd := parser.NewCommand("status", "Show the working tree status.")
	checkIgnoreCmd := parser.NewCommand("check-ignore", "Check path(s) against ignore rules.")
	checkIgnorePaths := checkIgnoreCmd.StringList("", "paths", &argparse.Options{Required: true, Help: "Paths to check"})
	rmCmd := parser.NewCommand("rm", "Remove files from the working tree and the index.")
	rmPaths := rmCmd.StringList("", "files", &argparse.Options{Required: true, Help: "Files to remove"})
	addCmd := parser.NewCommand("add", "Add file contents to the index.")
	addPaths := addCmd.StringList("f", "files", &argparse.Options{Required: true, Help: "Files to add"})
	commitCmd := parser.NewCommand("commit", "Record changes to the repository.")
	commitMessage := commitCmd.String("m", "message", &argparse.Options{Required: true, Help: "Message to associate with this commit."})
	// ... other commands will be added here
	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
		return
	}

	switch {
	case initCmd.Happened():
		_, err := repoCreate(*initPath)
		if err != nil {
			log.Fatalf("Error creating repository: %v", err)
		}
		fmt.Printf("Initialized empty gvcs repository in %s\n", *initPath)
		break
	case catFileCmd.Happened():
		err := cmdCatFile(*catFileType, *catFileObject)
		if err != nil {
			log.Fatalf("Error cat-file: %v", err)
		}
		break
	case hashObjectCmd.Happened():
		err := cmdHashObject(*hashObjectWrite, *hashObjectType, *hashObjectPath)
		if err != nil {
			log.Fatalf("Error hash-object: %v", err)
		}
		break
	case logCmd.Happened():
		err := cmdLog(*logCommit)
		if err != nil {
			log.Fatalf("Error log: %v", err)
		}
		break
	case lsTreeCmd.Happened():
		err := cmdLsTree(*lsTreeObject, *lsTreeRecursive)
		if err != nil {
			log.Fatalf("Error ls-tree: %v", err)
		}
		break
	case checkoutCmd.Happened():
		err := cmdCheckout(*checkoutCommit, *checkoutPath)
		if err != nil {
			log.Fatalf("Error checkout: %v", err)
		}
		break
	case showRefCmd.Happened():
		err := cmdShowRef()
		if err != nil {
			log.Fatalf("Error show-ref: %v", err)
		}
		break
	case tagCmd.Happened():
		err := cmdTag(*tagName, *tagObject, *tagAnnotated)
		if err != nil {
			log.Fatalf("Error tag: %v", err)
		}
		break
	case revParseCmd.Happened():
		err := cmdRevParse(*revParseName, *revParseType)
		if err != nil {
			log.Fatalf("Error rev-parse: %v", err)
		}
		break
	case lsFilesCmd.Happened():
		err := cmdLsFiles(*lsFilesVerbose)
		if err != nil {
			log.Fatalf("Error ls-files: %v", err)
		}
		break
	case statusCmd.Happened():
		err := cmdStatus()
		if err != nil {
			log.Fatalf("Error status: %v", err)
		}
		break
	case checkIgnoreCmd.Happened():
		err := cmdCheckIgnore(*checkIgnorePaths)
		if err != nil {
			log.Fatalf("Error check-ignore: %v", err)
		}
		break
	case rmCmd.Happened():
		err := cmdRm(*rmPaths)
		if err != nil {
			log.Fatalf("Error rm: %v", err)
		}
		break
	case addCmd.Happened():
		err := cmdAdd(*addPaths)
		if err != nil {
			log.Fatalf("Error add: %v", err)
		}
		break
	case commitCmd.Happened():
		err := cmdCommit(*commitMessage)
		if err != nil {
			log.Fatalf("Error commit: %v", err)
		}
		break
	// ... other command cases will be here
	default:
		log.Fatal("Bad command.")
	}
}

// GitRepository represents a git repository
type GitRepository struct {
	worktree string
	gitdir   string
	conf     *configparser.ConfigParser
}

// newGitRepository creates a new GitRepository instance
func newGitRepository(path string, force bool) (*GitRepository, error) {
	repo := &GitRepository{
		worktree: path,
		gitdir:   filepath.Join(path, ".git"),
	}

	if !force {
		if _, err := os.Stat(repo.gitdir); os.IsNotExist(err) {
			return nil, fmt.Errorf("not a Git repository %s", path)
		}
	}

	// Read configuration file in .git/config
	cf, err := repoFile(repo, false, "config")
	if err != nil {
		return nil, err
	}

	// Read the config file and trim spaces from each line before parsing
	data, err := os.ReadFile(cf)
	if err != nil {
		if !force {
			return nil, err
		}
		repo.conf = configparser.New()
	} else {
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			lines[i] = strings.TrimSpace(line)
		}
		trimmedData := strings.Join(lines, "\n")

		// Create a temporary file with trimmed content
		tmpFile, err := os.CreateTemp("", "gvcs_config_*.ini")
		if err != nil {
			if !force {
				return nil, err
			}
			repo.conf = configparser.New()
		} else {
			defer os.Remove(tmpFile.Name()) // Clean up temp file
			if _, err := tmpFile.WriteString(trimmedData); err != nil {
				tmpFile.Close()
				if !force {
					return nil, err
				}
				repo.conf = configparser.New()
			} else {
				tmpFile.Close()
				repo.conf, err = configparser.NewConfigParserFromFile(tmpFile.Name())
				if err != nil {
					if !force {
						return nil, err
					}
					repo.conf = configparser.New()
				} else if !force {
					vers, err := repo.conf.Get("core", "repositoryformatversion")
					if err != nil || vers != "0" {
						return nil, fmt.Errorf("unsupported repositoryformatversion: %s", vers)
					}
				}
			}
		}
	}

	return repo, nil
}

// repoPath computes a path under the repo's gitdir.
func repoPath(repo *GitRepository, path ...string) string {
	return filepath.Join(append([]string{repo.gitdir}, path...)...)
}

// repoFile returns the path to a file in the gitdir, optionally creating directories.
func repoFile(repo *GitRepository, mkdir bool, path ...string) (string, error) {
	fullPath := repoPath(repo, path...)
	if mkdir {
		dir := filepath.Dir(fullPath)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return "", err
			}
		}
	}
	return fullPath, nil
}

// repoDir returns the path to a directory in the gitdir, optionally creating it.
func repoDir(repo *GitRepository, mkdir bool, path ...string) (string, error) {
	pathStr := repoPath(repo, path...)
	if _, err := os.Stat(pathStr); err == nil {
		// Path exists
		return pathStr, nil
	}

	if mkdir {
		if err := os.MkdirAll(pathStr, 0755); err != nil {
			return "", err
		}
		return pathStr, nil
	}
	return "", os.ErrNotExist
}

// repoCreate creates a new repository at the given path.
func repoCreate(path string) (*GitRepository, error) {
	repo, err := newGitRepository(path, true)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(repo.worktree); !os.IsNotExist(err) {
		if _, err := os.Stat(repo.gitdir); !os.IsNotExist(err) {
			entries, err := os.ReadDir(repo.gitdir)
			if err != nil {
				return nil, err
			}
			if len(entries) != 0 {
				return nil, fmt.Errorf("%s is not empty", path)
			}
		}
	} else {
		if err := os.MkdirAll(repo.worktree, 0755); err != nil {
			return nil, err
		}
	}

	// Create directories
	_, err = repoDir(repo, true, "branches")
	if err != nil {
		return nil, err
	}
	_, err = repoDir(repo, true, "objects")
	if err != nil {
		return nil, err
	}
	_, err = repoDir(repo, true, "refs", "tags")
	if err != nil {
		return nil, err
	}
	_, err = repoDir(repo, true, "refs", "heads")
	if err != nil {
		return nil, err
	}

	// folder hidden on Windows
	if runtime.GOOS == "windows" {
		pathPtr, err := syscall.UTF16PtrFromString(repo.gitdir)
		if err == nil {
			_ = syscall.SetFileAttributes(pathPtr, syscall.FILE_ATTRIBUTE_HIDDEN)
		}
	}

	// .git/description
	descPath, err := repoFile(repo, false, "description")
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(descPath, []byte("Unnamed repository; edit this file 'description' to name the repository.\n"), 0644)
	if err != nil {
		return nil, err
	}

	// .git/HEAD
	headPath, err := repoFile(repo, false, "HEAD")
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(headPath, []byte("ref: refs/heads/master\n"), 0644)
	if err != nil {
		return nil, err
	}

	// .git/config
	configPath, err := repoFile(repo, false, "config")
	if err != nil {
		return nil, err
	}
	config := repoDefaultConfig()
	err = config.SaveWithDelimiter(configPath, "=")
	if err != nil {
		return nil, err
	}

	return repo, nil
}

// repoDefaultConfig creates a default config parser object.
func repoDefaultConfig() *configparser.ConfigParser {
	config := configparser.New()
	config.AddSection("core")
	config.Set("core", "repositoryformatversion", "0")
	config.Set("core", "filemode", "false")
	config.Set("core", "bare", "false")
	return config
}

// repoFind finds the root of the repository.
func repoFind(path string, required bool) (*GitRepository, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	gitdir := filepath.Join(absPath, ".git")
	if _, err := os.Stat(gitdir); err == nil {
		return newGitRepository(absPath, false)
	}

	parent := filepath.Dir(absPath)
	if parent == absPath {
		// At the root
		if required {
			return nil, errors.New("no git directory")
		}
		return nil, nil
	}

	return repoFind(parent, required)
}

// GitObject defines the interface for all git objects.
type GitObject interface {
	Serialize() ([]byte, error)
	Deserialize(data []byte) error
	Type() string
}

// objectRead reads an object from the repository.
func objectRead(repo *GitRepository, sha string) (GitObject, error) {
	path, err := repoFile(repo, false, "objects", sha[0:2], sha[2:])
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	z, err := zlib.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer z.Close()

	raw, err := io.ReadAll(z)
	if err != nil {
		return nil, err
	}

	// Read object type
	spaceIndex := bytes.IndexByte(raw, ' ')
	if spaceIndex == -1 {
		return nil, errors.New("invalid object format: missing space")
	}
	fmtStr := string(raw[0:spaceIndex])

	// Read and validate object size
	nullIndex := bytes.IndexByte(raw, '\x00')
	if nullIndex == -1 {
		return nil, errors.New("invalid object format: missing null terminator")
	}
	size, err := strconv.Atoi(string(raw[spaceIndex+1 : nullIndex]))
	if err != nil {
		return nil, err
	}
	if size != len(raw)-nullIndex-1 {
		return nil, fmt.Errorf("malformed object %s: bad length", sha)
	}

	var obj GitObject
	switch fmtStr {
	case "commit":
		obj = new(GitCommit)
	case "tree":
		obj = new(GitTree)
	case "tag":
		obj = new(GitTag)
	case "blob":
		obj = new(GitBlob)
	default:
		return nil, fmt.Errorf("unknown type %s for object %s", fmtStr, sha)
	}

	err = obj.Deserialize(raw[nullIndex+1:])
	if err != nil {
		return nil, err
	}

	return obj, nil
}

// objectWrite writes a GitObject to the repository.
func objectWrite(obj GitObject, repo *GitRepository) (string, error) {
	data, err := obj.Serialize()
	if err != nil {
		return "", err
	}

	// Add header
	header := []byte(fmt.Sprintf("%s %d\x00", obj.Type(), len(data)))
	result := append(header, data...)

	// Compute hash
	h := sha1.New()
	h.Write(result)
	sha := hex.EncodeToString(h.Sum(nil))

	if repo != nil {
		path, err := repoFile(repo, true, "objects", sha[0:2], sha[2:])
		if err != nil {
			return "", err
		}

		f, err := os.Create(path)
		if err != nil {
			return "", err
		}
		defer f.Close()

		w := zlib.NewWriter(f)
		_, err = w.Write(result)
		if err != nil {
			return "", err
		}
		w.Close()
	}

	return sha, nil
}

// GitBlob represents a blob object.
type GitBlob struct {
	data []byte
}

func (b *GitBlob) Serialize() ([]byte, error) {
	return b.data, nil
}

func (b *GitBlob) Deserialize(data []byte) error {
	b.data = data
	return nil
}

func (b *GitBlob) Type() string {
	return "blob"
}

// cmdCatFile is the handler for the cat-file command.
func cmdCatFile(objType, object string) error {
	repo, err := repoFind(".", true)
	if err != nil {
		return err
	}
	return catFile(repo, object, objType)
}

func catFile(repo *GitRepository, objName, fmt string) error {
	// objectFind is a placeholder for now
	sha, err := objectFind(repo, objName, fmt, true)
	if err != nil {
		return err
	}
	obj, err := objectRead(repo, sha)
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

// objectFind resolves a name and optionally follows it to the desired object type.
func objectFind(repo *GitRepository, name, objType string, follow bool) (string, error) {
	shas, err := objectResolve(repo, name)
	if err != nil {
		return "", err
	}

	if len(shas) == 0 {
		return "", fmt.Errorf("no such reference %s", name)
	}
	if len(shas) > 1 {
		return "", fmt.Errorf("ambiguous reference %s: candidates are %v", name, shas)
	}
	sha := shas[0]

	if objType == "" {
		return sha, nil
	}

	// Follow tags and commits if needed
	for {
		obj, err := objectRead(repo, sha)
		if err != nil {
			return "", err
		}

		if obj.Type() == objType {
			return sha, nil
		}

		if !follow {
			return "", nil
		}

		switch o := obj.(type) {
		case *GitTag:
			sha = o.kvlm["object"][0]
		case *GitCommit:
			if objType == "tree" {
				sha = o.kvlm["tree"][0]
			} else {
				return "", nil
			}
		default:
			return "", nil
		}
	}
}

func cmdHashObject(write bool, objType, path string) error {
	var repo *GitRepository
	var err error

	if write {
		repo, err = repoFind(".", true)
		if err != nil {
			return err
		}
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	sha, err := objectHash(f, objType, repo)
	if err != nil {
		return err
	}

	fmt.Println(sha)
	return nil
}

// objectHash hashes an object from a reader.
func objectHash(fd io.Reader, objType string, repo *GitRepository) (string, error) {
	data, err := io.ReadAll(fd)
	if err != nil {
		return "", err
	}

	var obj GitObject
	switch objType {
	case "commit":
		obj = new(GitCommit)
		obj.Deserialize(data)
	case "tree":
		obj = new(GitTree)
		obj.Deserialize(data)
	case "tag":
		obj = new(GitTag)
		obj.Deserialize(data)
	case "blob":
		obj = new(GitBlob)
		err = obj.Deserialize(data)
		if err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unknown type %s", objType) // objtype instead of fmt as fmt is format, but it is a package
	}

	return objectWrite(obj, repo)
}

// kvlmParse parses a key-value list with message format.
func kvlmParse(raw []byte) (map[string][]string, string, error) {
	kvlm := make(map[string][]string)
	var message string

	// Find the end of the key-value pairs (first blank line)
	endOfPairs := bytes.Index(raw, []byte("\n\n"))
	if endOfPairs == -1 {
		// Maybe there's no message
		endOfPairs = len(raw)
	}

	// The rest is the message
	message = string(raw[endOfPairs+2:])

	pairs := bytes.Split(raw[:endOfPairs], []byte("\n"))
	var lastKey string

	for _, pair := range pairs {
		if len(pair) == 0 {
			continue
		}

		if pair[0] == ' ' { // Continuation line
			if lastKey == "" {
				return nil, "", errors.New("invalid kvlm: continuation line with no key")
			}
			// Append to the last value of the last key
			lastValueIndex := len(kvlm[lastKey]) - 1
			kvlm[lastKey][lastValueIndex] = kvlm[lastKey][lastValueIndex] + string(pair[1:])
		} else {
			space := bytes.Index(pair, []byte(" "))
			if space == -1 {
				return nil, "", errors.New("invalid kvlm: missing space")
			}
			key := string(pair[:space])
			value := string(pair[space+1:])
			kvlm[key] = append(kvlm[key], value)
			lastKey = key
		}
	}

	return kvlm, message, nil
}

// kvlmSerialize serializes a key-value map and a message back into bytes.
func kvlmSerialize(kvlm map[string][]string, message string) []byte {
	var b bytes.Buffer

	// Canonical order
	order := []string{"tree", "parent", "author", "committer", "gpgsig"}

	for _, key := range order {
		if values, ok := kvlm[key]; ok {
			for _, value := range values {
				// Handle multi-line values by prepending a space to subsequent lines
				valLines := strings.Split(value, "\n")
				b.WriteString(key)
				b.WriteString(" ")
				b.WriteString(valLines[0])
				b.WriteString("\n")
				for _, line := range valLines[1:] {
					b.WriteString(" ")
					b.WriteString(line)
					b.WriteString("\n")
				}
			}
		}
	}

	b.WriteString("\n")
	b.WriteString(message)

	return b.Bytes()
}

// GitCommit represents a commit object.
type GitCommit struct {
	kvlm    map[string][]string
	message string
}

func (c *GitCommit) Type() string {
	return "commit"
}

func (c *GitCommit) Deserialize(data []byte) error {
	var err error
	c.kvlm, c.message, err = kvlmParse(data)
	return err
}

func (c *GitCommit) Serialize() ([]byte, error) {
	return kvlmSerialize(c.kvlm, c.message), nil
}

// cmdLog is the handler for the log command.
func cmdLog(commitRef string) error {
	repo, err := repoFind(".", true)
	if err != nil {
		return err
	}

	fmt.Println("digraph gvcslog{")
	fmt.Println("  node[shape=rect]")

	sha, err := objectFind(repo, commitRef, "", true)
	if err != nil {
		return err
	}

	seen := make(map[string]bool)
	err = logGraphviz(repo, sha, seen)
	if err != nil {
		return err
	}

	fmt.Println("}")
	return nil
}

func logGraphviz(repo *GitRepository, sha string, seen map[string]bool) error {
	if seen[sha] {
		return nil
	}
	seen[sha] = true

	obj, err := objectRead(repo, sha)
	if err != nil {
		return err
	}
	commit, ok := obj.(*GitCommit)
	if !ok {
		return fmt.Errorf("object %s is not a commit", sha)
	}

	message := strings.TrimSpace(commit.message)
	// Escape backslashes and quotes for dot format
	message = strings.ReplaceAll(message, "\\", "\\\\")
	message = strings.ReplaceAll(message, "\"", "\\\"")
	if idx := strings.Index(message, "\n"); idx != -1 {
		message = message[:idx] // Keep only the first line
	}

	fmt.Printf("  c_%s [label=\"%s: %s\"]\n", sha, sha[0:7], message)

	if parents, ok := commit.kvlm["parent"]; ok {
		for _, p := range parents {
			p = strings.TrimSpace(p)
			fmt.Printf("  c_%s -> c_%s;\n", sha, p)
			if err := logGraphviz(repo, p, seen); err != nil {
				return err
			}
		}
	}
	return nil
}

// GitTreeLeaf represents a single entry in a tree object.
type GitTreeLeaf struct {
	Mode string
	Path string
	SHA  string // Stored as a hex string
}

// treeParse parses the raw data of a tree object.
func treeParse(raw []byte) ([]GitTreeLeaf, error) {
	var leaves []GitTreeLeaf
	pos := 0
	for pos < len(raw) {
		// Find the space after the mode
		space := bytes.IndexByte(raw[pos:], ' ')
		if space == -1 {
			return nil, errors.New("invalid tree object: missing space")
		}
		mode := string(raw[pos : pos+space])

		// Find the null terminator of the path
		null := bytes.IndexByte(raw[pos:], '\x00')
		if null == -1 {
			return nil, errors.New("invalid tree object: missing null terminator")
		}
		path := string(raw[pos+space+1 : pos+null])

		// Read the 20-byte SHA
		shaBytes := raw[pos+null+1 : pos+null+21]
		sha := hex.EncodeToString(shaBytes)

		leaves = append(leaves, GitTreeLeaf{Mode: mode, Path: path, SHA: sha})
		pos += null + 21
	}
	return leaves, nil
}

// treeSerialize serializes a GitTree object back to its byte representation.
func treeSerialize(tree *GitTree) ([]byte, error) {
	// Sort the items according to Git's rules
	sort.Slice(tree.Items, func(i, j int) bool {
		pathA := tree.Items[i].Path
		pathB := tree.Items[j].Path

		// Directories are sorted with a trailing slash
		isDirA := tree.Items[i].Mode == "040000"
		isDirB := tree.Items[j].Mode == "040000"

		if isDirA {
			pathA += "/"
		}
		if isDirB {
			pathB += "/"
		}

		return pathA < pathB
	})

	var b bytes.Buffer
	for _, item := range tree.Items {
		// Mode
		b.WriteString(item.Mode)
		b.WriteByte(' ')
		// Path
		b.WriteString(item.Path)
		b.WriteByte('\x00')
		// SHA
		shaBytes, err := hex.DecodeString(item.SHA)
		if err != nil {
			return nil, err
		}
		b.Write(shaBytes)
	}
	return b.Bytes(), nil
}

// GitTree represents a tree object.
type GitTree struct {
	Items []GitTreeLeaf
}

func (t *GitTree) Type() string {
	return "tree"
}

func (t *GitTree) Deserialize(data []byte) error {
	var err error
	t.Items, err = treeParse(data)
	return err
}

func (t *GitTree) Serialize() ([]byte, error) {
	return treeSerialize(t)
}
func cmdLsTree(ref string, recursive bool) error {
	repo, err := repoFind(".", true)
	if err != nil {
		return err
	}
	return lsTree(repo, ref, recursive, "")
}

func lsTree(repo *GitRepository, ref string, recursive bool, prefix string) error {
	sha, err := objectFind(repo, ref, "tree", true)
	if err != nil {
		return err
	}
	obj, err := objectRead(repo, sha)
	if err != nil {
		return err
	}
	tree, ok := obj.(*GitTree)
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
			if err := lsTree(repo, item.SHA, recursive, fullPath); err != nil {
				return err
			}
		} else {
			fmt.Printf("%s %s %s\t%s\n", item.Mode, objType, item.SHA, fullPath)
		}
	}
	return nil
}

func cmdCheckout(commitRef, path string) error {
	repo, err := repoFind(".", true)
	if err != nil {
		return err
	}

	sha, err := objectFind(repo, commitRef, "", true)
	if err != nil {
		return err
	}

	obj, err := objectRead(repo, sha)
	if err != nil {
		return err
	}

	// If the object is a commit, we grab its tree
	if commit, ok := obj.(*GitCommit); ok {
		treeSHA, ok := commit.kvlm["tree"]
		if !ok || len(treeSHA) == 0 {
			return errors.New("commit has no tree")
		}
		obj, err = objectRead(repo, treeSHA[0])
		if err != nil {
			return err
		}
	}

	tree, ok := obj.(*GitTree)
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

	return treeCheckout(repo, tree, path)
}

func treeCheckout(repo *GitRepository, tree *GitTree, path string) error {
	for _, item := range tree.Items {
		obj, err := objectRead(repo, item.SHA)
		if err != nil {
			return err
		}
		dest := filepath.Join(path, item.Path)

		switch o := obj.(type) {
		case *GitTree:
			if err := os.Mkdir(dest, 0755); err != nil {
				return err
			}
			if err := treeCheckout(repo, o, dest); err != nil {
				return err
			}
		case *GitBlob:
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

// refResolve follows a ref to its ultimate object SHA.
func refResolve(repo *GitRepository, ref string) (string, error) {
	path, err := repoFile(repo, false, ref)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(path)
	// Special case: In a new repo, HEAD points to 'refs/heads/master',
	// but that file doesn't exist yet. This is not an error.
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	content := strings.TrimSpace(string(data))

	if strings.HasPrefix(content, "ref: ") {
		return refResolve(repo, content[5:])
	}
	return content, nil
}

// refList recursively collects all refs in a given path.
func refList(repo *GitRepository, path string) (map[string]interface{}, error) {
	if path == "" {
		path = repoPath(repo, "refs")
	}

	ret := make(map[string]interface{})
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	// Sort entries for consistent output
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		can := filepath.Join(path, entry.Name())
		if entry.IsDir() {
			sub, err := refList(repo, can)
			if err != nil {
				return nil, err
			}
			ret[entry.Name()] = sub
		} else {
			// The ref path should be relative to the gitdir
			relPath, err := filepath.Rel(repo.gitdir, can)
			if err != nil {
				return nil, err
			}
			sha, err := refResolve(repo, relPath)
			if err != nil {
				return nil, err
			}
			ret[entry.Name()] = sha
		}
	}
	return ret, nil
}

func cmdShowRef() error {
	repo, err := repoFind(".", true)
	if err != nil {
		return err
	}
	refs, err := refList(repo, "")
	if err != nil {
		return err
	}
	showRef(refs, "refs", true)
	return nil
}

func showRef(refs map[string]interface{}, prefix string, withHash bool) {
	for k, v := range refs {
		fullPath := fmt.Sprintf("%s/%s", prefix, k)
		switch val := v.(type) {
		case string:
			if withHash {
				fmt.Printf("%s %s\n", val, fullPath)
			} else {
				fmt.Printf("%s\n", fullPath)
			}
		case map[string]interface{}:
			showRef(val, fullPath, withHash)
		}
	}
}

// GitTag represents a tag object.
type GitTag struct {
	kvlm    map[string][]string
	message string
}

func (t *GitTag) Type() string {
	return "tag"
}

func (t *GitTag) Deserialize(data []byte) error {
	var err error
	t.kvlm, t.message, err = kvlmParse(data)
	return err
}

func (t *GitTag) Serialize() ([]byte, error) {
	// The canonical order for tags is different
	// order := []string{"object", "type", "tag", "tagger", "gpgsig"}
	// We need a slightly modified serializer for tags if the order matters strictly.
	// For simplicity here, we'll reuse the commit serializer's logic.
	return kvlmSerialize(t.kvlm, t.message), nil
}

func cmdTag(name, object string, annotated bool) error {
	repo, err := repoFind(".", true)
	if err != nil {
		return err
	}

	if name != "" {
		return tagCreate(repo, name, object, annotated)
	}

	// List tags
	refs, err := refList(repo, repoPath(repo, "refs", "tags"))
	if err != nil {
		return err
	}
	showRef(refs, "tags", false)
	return nil
}
func tagCreate(repo *GitRepository, name, ref string, createTagObject bool) error {
	sha, err := objectFind(repo, ref, "", true)
	if err != nil {
		return err
	}

	if createTagObject {
		// Create tag object
		tag := &GitTag{
			kvlm: make(map[string][]string),
		}
		tag.kvlm["object"] = []string{sha}
		tag.kvlm["type"] = []string{"commit"} // Assuming we're tagging commits
		tag.kvlm["tag"] = []string{name}
		// In a real implementation, you'd get the user's name from config
		tag.kvlm["tagger"] = []string{"gvcs <gvcs@example.com>"}
		tag.message = "A tag generated by gvcs!\n"

		tagSHA, err := objectWrite(tag, repo)
		if err != nil {
			return err
		}
		// Create reference to the tag object
		return refCreate(repo, "refs/tags/"+name, tagSHA)
	}

	// Create lightweight tag (just a ref)
	return refCreate(repo, "refs/tags/"+name, sha)
}

func refCreate(repo *GitRepository, refName, sha string) error {
	path, err := repoFile(repo, true, refName)
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(sha+"\n"), 0644)
}

// objectResolve resolves a name to a list of candidate object hashes.
func objectResolve(repo *GitRepository, name string) ([]string, error) {
	if name == "" {
		return nil, nil
	}
	if name == "HEAD" {
		sha, err := refResolve(repo, "HEAD")
		if err != nil {
			return nil, err
		}
		if sha == "" {
			return nil, nil
		}
		return []string{sha}, nil
	}

	var candidates []string
	hashRE := regexp.MustCompile(`^[0-9a-fA-F]{4,40}$`)

	// Is it a hash?
	if hashRE.MatchString(name) {
		name = strings.ToLower(name)
		prefix := name[0:2]
		path := repoPath(repo, "objects", prefix)
		if _, err := os.Stat(path); err == nil {
			rem := name[2:]
			files, err := os.ReadDir(path)
			if err != nil {
				return nil, err
			}
			for _, f := range files {
				if strings.HasPrefix(f.Name(), rem) {
					candidates = append(candidates, prefix+f.Name())
				}
			}
		}
	}

	// Try for references
	for _, refPath := range []string{"refs/tags/", "refs/heads/"} {
		if sha, err := refResolve(repo, refPath+name); err == nil && sha != "" {
			candidates = append(candidates, sha)
		}
	}

	// Remove duplicates
	uniqueCandidates := make(map[string]bool)
	var result []string
	for _, c := range candidates {
		if !uniqueCandidates[c] {
			uniqueCandidates[c] = true
			result = append(result, c)
		}
	}

	return result, nil
}

func cmdRevParse(name, objType string) error {
	repo, err := repoFind(".", true)
	if err != nil {
		return err
	}
	sha, err := objectFind(repo, name, objType, true)
	if err != nil {
		return err
	}
	fmt.Println(sha)
	return nil
}

// GitIndexEntry represents a single entry in the Git index.
type GitIndexEntry struct {
	CTime [2]uint32 // ctime seconds and nanoseconds
	MTime [2]uint32 // mtime seconds and nanoseconds
	Dev   uint32
	Ino   uint32
	Mode  uint32 // file mode
	UID   uint32
	GID   uint32
	FSize uint32
	SHA   string // hex SHA
	Flags uint16
	Name  string
}

// GitIndex represents the Git index file.
type GitIndex struct {
	Version uint32
	Entries []*GitIndexEntry
}

// indexRead reads and parses the index file from the repository.
func indexRead(repo *GitRepository) (*GitIndex, error) {
	indexFile := repoPath(repo, "index")

	// New repositories have no index
	data, err := os.ReadFile(indexFile)
	if os.IsNotExist(err) {
		return &GitIndex{Version: 2, Entries: []*GitIndexEntry{}}, nil
	}
	if err != nil {
		return nil, err
	}

	// Header
	if string(data[0:4]) != "DIRC" {
		return nil, errors.New("invalid index signature")
	}
	version := binary.BigEndian.Uint32(data[4:8])
	if version != 2 {
		return nil, fmt.Errorf("gvcs only supports index file version 2, got %d", version)
	}
	count := binary.BigEndian.Uint32(data[8:12])

	index := &GitIndex{Version: version}
	index.Entries = make([]*GitIndexEntry, 0, count)

	// Entries
	pos := 12
	for i := 0; i < int(count); i++ {
		entry := &GitIndexEntry{}
		entry.CTime[0] = binary.BigEndian.Uint32(data[pos : pos+4])
		entry.CTime[1] = binary.BigEndian.Uint32(data[pos+4 : pos+8])
		entry.MTime[0] = binary.BigEndian.Uint32(data[pos+8 : pos+12])
		entry.MTime[1] = binary.BigEndian.Uint32(data[pos+12 : pos+16])
		entry.Dev = binary.BigEndian.Uint32(data[pos+16 : pos+20])
		entry.Ino = binary.BigEndian.Uint32(data[pos+20 : pos+24])
		entry.Mode = binary.BigEndian.Uint32(data[pos+24 : pos+28])
		entry.UID = binary.BigEndian.Uint32(data[pos+28 : pos+32])
		entry.GID = binary.BigEndian.Uint32(data[pos+32 : pos+36])
		entry.FSize = binary.BigEndian.Uint32(data[pos+36 : pos+40])

		shaBytes := data[pos+40 : pos+60]
		entry.SHA = hex.EncodeToString(shaBytes)

		entry.Flags = binary.BigEndian.Uint16(data[pos+60 : pos+62])

		pos += 62

		// Read file name until null terminator
		nameEnd := bytes.IndexByte(data[pos:], '\x00')
		if nameEnd == -1 {
			return nil, errors.New("invalid index entry: missing null terminator in name")
		}
		entry.Name = string(data[pos : pos+nameEnd])
		pos += nameEnd + 1 // pos is now at the start of padding

		// FIX: Calculate padding required to reach the next 8-byte boundary
		// Total bytes for fixed metadata (62) + name + null terminator (1)

		totalNonPaddingBytes := uint32(62 + nameEnd + 1)
		padLen := (8 - (totalNonPaddingBytes % 8)) % 8

		pos += int(padLen) // Advance pos by the precise padding length

		index.Entries = append(index.Entries, entry)
	}

	return index, nil
}

func cmdLsFiles(verbose bool) error {
	repo, err := repoFind(".", true)
	if err != nil {
		return err
	}
	index, err := indexRead(repo)
	if err != nil {
		return err
	}
	if verbose {
		fmt.Printf("Index file format v%d, containing %d entries.\n", index.Version, len(index.Entries))
	}

	for _, e := range index.Entries {
		fmt.Println(e.Name)
		if verbose {
			fmt.Printf("  Mode: %o\n", e.Mode)
			fmt.Printf("  SHA: %s\n", e.SHA)
			fmt.Printf("  Size: %d\n", e.FSize)
			fmt.Printf("  ctime: %s\n", time.Unix(int64(e.CTime[0]), int64(e.CTime[1])))
			fmt.Printf("  mtime: %s\n", time.Unix(int64(e.MTime[0]), int64(e.MTime[1])))
			// ... add more details if you like
		}
	}
	return nil
}

func cmdStatus() error {
	repo, err := repoFind(".", true)
	if err != nil {
		return err
	}
	index, err := indexRead(repo)
	if err != nil {
		return err
	}

	// Part 1: Print current branch
	if err := statusPrintBranch(repo); err != nil {
		return err
	}

	// Part 2: Compare HEAD to index
	if err := statusHeadIndex(repo, index); err != nil {
		return err
	}

	fmt.Println()

	// Part 3: Compare index to worktree
	if err := statusIndexWorktree(repo, index); err != nil {
		return err
	}

	return nil
}

// branchGetActive reads HEAD to find the current active branch.
func branchGetActive(repo *GitRepository) (string, bool, error) {
	headFile := repoPath(repo, "HEAD")
	data, err := os.ReadFile(headFile)
	if err != nil {
		return "", false, err
	}
	content := strings.TrimSpace(string(data))
	if strings.HasPrefix(content, "ref: refs/heads/") {
		return content[16:], false, nil // false means not detached
	}
	return content, true, nil // true means detached HEAD
}

func statusPrintBranch(repo *GitRepository) error {
	branch, detached, err := branchGetActive(repo)
	if err != nil {
		return err
	}
	if detached {
		fmt.Printf("HEAD detached at %s\n", branch)
	} else {
		fmt.Printf("On branch %s\n", branch)
	}
	return nil
}

// treeToMap recursively reads a tree and flattens it into a map.
func treeToMap(repo *GitRepository, ref, prefix string) (map[string]string, error) {
	ret := make(map[string]string)
	treeSHA, err := objectFind(repo, ref, "tree", true)
	if err != nil {
		return nil, err
	}
	obj, err := objectRead(repo, treeSHA)
	if err != nil {
		return nil, err
	}
	tree := obj.(*GitTree)

	for _, leaf := range tree.Items {
		fullPath := filepath.Join(prefix, leaf.Path)
		if leaf.Mode == "040000" { // is a subtree
			subMap, err := treeToMap(repo, leaf.SHA, fullPath)
			if err != nil {
				return nil, err
			}
			for k, v := range subMap {
				ret[k] = v
			}
		} else { // is a blob
			ret[fullPath] = leaf.SHA
		}
	}
	return ret, nil
}

func statusHeadIndex(repo *GitRepository, index *GitIndex) error {
	fmt.Println("Changes to be committed:")

	headMap, err := treeToMap(repo, "HEAD", "")
	if err != nil {
		// This can happen in a new repo with no commits
		if strings.Contains(err.Error(), "no such reference") {
			headMap = make(map[string]string)
		} else {
			return err
		}
	}

	indexMap := make(map[string]string)
	for _, entry := range index.Entries {
		indexMap[entry.Name] = entry.SHA
	}

	// Compare index to head
	for path, sha := range indexMap {
		headSHA, ok := headMap[path]
		if !ok {
			fmt.Printf("  added:    %s\n", path)
		} else if headSHA != sha {
			fmt.Printf("  modified: %s\n", path)
		}
	}

	// Find deleted files
	for path := range headMap {
		if _, ok := indexMap[path]; !ok {
			fmt.Printf("  deleted:  %s\n", path)
		}
	}
	return nil
}

func statusIndexWorktree(repo *GitRepository, index *GitIndex) error {
	fmt.Println("Changes not staged for commit:")

	worktreeFiles := make(map[string]bool)
	err := filepath.Walk(repo.worktree, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Ignore .git directory and other directories
		if strings.HasPrefix(path, repo.gitdir) || info.IsDir() {
			return nil
		}
		relPath, _ := filepath.Rel(repo.worktree, path)
		worktreeFiles[relPath] = true
		return nil
	})
	if err != nil {
		return err
	}

	indexMap := make(map[string]*GitIndexEntry)
	for _, e := range index.Entries {
		indexMap[e.Name] = e
	}

	// Check for modified and deleted files
	for _, entry := range index.Entries {
		fullPath := filepath.Join(repo.worktree, entry.Name)
		stat, err := os.Stat(fullPath)
		if os.IsNotExist(err) {
			fmt.Printf("  deleted:  %s\n", entry.Name)
			continue
		}

		// Compare metadata. A simple mtime check is a good start.
		mtime_s := uint32(stat.ModTime().Unix())
		if mtime_s != entry.MTime[0] {
			// Metadata differs, do a full content check
			f, err := os.Open(fullPath)
			if err != nil {
				return err
			}

			// We pass nil for repo so it doesn't write the object
			newSHA, err := objectHash(f, "blob", nil)
			f.Close()
			if err != nil {
				return err
			}

			if newSHA != entry.SHA {
				fmt.Printf("  modified: %s\n", entry.Name)
			}
		}
	}

	rules, err := gitignoreRead(repo)
	if err != nil {
		return err
	}

	fmt.Println("\nUntracked files:")
	for path := range worktreeFiles {
		if _, inIndex := indexMap[path]; !inIndex {
			if !checkIgnore(rules, path) {
				fmt.Printf("  %s\n", path)
			}
		}
	}

	return nil
}

func cmdCheckIgnore(paths []string) error {
	repo, err := repoFind(".", true)
	if err != nil {
		return err
	}
	rules, err := gitignoreRead(repo)
	if err != nil {
		return err
	}
	for _, path := range paths {
		if checkIgnore(rules, path) {
			fmt.Println(path)
		}
	}
	return nil
}

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

func gitignoreRead(repo *GitRepository) (*GitIgnore, error) {
	ignore := &GitIgnore{
		Scoped: make(map[string][]gitignoreRule),
	}

	// Read repo-specific .git/info/exclude
	excludeFile := repoPath(repo, "info/exclude")
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
	index, err := indexRead(repo)
	if err != nil {
		return nil, err
	}

	for _, entry := range index.Entries {
		if filepath.Base(entry.Name) == ".gitignore" {
			obj, err := objectRead(repo, entry.SHA)
			if err != nil {
				return nil, err
			}
			blob := obj.(*GitBlob)
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

// checkIgnore checks if a path should be ignored.
func checkIgnore(rules *GitIgnore, path string) bool {
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

func indexWrite(repo *GitRepository, index *GitIndex) error {
	// Build index in-memory first so we can compute checksum.
	var buf bytes.Buffer

	// HEADER
	buf.Write([]byte("DIRC"))
	// version
	if err := binary.Write(&buf, binary.BigEndian, index.Version); err != nil {
		return err
	}
	// number of entries
	if err := binary.Write(&buf, binary.BigEndian, uint32(len(index.Entries))); err != nil {
		return err
	}

	// ENTRIES
	for _, e := range index.Entries {
		// ctime (2 x uint32)
		if err := binary.Write(&buf, binary.BigEndian, e.CTime); err != nil {
			return err
		}
		// mtime (2 x uint32)
		if err := binary.Write(&buf, binary.BigEndian, e.MTime); err != nil {
			return err
		}
		// dev, ino, mode, uid, gid, fsize
		if err := binary.Write(&buf, binary.BigEndian, e.Dev); err != nil {
			return err
		}
		if err := binary.Write(&buf, binary.BigEndian, e.Ino); err != nil {
			return err
		}
		if err := binary.Write(&buf, binary.BigEndian, e.Mode); err != nil {
			return err
		}
		if err := binary.Write(&buf, binary.BigEndian, e.UID); err != nil {
			return err
		}
		if err := binary.Write(&buf, binary.BigEndian, e.GID); err != nil {
			return err
		}
		if err := binary.Write(&buf, binary.BigEndian, e.FSize); err != nil {
			return err
		}

		// SHA (20 bytes)
		shaBytes, err := hex.DecodeString(e.SHA)
		if err != nil {
			return err
		}
		if len(shaBytes) != 20 {
			return fmt.Errorf("invalid sha length for entry %s: %d", e.Name, len(shaBytes))
		}
		if _, err := buf.Write(shaBytes); err != nil {
			return err
		}

		// Flags: preserve top bits (assume-valid / stage) and encode name length in low 12 bits
		nameBytes := []byte(e.Name)
		nameLen := len(nameBytes)
		if nameLen >= 0xFFF {
			nameLen = 0xFFF
		}
		// Preserve top 4 bits of existing Flags (bits 12-15) and set low 12 bits to nameLen.
		flags := (e.Flags & 0xF000) | uint16(nameLen)
		if err := binary.Write(&buf, binary.BigEndian, flags); err != nil {
			return err
		}

		// Name + null terminator
		if _, err := buf.Write(nameBytes); err != nil {
			return err
		}
		if err := buf.WriteByte(0); err != nil {
			return err
		}

		// Padding: each entry (62 + name + 1) must be aligned to an 8-byte boundary.
		entryLen := 62 + nameLen + 1
		padLen := (8 - (entryLen % 8)) % 8
		if padLen > 0 {
			if _, err := buf.Write(make([]byte, padLen)); err != nil {
				return err
			}
		}
	}

	// Compute checksum for the whole content and append it.
	content := buf.Bytes()
	sum := sha1.Sum(content)
	final := append(content, sum[:]...)

	// Atomically write the final index file.
	if err := os.WriteFile(repoPath(repo, "index"), final, 0644); err != nil {
		return err
	}

	return nil
}

func cmdRm(paths []string) error {
	repo, err := repoFind(".", true)
	if err != nil {
		return err
	}
	return rm(repo, paths)
}

func rm(repo *GitRepository, paths []string) error {
	index, err := indexRead(repo)
	if err != nil {
		return err
	}

	// Create a set of paths to remove for efficient lookup
	toRemove := make(map[string]bool)
	for _, p := range paths {
		toRemove[p] = true
	}

	var keptEntries []*GitIndexEntry
	for _, entry := range index.Entries {
		if !toRemove[entry.Name] {
			keptEntries = append(keptEntries, entry)
		}
	}

	index.Entries = keptEntries
	if err := indexWrite(repo, index); err != nil {
		return err
	}

	// Physically delete the files
	for _, path := range paths {
		if err := os.Remove(filepath.Join(repo.worktree, path)); err != nil {
			// Don't fail if file is already gone
			if !os.IsNotExist(err) {
				return err
			}
		}
	}

	return nil
}

func cmdAdd(paths []string) error {
	repo, err := repoFind(".", true)
	if err != nil {
		return err
	}
	index, err := indexRead(repo)
	if err != nil {
		return err
	}

	// Create a map for quick access to existing entries
	indexMap := make(map[string]*GitIndexEntry)
	var keptEntries []*GitIndexEntry
	for _, entry := range index.Entries {
		indexMap[entry.Name] = entry
		// Keep all entries that are *not* being added
		isAdding := false
		for _, p := range paths {
			if entry.Name == p {
				isAdding = true
				break
			}
		}
		if !isAdding {
			keptEntries = append(keptEntries, entry)
		}
	}

	// Process files to add
	for _, path := range paths {
		fullPath := filepath.Join(repo.worktree, path)
		f, err := os.Open(fullPath)
		if err != nil {
			return err
		}

		sha, err := objectHash(f, "blob", repo)
		f.Close()
		if err != nil {
			return err
		}

		stat, err := os.Stat(fullPath)
		if err != nil {
			return err
		}

		// Mode for regular file is 100644
		mode := uint32(0100644)

		relPath, err := filepath.Rel(repo.worktree, fullPath)
		if err != nil {
			return err
		}

		entry := &GitIndexEntry{
			CTime: [2]uint32{uint32(stat.ModTime().Unix()), uint32(stat.ModTime().Nanosecond())},
			MTime: [2]uint32{uint32(stat.ModTime().Unix()), uint32(stat.ModTime().Nanosecond())},
			Dev:   0,
			Ino:   0,
			Mode:  mode,
			UID:   0,
			GID:   0,
			FSize: uint32(stat.Size()),
			SHA:   sha,
			Name:  relPath,
		}
		keptEntries = append(keptEntries, entry)
	}

	// Sort entries by name, as Git requires
	sort.Slice(keptEntries, func(i, j int) bool {
		return keptEntries[i].Name < keptEntries[j].Name
	})

	index.Entries = keptEntries
	return indexWrite(repo, index)
}

// treeFromIndex builds a tree object from the current index.
func treeFromIndex(repo *GitRepository, index *GitIndex) (string, error) {
	// Group entries by directory
	dirEntries := make(map[string][]GitTreeLeaf)
	for _, entry := range index.Entries {
		dir := filepath.Dir(entry.Name)
		if dir == "." {
			dir = ""
		}
		leaf := GitTreeLeaf{
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
		tree := &GitTree{Items: dirEntries[dir]}

		// Add subtrees that we've already built
		for subDir, sha := range treeSHAs {
			if filepath.Dir(subDir) == dir {
				leaf := GitTreeLeaf{
					Mode: "040000",
					Path: filepath.Base(subDir),
					SHA:  sha,
				}
				tree.Items = append(tree.Items, leaf)
			}
		}

		sha, err := objectWrite(tree, repo)
		if err != nil {
			return "", err
		}
		treeSHAs[dir] = sha
	}
	return treeSHAs[""], nil
}

func commitCreate(repo *GitRepository, tree, parent, message string) (string, error) {
	commit := &GitCommit{
		kvlm: make(map[string][]string),
	}
	commit.kvlm["tree"] = []string{tree}
	if parent != "" {
		commit.kvlm["parent"] = []string{parent}
	}

	// Get author name and email from git config (mirror libwyag - no fallbacks)
	var authorName, authorEmail string

	// Try repo config first
	if name, email := parseUserConfig(repo.gitdir, "config"); name != "" && email != "" {
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

	commit.kvlm["author"] = []string{fmt.Sprintf("%s %d %s", author, time.Now().Unix(), timestamp[len(timestamp)-5:])}
	commit.kvlm["committer"] = commit.kvlm["author"]
	commit.message = message

	return objectWrite(commit, repo)
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

func cmdCommit(message string) error {
	repo, err := repoFind(".", true)
	if err != nil {
		return err
	}
	index, err := indexRead(repo)
	if err != nil {
		return err
	}

	// Create trees
	treeSHA, err := treeFromIndex(repo, index)
	if err != nil {
		return err
	}

	// Get parent commit (mirror libwyag behavior)
	parent, err := objectFind(repo, "HEAD", "", true)
	if err != nil {
		// For initial commit, parent is empty
		parent = ""
	}

	// Trim message and add newline (mirror libwyag)
	message = strings.TrimSpace(message) + "\n"

	// Create the commit object
	commitSHA, err := commitCreate(repo, treeSHA, parent, message)
	if err != nil {
		return err
	}

	// Update HEAD
	branch, detached, err := branchGetActive(repo)
	if err != nil {
		return err
	}
	var refToUpdate string
	if detached {
		refToUpdate = "HEAD"
	} else {
		refToUpdate = "refs/heads/" + branch
	}

	return refCreate(repo, refToUpdate, commitSHA)
}
