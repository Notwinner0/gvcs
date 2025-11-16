//go:build !windows

package repo

func hideGitDir(path string) {
	// no-op on non-Windows
}
