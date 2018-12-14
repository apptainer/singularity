// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package fs

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// IsFile check if name component is regular file
func IsFile(name string) bool {
	info, err := os.Stat(name)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}

// IsDir check if name component is a directory
func IsDir(name string) bool {
	info, err := os.Stat(name)
	if err != nil {
		return false
	}
	return info.Mode().IsDir()
}

// IsLink check if name component is a symlink
func IsLink(name string) bool {
	info, err := os.Lstat(name)
	if err != nil {
		return false
	}
	return (info.Mode()&os.ModeSymlink != 0)
}

// IsOwner check if name component is owned by user identified with uid
func IsOwner(name string, uid uint32) bool {
	info, err := os.Stat(name)
	if err != nil {
		return false
	}
	return (info.Sys().(*syscall.Stat_t).Uid == uid)
}

// IsExec check if name component has executable bit permission set
func IsExec(name string) bool {
	info, err := os.Stat(name)
	if err != nil {
		return false
	}
	return (info.Sys().(*syscall.Stat_t).Mode&syscall.S_IXUSR != 0)
}

// IsSuid check if name component has setuid bit permission set
func IsSuid(name string) bool {
	info, err := os.Stat(name)
	if err != nil {
		return false
	}
	return (info.Sys().(*syscall.Stat_t).Mode&syscall.S_ISUID != 0)
}

// MkdirAll creates a directory and parents if it doesn't exist with
// mode after umask reset
func MkdirAll(path string, mode os.FileMode) error {
	oldmask := syscall.Umask(0)
	defer syscall.Umask(oldmask)

	return os.MkdirAll(path, mode)
}

// RootDir returns the root directory of path (rootdir of /my/path is /my).
// Returns "." if path is empty
func RootDir(path string) string {
	if path == "" {
		return "."
	}

	p := filepath.Clean(path)
	iter := filepath.Dir(p)
	for iter != "/" && iter != "." {
		p = iter
		iter = filepath.Dir(p)
	}

	return p
}

// EvalRelative evaluates symlinks in path relative to root path. This
// function doesn't return error but always returns an evaluated path
func EvalRelative(path string, root string) string {
	splitted := strings.Split(filepath.Clean(path), string(os.PathSeparator))
	dest := string(os.PathSeparator)

	for i := 1; i < len(splitted); i++ {
		s := splitted[i : i+1][0]
		dest = filepath.Join(dest, s)

		if s != "" {
			rootDestPath := filepath.Join(root, dest)
			for {
				target, err := filepath.EvalSymlinks(rootDestPath)
				if err != nil {
					break
				}
				if !strings.HasPrefix(target, root) {
					rootDestPath = filepath.Join(root, target)
					continue
				}
				dest = strings.Replace(target, root, "", 1)
				break
			}
		}
	}

	return dest
}
