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

// walkSymRelative follows and resolves all symlinks found in a path
// located in root, it ensures symlinks resolution never go past the
// provided root path.
func walkSymRelative(path string, root string, maxLinks uint) string {
	// symlinks counter
	symlinks := uint(0)

	// generate clean absolute path
	absRoot := filepath.Join("/", root)
	absPath := filepath.Join("/", path)

	sep := string(os.PathSeparator)

	// get path components by skipping the first
	// character as it will be always "/" to avoid
	// to add an empty first element in the array
	comp := strings.Split(absPath[1:], sep)

	// start from absolute root path
	dest := absRoot

	for i := 0; i < len(comp); i++ {
		dest = filepath.Join(dest, comp[i])

		// this call can return various errors that we don't need
		// or want to deal with like a lack of permission for a
		// directory traversal, not a symlink, non-existent path.
		// As this function doesn't return any error we ignore them
		// and generate the path assuming there is no hidden symlink
		// in the next components
		d, err := os.Readlink(dest)
		if err != nil {
			continue
		}
		symlinks++

		newDest := absRoot

		if !filepath.IsAbs(d) {
			// this is a relative target, we are taking the current
			// parent path of dest concatenated with the target
			parentDest := filepath.Dir(dest)
			dest = filepath.Join(parentDest, d)

			// if we are outside of root, we join the relative target
			// with "/" to obtain an absolute path as we were at root
			// of "/" thanks to filepath.Clean implicitly called by
			// filepath.Join
			if !strings.HasPrefix(dest, absRoot) {
				d = filepath.Join("/", d)
				dest = filepath.Join(absRoot, d)
			} else {
				if strings.HasPrefix(dest, parentDest) {
					// trivial case where the resulting path is
					// within the current path
					d = strings.TrimPrefix(dest, parentDest)
					newDest = parentDest
				} else {
					// we go back in the hierarchy and take a
					// naive approach by trimming root prefix
					// from path instead of finding the exact
					// path chunk involving too much complexity
					d = strings.TrimPrefix(dest, absRoot)
				}
			}
		} else {
			// it's an absolute path, simply concatenate root
			// and symlink target
			dest = filepath.Join(absRoot, d)
		}

		// too many symbolic links, stop and return the
		// resolved path as is, should not happen with
		// sane images
		if symlinks == maxLinks {
			break
		}
		// symlink target point to the current destination,
		// nothing to do
		if len(d) == 0 {
			continue
		}

		dest = newDest
		// either replace current path components or merge
		// the components of the symlink target with the next
		// components
		if i+1 < len(comp) {
			comp = append(strings.Split(d[1:], sep), comp[i+1:]...)
		} else {
			comp = strings.Split(d[1:], sep)
		}
		i = -1
	}

	// ensure the final path is absolute
	return filepath.Join("/", strings.TrimPrefix(dest, absRoot))
}

// EvalRelative evaluates symlinks in path relative to root path, it returns
// a path as if it was evaluated from chroot. This function always returns
// an absolute path and is intended to be used to resolve mount points
// destinations, it helps the runtime to not bind mount directories/files
// outside of the container image provided by the root argument.
func EvalRelative(path string, root string) string {
	// return "/ if path is empty
	if path == "" {
		return "/"
	}
	// resolve path and allow up to 40 symlinks
	return walkSymRelative(path, root, 40)
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
	err = PermWalk(path, func(path string, f os.FileInfo, err error) error {
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

// PermWalk is similar to filepath.Walk - but:
//   1. The skipDir checks are removed (we never want to skip anything here)
//   2. Our walk will call walkFn on a directory *before* attempting to look
//      inside that directory.
func PermWalk(root string, walkFn filepath.WalkFunc) error {
	info, err := os.Lstat(root)
	if err != nil {
		return fmt.Errorf("could not access path %s: %s", root, err)
	}
	return permWalk(root, info, walkFn)
}

func permWalk(path string, info os.FileInfo, walkFn filepath.WalkFunc) error {
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
			err = permWalk(filename, fileInfo, walkFn)
			if err != nil {
				if !fileInfo.IsDir() {
					return err
				}
			}
		}
	}
	return nil
}

// PermWalkRaiseError is similar to filepath.Walk - but:
//   1. The skipDir checks are removed (we never want to skip anything here)
//   2. Our walk will call walkFn on a directory *before* attempting to look
//      inside that directory.
//   3. We back out of the recursion at the *first* error... we don't attempt
//      to go through as much as we can.
func PermWalkRaiseError(root string, walkFn filepath.WalkFunc) error {
	info, err := os.Lstat(root)
	if err != nil {
		return fmt.Errorf("could not access path %s: %s", root, err)
	}
	return permWalkRaiseError(root, info, walkFn)
}

func permWalkRaiseError(path string, info os.FileInfo, walkFn filepath.WalkFunc) error {
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
			return err
		}
		if err = permWalkRaiseError(filename, fileInfo, walkFn); err != nil {
			return err
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
