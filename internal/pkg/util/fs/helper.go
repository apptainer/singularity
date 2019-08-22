// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package fs

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
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

// Mkdir creates a directory if it doesn't exist with
// mode after umask reset
func Mkdir(path string, mode os.FileMode) error {
	oldmask := syscall.Umask(0)
	defer syscall.Umask(oldmask)

	return os.Mkdir(path, mode)
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

// Touch behaves like touch command.
func Touch(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	f.Close()
	return nil
}

// MakeTmpDir creates a temporary directory with provided mode
// in os.TempDir if basedir is "". This function assumes that
// basedir exists, so it's the caller's responsibility to create
// it before calling it.
func MakeTmpDir(basedir, pattern string, mode os.FileMode) (string, error) {
	name, err := ioutil.TempDir(basedir, pattern)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %s", err)
	}
	if err := os.Chmod(name, mode); err != nil {
		return "", fmt.Errorf("failed to change permission of %s: %s", name, err)
	}
	return name, nil
}

// MakeTmpFile creates a temporary file with provided mode
// in os.TempDir if basedir is "". This function assumes that
// basedir exists, so it's the caller's responsibility to create
// it before calling it.
func MakeTmpFile(basedir, pattern string, mode os.FileMode) (*os.File, error) {
	f, err := ioutil.TempFile(basedir, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file: %s", err)
	}
	if err := f.Chmod(mode); err != nil {
		return nil, fmt.Errorf("failed to change permission of %s: %s", f.Name(), err)
	}
	return f, nil
}

// FileExists simply checks if a path exists.
func FileExists(path string) (bool, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

// CopyFile copies file to the provided location. To honor umask
// correctly, the to file must not exist.
func CopyFile(from, to string, mode os.FileMode) error {
	exist, err := FileExists(to)
	if err != nil {
		return err
	}
	if exist {
		return fmt.Errorf("file %s already exists", to)
	}

	dstFile, err := os.OpenFile(to, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return fmt.Errorf("could not open file: %s: %s", to, err)
	}
	defer dstFile.Close()

	srcFile, err := os.Open(from)
	if err != nil {
		return fmt.Errorf("could not open file to copy: %v", err)
	}
	defer srcFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		os.Remove(to)
		return fmt.Errorf("could not copy file: %v", err)
	}
	return nil
}

// IsWritable returns true of the directory that is passed in is writable by the
// the current user.
func IsWritable(dir string) bool {
	return unix.Access(dir, unix.W_OK) == nil
}
