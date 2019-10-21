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
	"sort"
	"strings"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"golang.org/x/sys/unix"
)

// EnsureFileWithPermission takes a file path, and 1. Creates it with
// the specified permission, or 2. ensures a file is the specified
// permission.
func EnsureFileWithPermission(fn string, mode os.FileMode) error {
	fs, err := os.OpenFile(fn, os.O_CREATE, mode)
	if err != nil {
		return err
	}
	defer fs.Close()

	// check the permissions.
	fsinfo, err := fs.Stat()
	if err != nil {
		return err
	}

	if currentMode := fsinfo.Mode(); currentMode != mode {
		sylog.Warningf("File mode (%o) on %s needs to be %o, fixing that...", currentMode, fn, mode)
		if err := fs.Chmod(mode); err != nil {
			return err
		}
	}

	return nil
}

// IsFile check if name component is regular file.
func IsFile(name string) bool {
	info, err := os.Stat(name)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}

// IsDir check if name component is a directory.
func IsDir(name string) bool {
	info, err := os.Stat(name)
	if err != nil {
		return false
	}
	return info.Mode().IsDir()
}

// IsLink check if name component is a symlink.
func IsLink(name string) bool {
	info, err := os.Lstat(name)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}

// IsOwner check if name component is owned by user identified with uid.
func IsOwner(name string, uid uint32) bool {
	info, err := os.Stat(name)
	if err != nil {
		return false
	}
	return info.Sys().(*syscall.Stat_t).Uid == uid
}

// IsExec check if name component has executable bit permission set.
func IsExec(name string) bool {
	info, err := os.Stat(name)
	if err != nil {
		return false
	}
	return info.Sys().(*syscall.Stat_t).Mode&syscall.S_IXUSR != 0
}

// IsSuid check if name component has setuid bit permission set.
func IsSuid(name string) bool {
	info, err := os.Stat(name)
	if err != nil {
		return false
	}
	return info.Sys().(*syscall.Stat_t).Mode&syscall.S_ISUID != 0
}

// MkdirAll creates a directory and parents if it doesn't exist with
// mode after umask reset.
func MkdirAll(path string, mode os.FileMode) error {
	oldmask := syscall.Umask(0)
	defer syscall.Umask(oldmask)

	return os.MkdirAll(path, mode)
}

// Mkdir creates a directory if it doesn't exist with
// mode after umask reset.
func Mkdir(path string, mode os.FileMode) error {
	oldmask := syscall.Umask(0)
	defer syscall.Umask(oldmask)

	return os.Mkdir(path, mode)
}

// RootDir returns the root directory of path (rootdir of /my/path is /my).
// Returns "." if path is empty.
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
// function doesn't return error but always returns an evaluated path.
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

// PathExists simply checks if a path exists.
func PathExists(path string) (bool, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

// CopyFile copies file to the provided location making sure the resulting
// file has permission bits set to the mode prior to umask. To honor umask
// correctly the resulting file must not exist.
func CopyFile(from, to string, mode os.FileMode) (err error) {
	exist, err := PathExists(to)
	if err != nil {
		return err
	}
	if exist {
		return fmt.Errorf("file %s already exists", to)
	}

	dstFile, err := os.OpenFile(to, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return fmt.Errorf("could not open %s: %v", to, err)
	}
	defer func() {
		dstFile.Close()
		if err != nil {
			os.Remove(to)
		}
	}()

	srcFile, err := os.Open(from)
	if err != nil {
		return fmt.Errorf("could not open file to copy: %v", err)
	}
	defer srcFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("could not copy file: %v", err)
	}

	return nil
}

// IsWritable returns true of the file that is passed in
// is writable by the user (note: uid is checked, not euid).
func IsWritable(path string) bool {
	return unix.Access(path, unix.W_OK) == nil
}

// FirstExistingParent walks up the supplied path and returns the first
// parent that exists. If the supplied path exists, it will just return that path.
// Assumes cwd and the root directory always exists
func FirstExistingParent(path string) (string, error) {
	p := filepath.Clean(path)
	for p != "/" && p != "." {
		exists, err := PathExists(p)
		if err != nil {
			return "", err
		}
		if exists {
			return p, nil
		}

		p = filepath.Dir(p)
	}

	return p, nil
}

// ForceRemoveAll removes a directory like os.RemoveAll, except that it will
// chmod any directory who's permissions are preventing the removal of contents
func ForceRemoveAll(path string) error {
	// First try to remove the directory with os.RemoveAll. This will remove
	// as much as it can, and return the first error (if any) - so we can avoid
	// messing with permissions unless we need to.
	err := os.RemoveAll(path)
	// Anything other than an permission error is out of scope for us to deal
	// with here.
	if err == nil || !os.IsPermission(err) {
		return err
	}

	// At this point there is a permissions error. Removal of files is dependent
	// on the permissions of the containing directory, so walk the (remaining)
	// tree and set perms that work.
	sylog.Debugf("Forcing permissions to remove %q completely", path)
	errors := 0
	err = permWalk(path, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			sylog.Errorf("Unable to access path %s: %s", path, err)
			errors++
			return nil
		}
		// Directories must have the owner 'rx' bits to allow traversal, reading content, and the 'w' bit
		// so their content can be deleted by the user when the bundle is deleted
		if f.Mode().IsDir() {
			if err := os.Chmod(path, f.Mode().Perm()|0700); err != nil {
				sylog.Errorf("Error setting permissions to remove %s: %s", path, err)
				errors++
			}
		}
		return nil
	})

	// Catastrophic error during the permission walk
	if err != nil {
		sylog.Errorf("Unable to set permissions to remove %q: %s", path, err)
	}
	// Individual errors accumulated while setting permissions in the walk
	if errors > 0 {
		sylog.Errorf("%d errors were encountered when setting permissions to remove bundle", errors)
	}

	// Call RemoveAll again to get rid of things... even if we had errors when
	// trying to set permissions, so we remove as much as possible.
	return os.RemoveAll(path)
}

// permWalk is similar to filepath.Walk - but:
//   1. The skipDir checks are removed (we never want to skip anything here)
//   2. Our walk will call walkFn on a directory *before* attempting to look
//      inside that directory.
func permWalk(root string, walkFn filepath.WalkFunc) error {
	info, err := os.Lstat(root)
	if err != nil {
		return fmt.Errorf("could not access rootfs %s: %s", root, err)
	}
	return walk(root, info, walkFn)
}

func walk(path string, info os.FileInfo, walkFn filepath.WalkFunc) error {
	if !info.IsDir() {
		return walkFn(path, info, nil)
	}

	// Unlike filepath.walk we call walkFn *before* trying to list the content of
	// the directory, so that walkFn has a chance to assign perms that allow us into
	// the directory, if we can't get in there already.
	if err := walkFn(path, info, nil); err != nil {
		return err
	}

	names, err := readDirNames(path)
	if err != nil {
		return err
	}

	for _, name := range names {
		filename := filepath.Join(path, name)
		fileInfo, err := os.Lstat(filename)
		if err != nil {
			if err := walkFn(filename, fileInfo, err); err != nil {
				return err
			}
		} else {
			err = walk(filename, fileInfo, walkFn)
			if err != nil {
				if !fileInfo.IsDir() {
					return err
				}
			}
		}
	}
	return nil
}

// readDirNames reads the directory named by dirname and returns
// a sorted list of directory entries.
func readDirNames(dirname string) ([]string, error) {
	f, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}
	names, err := f.Readdirnames(-1)
	f.Close()
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	return names, nil
}
