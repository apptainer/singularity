/*
 * umoci: Umoci Modifies Open Containers' Images
 * Copyright (C) 2016, 2017, 2018 SUSE LLC.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package unpriv

import (
	"archive/tar"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cyphar/filepath-securejoin"
	"github.com/openSUSE/umoci/pkg/system"
	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
)

// fiRestore restores the state given by an os.FileInfo instance at the given
// path by ensuring that an Lstat(path) will return as-close-to the same
// os.FileInfo.
func fiRestore(path string, fi os.FileInfo) {
	// archive/tar handles the OS-specific syscall stuff required to get atime
	// and mtime information for a file.
	hdr, _ := tar.FileInfoHeader(fi, "")

	// Apply the relevant information from the FileInfo.
	// XXX: Should we return errors here to ensure that everything is
	//      deterministic or we fail?
	os.Chmod(path, fi.Mode())
	os.Chtimes(path, hdr.AccessTime, hdr.ModTime)
}

// splitpath splits the given path into each of the path components.
func splitpath(path string) []string {
	path = filepath.Clean(path)
	parts := strings.Split(path, string(os.PathSeparator))
	if filepath.IsAbs(path) {
		parts = append([]string{string(os.PathSeparator)}, parts...)
	}
	return parts
}

// WrapFunc is a function that can be passed to Wrap. It takes a path (and
// presumably operates on it -- since Wrap only ensures that the path given is
// resolvable) and returns some form of error.
type WrapFunc func(path string) error

// Wrap will wrap a given function, and call it in a context where all of the
// parent directories in the given path argument are such that the path can be
// resolved (you may need to make your own changes to the path to make it
// readable). Note that the provided function may be called several times, and
// if the error returned is such that !os.IsPermission(err), then no trickery
// will be performed. If fn returns an error, so will this function. All of the
// trickery is reverted when this function returns (which is when fn returns).
func Wrap(path string, fn WrapFunc) error {
	// FIXME: Should we be calling fn() here first?
	if err := fn(path); err == nil || !os.IsPermission(errors.Cause(err)) {
		return err
	}

	// We need to chown all of the path components we don't have execute rights
	// to. Specifically these are the path components which are parents of path
	// components we cannot stat. However, we must make sure to not touch the
	// path itself.
	parts := splitpath(filepath.Dir(path))
	start := len(parts)
	for {
		current := filepath.Join(parts[:start]...)
		_, err := os.Lstat(current)
		if err == nil {
			// We've hit the first element we can chown.
			break
		}
		if !os.IsPermission(err) {
			// This is a legitimate error.
			return errors.Wrapf(err, "unpriv.wrap: lstat parent: %s", current)
		}
		start--
	}
	// Chown from the top down.
	for i := start; i <= len(parts); i++ {
		current := filepath.Join(parts[:i]...)
		fi, err := os.Lstat(current)
		if err != nil {
			return errors.Wrapf(err, "unpriv.wrap: lstat parent: %s", current)
		}
		// Add +rwx permissions to directories. If we have the access to change
		// the mode at all then we are the user owner (not just a group owner).
		if err := os.Chmod(current, fi.Mode()|0700); err != nil {
			return errors.Wrapf(err, "unpriv.wrap: chmod parent: %s", current)
		}
		defer fiRestore(current, fi)
	}

	// Everything is wrapped. Return from this nightmare.
	return fn(path)
}

// Open is a wrapper around os.Open which has been wrapped with unpriv.Wrap to
// make it possible to open paths even if you do not currently have read
// permission. Note that the returned file handle references a path that you do
// not have read access to (since all changes are reverted when this function
// returns), so attempts to do Readdir() or similar functions that require
// doing lstat(2) may fail.
func Open(path string) (*os.File, error) {
	var fh *os.File
	err := Wrap(path, func(path string) error {
		// Get information so we can revert it.
		fi, err := os.Lstat(path)
		if err != nil {
			return errors.Wrap(err, "lstat file")
		}

		// Add +r permissions to the file.
		if err := os.Chmod(path, fi.Mode()|0400); err != nil {
			return errors.Wrap(err, "chmod +r")
		}
		defer fiRestore(path, fi)

		// Open the damn thing.
		fh, err = os.Open(path)
		return err
	})
	return fh, errors.Wrap(err, "unpriv.open")
}

// Create is a wrapper around os.Create which has been wrapped with unpriv.Wrap
// to make it possible to create paths even if you do not currently have read
// permission. Note that the returned file handle references a path that you do
// not have read access to (since all changes are reverted when this function
// returns).
func Create(path string) (*os.File, error) {
	var fh *os.File
	err := Wrap(path, func(path string) error {
		var err error
		fh, err = os.Create(path)
		return err
	})
	return fh, errors.Wrap(err, "unpriv.create")
}

// Readdir is a wrapper around (*os.File).Readdir which has been wrapper with
// unpriv.Wrap to make it possible to get []os.FileInfo for the set of children
// of the provided directory path. The interface for this is quite different to
// (*os.File).Readdir because we have to have a proper filesystem path in order
// to get the set of child FileInfos (because all of the child paths need to be
// resolveable).
func Readdir(path string) ([]os.FileInfo, error) {
	var infos []os.FileInfo
	err := Wrap(path, func(path string) error {
		// Get information so we can revert it.
		fi, err := os.Lstat(path)
		if err != nil {
			return errors.Wrap(err, "lstat dir")
		}

		// Add +rx permissions to the file.
		if err := os.Chmod(path, fi.Mode()|0500); err != nil {
			return errors.Wrap(err, "chmod +rx")
		}
		defer fiRestore(path, fi)

		// Open the damn thing.
		fh, err := os.Open(path)
		if err != nil {
			return errors.Wrap(err, "opendir")
		}
		defer fh.Close()

		// Get the set of dirents.
		infos, err = fh.Readdir(-1)
		return err
	})
	return infos, errors.Wrap(err, "unpriv.readdir")
}

// Lstat is a wrapper around os.Lstat which has been wrapped with unpriv.Wrap
// to make it possible to get os.FileInfo about a path even if you do not
// currently have the required mode bits set to resolve the path. Note that you
// may not have resolve access after this function returns because all of the
// trickery is reverted by unpriv.Wrap.
func Lstat(path string) (os.FileInfo, error) {
	var fi os.FileInfo
	err := Wrap(path, func(path string) error {
		// Fairly simple.
		var err error
		fi, err = os.Lstat(path)
		return err
	})
	return fi, errors.Wrap(err, "unpriv.lstat")
}

// Lstatx is like Lstat but uses unix.Lstat and returns unix.Stat_t instead
func Lstatx(path string) (unix.Stat_t, error) {
	var s unix.Stat_t
	err := Wrap(path, func(path string) error {
		return unix.Lstat(path, &s)
	})
	return s, errors.Wrap(err, "unpriv.lstatx")
}

// Readlink is a wrapper around os.Readlink which has been wrapped with
// unpriv.Wrap to make it possible to get the linkname of a symlink even if you
// do not currently have teh required mode bits set to resolve the path. Note
// that you may not have resolve access after this function returns because all
// of this trickery is reverted by unpriv.Wrap.
func Readlink(path string) (string, error) {
	var linkname string
	err := Wrap(path, func(path string) error {
		// Fairly simple.
		var err error
		linkname, err = os.Readlink(path)
		return err
	})
	return linkname, errors.Wrap(err, "unpriv.readlink")
}

// Symlink is a wrapper around os.Symlink which has been wrapped with
// unpriv.Wrap to make it possible to create a symlink even if you do not
// currently have the required access bits to create the symlink. Note that you
// may not have resolve access after this function returns because all of the
// trickery is reverted by unpriv.Wrap.
func Symlink(linkname, path string) error {
	return errors.Wrap(Wrap(path, func(path string) error {
		return os.Symlink(linkname, path)
	}), "unpriv.symlink")
}

// Link is a wrapper around os.Link which has been wrapped with unpriv.Wrap to
// make it possible to create a hard link even if you do not currently have the
// required access bits to create the hard link. Note that you may not have
// resolve access after this function returns because all of the trickery is
// reverted by unpriv.Wrap.
func Link(linkname, path string) error {
	return errors.Wrap(Wrap(path, func(path string) error {
		// We have to double-wrap this, because you need search access to the
		// linkname. This is safe because any common ancestors will be reverted
		// in reverse call stack order.
		return errors.Wrap(Wrap(linkname, func(linkname string) error {
			return os.Link(linkname, path)
		}), "unpriv.wrap linkname")
	}), "unpriv.link")
}

// Chmod is a wrapper around os.Chmod which has been wrapped with unpriv.Wrap
// to make it possible to change the permission bits of a path even if you do
// not currently have the required access bits to access the path.
func Chmod(path string, mode os.FileMode) error {
	return errors.Wrap(Wrap(path, func(path string) error {
		return os.Chmod(path, mode)
	}), "unpriv.chmod")
}

// Lchown is a wrapper around os.Lchown which has been wrapped with unpriv.Wrap
// to make it possible to change the owner of a path even if you do not
// currently have the required access bits to access the path. Note that this
// function is not particularly useful in most rootless scenarios.
//
// FIXME: This probably should be removed because it's questionably useful.
func Lchown(path string, uid, gid int) error {
	return errors.Wrap(Wrap(path, func(path string) error {
		return os.Lchown(path, uid, gid)
	}), "unpriv.lchown")
}

// Chtimes is a wrapper around os.Chtimes which has been wrapped with
// unpriv.Wrap to make it possible to change the modified times of a path even
// if you do not currently have the required access bits to access the path.
func Chtimes(path string, atime, mtime time.Time) error {
	return errors.Wrap(Wrap(path, func(path string) error {
		return os.Chtimes(path, atime, mtime)
	}), "unpriv.chtimes")
}

// Lutimes is a wrapper around system.Lutimes which has been wrapped with
// unpriv.Wrap to make it possible to change the modified times of a path even
// if you do no currently have the required access bits to access the path.
func Lutimes(path string, atime, mtime time.Time) error {
	return errors.Wrap(Wrap(path, func(path string) error {
		return system.Lutimes(path, atime, mtime)
	}), "unpriv.lutimes")
}

// Remove is a wrapper around os.Remove which has been wrapped with unpriv.Wrap
// to make it possible to remove a path even if you do not currently have the
// required access bits to modify or resolve the path.
func Remove(path string) error {
	return errors.Wrap(Wrap(path, os.Remove), "unpriv.remove")
}

// foreachSubpath executes WrapFunc for each child of the given path (not
// including the path itself). If path is not a directory, then WrapFunc will
// not be called and no error will be returned. This should be called within a
// context where path has already been made resolveable, however the . If WrapFunc returns an
// error, the first error is returned and iteration is halted.
func foreachSubpath(path string, wrapFn WrapFunc) error {
	// Is the path a directory?
	fi, err := os.Lstat(path)
	if err != nil {
		return errors.WithStack(err)
	}
	if !fi.IsDir() {
		return nil
	}

	// Open the directory.
	fd, err := Open(path)
	if err != nil {
		return errors.WithStack(err)
	}
	defer fd.Close()

	// We need to change the mode to Readdirnames. We don't need to worry about
	// permissions because we're already in a context with filepath.Dir(path)
	// is at least a+rx. However, because we are calling wrapFn we need to
	// restore the original mode immediately.
	os.Chmod(path, fi.Mode()|0444)
	names, err := fd.Readdirnames(-1)
	fiRestore(path, fi)
	if err != nil {
		return errors.WithStack(err)
	}

	// Make iteration order consistent.
	sort.Strings(names)

	// Call on all the sub-directories. We run it in a Wrap context to ensure
	// that the path we pass is resolveable when executed.
	for _, name := range names {
		subpath := filepath.Join(path, name)
		if err := Wrap(subpath, wrapFn); err != nil {
			return err
		}
	}
	return nil
}

// RemoveAll is similar to os.RemoveAll but with all of the internal functions
// wrapped with unpriv.Wrap to make it possible to remove a path (even if it
// has child paths) even if you do not currently have enough access bits.
func RemoveAll(path string) error {
	return errors.Wrap(Wrap(path, func(path string) error {
		// If remove works, we're done.
		err := os.Remove(path)
		if err == nil || os.IsNotExist(errors.Cause(err)) {
			return nil
		}

		// Is this a directory?
		fi, serr := os.Lstat(path)
		if serr != nil {
			// Use securejoin's IsNotExist to handle ENOTDIR sanely.
			if securejoin.IsNotExist(errors.Cause(serr)) {
				serr = nil
			}
			return errors.Wrap(serr, "lstat")
		}
		// Return error from remove if it's not a directory.
		if !fi.IsDir() {
			return errors.Wrap(err, "remove non-directory")
		}
		err = nil

		err1 := foreachSubpath(path, func(subpath string) error {
			err2 := RemoveAll(subpath)
			if err == nil {
				err = err2
			}
			return nil
		})
		if err1 != nil {
			// We must have hit a race, but we don't care.
			if os.IsNotExist(errors.Cause(err1)) {
				err1 = nil
			}
			return errors.Wrap(err1, "foreach subpath")
		}

		// Remove the directory. This should now work.
		err1 = os.Remove(path)
		if err1 == nil || os.IsNotExist(errors.Cause(err1)) {
			return nil
		}
		if err == nil {
			err = err1
		}
		return errors.Wrap(err, "remove")
	}), "unpriv.removeall")
}

// Mkdir is a wrapper around os.Mkdir which has been wrapped with unpriv.Wrap
// to make it possible to remove a path even if you do not currently have the
// required access bits to modify or resolve the path.
func Mkdir(path string, perm os.FileMode) error {
	return errors.Wrap(Wrap(path, func(path string) error {
		return os.Mkdir(path, perm)
	}), "unpriv.mkdir")
}

// MkdirAll is similar to os.MkdirAll but in order to implement it properly all
// of the internal functions were wrapped with unpriv.Wrap to make it possible
// to create a path even if you do not currently have enough access bits.
func MkdirAll(path string, perm os.FileMode) error {
	return errors.Wrap(Wrap(path, func(path string) error {
		// Check whether the path already exists.
		fi, err := os.Stat(path)
		if err == nil {
			if fi.IsDir() {
				return nil
			}
			return &os.PathError{Op: "mkdir", Path: path, Err: unix.ENOTDIR}
		}

		// Create parent.
		parent := filepath.Dir(path)
		if parent != "." && parent != "/" {
			err = MkdirAll(parent, perm)
			if err != nil {
				return err
			}
		}

		// Parent exists, now we can create the path.
		err = os.Mkdir(path, perm)
		if err != nil {
			// Handle "foo/.".
			fi, err1 := os.Lstat(path)
			if err1 == nil && fi.IsDir() {
				return nil
			}
			return err
		}
		return nil
	}), "unpriv.mkdirall")
}

// Mknod is a wrapper around unix.Mknod which has been wrapped with unpriv.Wrap
// to make it possible to remove a path even if you do not currently have the
// required access bits to modify or resolve the path.
func Mknod(path string, mode os.FileMode, dev uint64) error {
	return errors.Wrap(Wrap(path, func(path string) error {
		return unix.Mknod(path, uint32(mode), int(dev))
	}), "unpriv.mknod")
}

// Llistxattr is a wrapper around system.Llistxattr which has been wrapped with
// unpriv.Wrap to make it possible to remove a path even if you do not
// currently have the required access bits to resolve the path.
func Llistxattr(path string) ([]string, error) {
	var xattrs []string
	err := Wrap(path, func(path string) error {
		var err error
		xattrs, err = system.Llistxattr(path)
		return err
	})
	return xattrs, errors.Wrap(err, "unpriv.llistxattr")
}

// Lremovexattr is a wrapper around system.Lremovexattr which has been wrapped
// with unpriv.Wrap to make it possible to remove a path even if you do not
// currently have the required access bits to resolve the path.
func Lremovexattr(path, name string) error {
	return errors.Wrap(Wrap(path, func(path string) error {
		return unix.Lremovexattr(path, name)
	}), "unpriv.lremovexattr")
}

// Lsetxattr is a wrapper around system.Lsetxattr which has been wrapped
// with unpriv.Wrap to make it possible to set a path even if you do not
// currently have the required access bits to resolve the path.
func Lsetxattr(path, name string, value []byte, flags int) error {
	return errors.Wrap(Wrap(path, func(path string) error {
		return unix.Lsetxattr(path, name, value, flags)
	}), "unpriv.lsetxattr")
}

// Lgetxattr is a wrapper around system.Lgetxattr which has been wrapped
// with unpriv.Wrap to make it possible to get a path even if you do not
// currently have the required access bits to resolve the path.
func Lgetxattr(path, name string) ([]byte, error) {
	var value []byte
	err := Wrap(path, func(path string) error {
		var err error
		value, err = system.Lgetxattr(path, name)
		return err
	})
	return value, errors.Wrap(err, "unpriv.lgetxattr")
}

// Lclearxattrs is similar to system.Lclearxattrs but in order to implement it
// properly all of the internal functions were wrapped with unpriv.Wrap to make
// it possible to create a path even if you do not currently have enough access
// bits.
func Lclearxattrs(path string, except map[string]struct{}) error {
	return errors.Wrap(Wrap(path, func(path string) error {
		names, err := Llistxattr(path)
		if err != nil {
			return err
		}
		for _, name := range names {
			if _, skip := except[name]; skip {
				continue
			}
			if err := Lremovexattr(path, name); err != nil {
				// SELinux won't let you change security.selinux (for obvious
				// security reasons), so we don't clear xattrs if attempting to
				// clear them causes an EPERM. This EPERM will not be due to
				// resolution issues (Llistxattr already has done that for us).
				if os.IsPermission(errors.Cause(err)) {
					continue
				}
				return err
			}
		}
		return nil
	}), "unpriv.lclearxattrs")
}

// walk is the inner implementation of Walk.
func walk(path string, info os.FileInfo, walkFn filepath.WalkFunc) error {
	// Always run walkFn first. If we're not a directory there's no children to
	// iterate over and so we bail even if there wasn't an error.
	err := walkFn(path, info, nil)
	if !info.IsDir() || err != nil {
		return err
	}

	// Now just execute walkFn over each subpath.
	return foreachSubpath(path, func(subpath string) error {
		info, err := Lstat(subpath)
		if err != nil {
			// If it doesn't exist, just pass it directly to walkFn.
			if err := walkFn(subpath, info, err); err != nil {
				// Ignore SkipDir.
				if errors.Cause(err) != filepath.SkipDir {
					return err
				}
			}
		} else {
			if err := walk(subpath, info, walkFn); err != nil {
				// Ignore error if it's SkipDir and subpath is a directory.
				if !(info.IsDir() && errors.Cause(err) == filepath.SkipDir) {
					return err
				}
			}
		}
		return nil
	})
}

// Walk is a reimplementation of filepath.Walk, wrapping all of the relevant
// function calls with Wrap, allowing you to walk over a tree even in the face
// of multiple nested cases where paths are not normally accessible. The
// os.FileInfo passed to walkFn is the "pristine" version (as opposed to the
// currently-on-disk version that may have been temporarily modified by Wrap).
func Walk(root string, walkFn filepath.WalkFunc) error {
	return Wrap(root, func(root string) error {
		info, err := Lstat(root)
		if err != nil {
			err = walkFn(root, nil, err)
		} else {
			err = walk(root, info, walkFn)
		}
		if errors.Cause(err) == filepath.SkipDir {
			err = nil
		}
		return errors.Wrap(err, "unpriv.walk")
	})
}
