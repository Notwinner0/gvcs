//go:build windows

package repo

import "syscall"

func hideGitDir(path string) {
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err == nil {
		_ = syscall.SetFileAttributes(pathPtr, syscall.FILE_ATTRIBUTE_HIDDEN)
	}
}
