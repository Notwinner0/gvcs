package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Notwinner0/gvcs/internal/commands"
	"github.com/Notwinner0/gvcs/internal/repo"

	"github.com/akamensky/argparse"
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
		_, err := repo.RepoCreate(*initPath)
		if err != nil {
			log.Fatalf("Error creating repository: %v", err)
		}
		fmt.Printf("Initialized empty gvcs repository in %s\n", *initPath)
		break
	case catFileCmd.Happened():
		err := commands.CmdCatFile(*catFileType, *catFileObject)
		if err != nil {
			log.Fatalf("Error cat-file: %v", err)
		}
		break
	case hashObjectCmd.Happened():
		err := commands.CmdHashObject(*hashObjectWrite, *hashObjectType, *hashObjectPath)
		if err != nil {
			log.Fatalf("Error hash-object: %v", err)
		}
		break
	case logCmd.Happened():
		err := commands.CmdLog(*logCommit)
		if err != nil {
			log.Fatalf("Error log: %v", err)
		}
		break
	case lsTreeCmd.Happened():
		err := commands.CmdLsTree(*lsTreeObject, *lsTreeRecursive)
		if err != nil {
			log.Fatalf("Error ls-tree: %v", err)
		}
		break
	case checkoutCmd.Happened():
		err := commands.CmdCheckout(*checkoutCommit, *checkoutPath)
		if err != nil {
			log.Fatalf("Error checkout: %v", err)
		}
		break
	case showRefCmd.Happened():
		err := commands.CmdShowRef()
		if err != nil {
			log.Fatalf("Error show-ref: %v", err)
		}
		break
	case tagCmd.Happened():
		err := commands.CmdTag(*tagName, *tagObject, *tagAnnotated)
		if err != nil {
			log.Fatalf("Error tag: %v", err)
		}
		break
	case revParseCmd.Happened():
		err := commands.CmdRevParse(*revParseName, *revParseType)
		if err != nil {
			log.Fatalf("Error rev-parse: %v", err)
		}
		break
	case lsFilesCmd.Happened():
		err := commands.CmdLsFiles(*lsFilesVerbose)
		if err != nil {
			log.Fatalf("Error ls-files: %v", err)
		}
		break
	case statusCmd.Happened():
		err := commands.CmdStatus()
		if err != nil {
			log.Fatalf("Error status: %v", err)
		}
		break
	case checkIgnoreCmd.Happened():
		err := commands.CmdCheckIgnore(*checkIgnorePaths)
		if err != nil {
			log.Fatalf("Error check-ignore: %v", err)
		}
		break
	case rmCmd.Happened():
		err := commands.CmdRm(*rmPaths)
		if err != nil {
			log.Fatalf("Error rm: %v", err)
		}
		break
	case addCmd.Happened():
		err := commands.CmdAdd(*addPaths)
		if err != nil {
			log.Fatalf("Error add: %v", err)
		}
		break
	case commitCmd.Happened():
		err := commands.CmdCommit(*commitMessage)
		if err != nil {
			log.Fatalf("Error commit: %v", err)
		}
		break
	// ... other command cases will be here
	default:
		log.Fatal("Bad command.")
	}
}
