<div align="center" markdown="1">
  <h2>gvcs - a Go version control system</h2>
  <a href="https://github.com/Notwinner0/gvcs/actions"><img src="https://github.com/Notwinner0/gvcs/actions/workflows/go.yml/badge.svg?branch=main" alt="Build Status"></a>
  <a href="http://github.com/Notwinner0/gvcs/releases"><img src="https://img.shields.io/github/v/tag/Notwinner0/gvcs" alt="Version"></a>
  <a href="https://github.com/Notwinner0/gvcs?tab=MIT-1-ov-file#readme"><img src="https://img.shields.io/github/license/Notwinner0/gvcs" alt="License"></a>
  <a href="https://github.com/Notwinner0/gvcs/graphs/contributors"><img src="https://img.shields.io/github/contributors/Notwinner0/gvcs" alt="Contributors"></a>
  <a href="https://github.com/Notwinner0/gvcs/stargazers"><img src="https://img.shields.io/github/stars/Notwinner0/gvcs?style=flat" alt="Stars"></a>
  <a href="https://codecov.io/github/Notwinner0/gvcs" > 
 <img src="https://codecov.io/github/Notwinner0/gvcs/graph/badge.svg?token=QZOTUY8TW9"/> 
 </a>
</div>

---

gvcs is a version control system written in Go, implementing core Git-like functionality.

It's a learning project that demonstrates how version control systems work under the hood, providing commands for repository management, object storage, commits, branches, and more.

---

**⚠️ Warning: This project is incomplete and experimental**

This is not even a minimum viable product (MVP). There are no tests, and there's no guarantee that something won't go wrong. Use at your own risk, especially in production environments.

---

Highlights
----------

- **Educational** — Learn how Git works internally by examining the Go implementation
- **Portable** — Single binary written in Go for easy distribution
- **Git-compatible** — Implements core Git commands and data structures
- **Simple** — Focused on core VCS concepts without the complexity of Git's full feature set

Table of Contents
-----------------

<!-- vim-markdown-toc GFM -->

* [Installation](#installation)
    * [Using go install](#using-go-install)
    * [Building from source](#building-from-source)
* [Getting Started](#getting-started)
* [Usage](#usage)
    * [Initializing a repository](#initializing-a-repository)
    * [Adding files](#adding-files)
    * [Committing changes](#committing-changes)
    * [Viewing history](#viewing-history)
    * [Working with objects](#working-with-objects)
    * [Checking status](#checking-status)
    * [Listing files](#listing-files)
    * [Listing tree contents](#listing-tree-contents)
    * [Listing references](#listing-references)
    * [Creating and listing tags](#creating-and-listing-tags)
    * [Parsing revisions](#parsing-revisions)
    * [Removing files](#removing-files)
    * [Checking ignore rules](#checking-ignore-rules)
* [Commands](#commands)
* [Examples](#examples)
* [Contributing](#contributing)
* [License](#license)

<!-- vim-markdown-toc -->

Installation
------------

### Using go install

You can install gvcs using Go's package manager:

```sh
go install github.com/Notwinner0/gvcs/cmd/gvcs@latest
```

Make sure your `$GOPATH/bin` is in your `$PATH`.

### Building from source

Clone the repository and build:

```sh
git clone https://github.com/Notwinner0/gvcs.git
cd gvcs
go build -o gvcs ./cmd/gvcs
```

The binary will be created in the current directory.

Getting Started
---------------

Initialize a new gvcs repository:

```sh
mkdir my-project
cd my-project
gvcs init
```

Add a file and commit:

```sh
echo "Hello, World!" > hello.txt
gvcs add hello.txt
gvcs commit -m "Initial commit"
```

View the commit history:

```sh
gvcs log
```

Usage
-----

### Initializing a repository

```sh
gvcs init [-p path]
```

Creates a new gvcs repository in the specified path (defaults to current directory).

### Adding files

```sh
gvcs add -f <files...>
```

Adds file contents to the index.

### Committing changes

```sh
gvcs commit -m "commit message"
```

Records changes to the repository with the given message.

### Viewing history

```sh
gvcs log [-c commit]
```

Displays the history of a given commit (defaults to HEAD).

### Working with objects

```sh
gvcs cat-file -t <type> -o <object>
```

Displays the content of a repository object.

```sh
gvcs hash-object [-w] [-t <type>] <file>
```

Computes the object ID for a file and optionally writes it to the database.

### Checking status

```sh
gvcs status
```

Shows the working tree status.

### Listing files

```sh
gvcs ls-files [-v]
```

Lists all the staged files (use -v for verbose output).

### Listing tree contents

```sh
gvcs ls-tree [-r] <tree-ish>
```

Pretty-prints a tree object (use -r to recurse into sub-trees).

### Listing references

```sh
gvcs show-ref
```

Lists references.

### Creating and listing tags

```sh
gvcs tag [-a] [-n <name>] [-o <object>]
```

Lists and creates tags (use -a for annotated tags).

### Parsing revisions

```sh
gvcs rev-parse [-t <type>] <name>
```

Parses revision (or other objects) identifiers.

### Removing files

```sh
gvcs rm <files...>
```

Removes files from the working tree and the index.

### Checking ignore rules

```sh
gvcs check-ignore <paths...>
```

Checks path(s) against ignore rules.

Commands
--------

gvcs supports the following commands:

- `init` — Initialize a new, empty repository
- `add` — Add file contents to the index
- `commit` — Record changes to the repository
- `log` — Display history of a given commit
- `status` — Show the working tree status
- `ls-files` — List all the staged files
- `ls-tree` — Pretty-print a tree object
- `cat-file` — Provide content of repository objects
- `hash-object` — Compute object ID and optionally creates a blob from a file
- `checkout` — Checkout a commit inside of a directory
- `show-ref` — List references
- `tag` — List and create tags
- `rev-parse` — Parse revision (or other objects) identifiers
- `rm` — Remove files from the working tree and the index
- `check-ignore` — Check path(s) against ignore rules

For detailed usage of each command, run `gvcs <command> --help`.

Examples
--------

### Basic workflow

```sh
# Initialize repository
gvcs init

# Create and add a file
echo "content" > file.txt
gvcs add file.txt

# Commit the changes
gvcs commit -m "Add file.txt"

# View history
gvcs log
```

### Working with objects

```sh
# Compute hash of a file
gvcs hash-object file.txt

# Store the file in the database
gvcs hash-object -w file.txt

# View the stored object
gvcs cat-file -t blob -o <hash>
```

### Branching and tagging

```sh
# Create a tag
gvcs tag -a -n v1.0 -o HEAD

# List tags
gvcs tag

# Checkout a specific commit
gvcs checkout <commit-hash> <directory>
```

Contributing
------------

Contributions are welcome! Please feel free to submit issues and pull requests.

To build and test:

```sh
go build ./...
go test ./...
```

License
-------

The MIT License (MIT)

Copyright (c) 2025 Notwinner

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
