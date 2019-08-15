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

package fseval

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/openSUSE/umoci/pkg/unpriv"
	"github.com/vbatts/go-mtree"
	"golang.org/x/sys/unix"
)

// RootlessFsEval is an FsEval implementation that uses "umoci/pkg/unpriv".*
// functions in order to provide the ability for unprivileged users (those
// without CAP_DAC_OVERRIDE and CAP_DAC_READ_SEARCH) to evaluate parts of a
// filesystem that they own. Note that by necessity this requires modifying the
// filesystem (and thus will not work on read-only filesystems).
var RootlessFsEval FsEval = unprivFsEval(0)

// unprivFsEval is a hack to be able to make RootlessFsEval a const.
type unprivFsEval int

// Open is equivalent to unpriv.Open.
func (fs unprivFsEval) Open(path string) (*os.File, error) {
	return unpriv.Open(path)
}

// Create is equivalent to unpriv.Create.
func (fs unprivFsEval) Create(path string) (*os.File, error) {
	return unpriv.Create(path)
}

// Readdir is equivalent to unpriv.Readdir.
func (fs unprivFsEval) Readdir(path string) ([]os.FileInfo, error) {
	return unpriv.Readdir(path)
}

// Lstat is equivalent to unpriv.Lstat.
func (fs unprivFsEval) Lstat(path string) (os.FileInfo, error) {
	return unpriv.Lstat(path)
}

func (fs unprivFsEval) Lstatx(path string) (unix.Stat_t, error) {
	return unpriv.Lstatx(path)
}

// Readlink is equivalent to unpriv.Readlink.
func (fs unprivFsEval) Readlink(path string) (string, error) {
	return unpriv.Readlink(path)
}

// Symlink is equivalent to unpriv.Symlink.
func (fs unprivFsEval) Symlink(linkname, path string) error {
	return unpriv.Symlink(linkname, path)
}

// Link is equivalent to unpriv.Link.
func (fs unprivFsEval) Link(linkname, path string) error {
	return unpriv.Link(linkname, path)
}

// Chmod is equivalent to unpriv.Chmod.
func (fs unprivFsEval) Chmod(path string, mode os.FileMode) error {
	return unpriv.Chmod(path, mode)
}

// Lutimes is equivalent to unpriv.Lutimes.
func (fs unprivFsEval) Lutimes(path string, atime, mtime time.Time) error {
	return unpriv.Lutimes(path, atime, mtime)
}

// Remove is equivalent to unpriv.Remove.
func (fs unprivFsEval) Remove(path string) error {
	return unpriv.Remove(path)
}

// RemoveAll is equivalent to unpriv.RemoveAll.
func (fs unprivFsEval) RemoveAll(path string) error {
	return unpriv.RemoveAll(path)
}

// Mkdir is equivalent to unpriv.Mkdir.
func (fs unprivFsEval) Mkdir(path string, perm os.FileMode) error {
	return unpriv.Mkdir(path, perm)
}

// Mknod is equivalent to unpriv.Mknod.
func (fs unprivFsEval) Mknod(path string, mode os.FileMode, dev uint64) error {
	return unpriv.Mknod(path, mode, dev)
}

// MkdirAll is equivalent to unpriv.MkdirAll.
func (fs unprivFsEval) MkdirAll(path string, perm os.FileMode) error {
	return unpriv.MkdirAll(path, perm)
}

// Llistxattr is equivalent to unpriv.Llistxattr
func (fs unprivFsEval) Llistxattr(path string) ([]string, error) {
	return unpriv.Llistxattr(path)
}

// Lremovexattr is equivalent to unpriv.Lremovexattr
func (fs unprivFsEval) Lremovexattr(path, name string) error {
	return unpriv.Lremovexattr(path, name)
}

// Lsetxattr is equivalent to unpriv.Lsetxattr
func (fs unprivFsEval) Lsetxattr(path, name string, value []byte, flags int) error {
	return unpriv.Lsetxattr(path, name, value, flags)
}

// Lgetxattr is equivalent to unpriv.Lgetxattr
func (fs unprivFsEval) Lgetxattr(path string, name string) ([]byte, error) {
	return unpriv.Lgetxattr(path, name)
}

// Lclearxattrs is equivalent to unpriv.Lclearxattrs
func (fs unprivFsEval) Lclearxattrs(path string, except map[string]struct{}) error {
	return unpriv.Lclearxattrs(path, except)
}

// KeywordFunc returns a wrapper around the given mtree.KeywordFunc.
func (fs unprivFsEval) KeywordFunc(fn mtree.KeywordFunc) mtree.KeywordFunc {
	return func(path string, info os.FileInfo, r io.Reader) ([]mtree.KeyVal, error) {
		var kv []mtree.KeyVal
		err := unpriv.Wrap(path, func(path string) error {
			var err error
			kv, err = fn(path, info, r)
			return err
		})
		return kv, err
	}
}

// Walk is equivalent to filepath.Walk.
func (fs unprivFsEval) Walk(root string, fn filepath.WalkFunc) error {
	return unpriv.Walk(root, fn)
}
