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
	"os"
	"path/filepath"
	"time"

	"github.com/vbatts/go-mtree"
	"golang.org/x/sys/unix"
)

// Ensure that mtree.FsEval is implemented by FsEval.
var _ mtree.FsEval = DefaultFsEval
var _ mtree.FsEval = RootlessFsEval

// FsEval is a super-interface that implements everything required for
// mtree.FsEval as well as including all of the imporant os.* wrapper functions
// needed for "oci/layers".tarExtractor.
type FsEval interface {
	// Open is equivalent to os.Open.
	Open(path string) (*os.File, error)

	// Create is equivalent to os.Create.
	Create(path string) (*os.File, error)

	// Readdir is equivalent to os.Readdir.
	Readdir(path string) ([]os.FileInfo, error)

	// Lstat is equivalent to os.Lstat.
	Lstat(path string) (os.FileInfo, error)

	// Lstatx is equivalent to unix.Lstat.
	Lstatx(path string) (unix.Stat_t, error)

	// Readlink is equivalent to os.Readlink.
	Readlink(path string) (string, error)

	// Symlink is equivalent to os.Symlink.
	Symlink(linkname, path string) error

	// Link is equivalent to os.Link.
	Link(linkname, path string) error

	// Chmod is equivalent to os.Chmod.
	Chmod(path string, mode os.FileMode) error

	// Lutimes is equivalent to os.Lutimes.
	Lutimes(path string, atime, mtime time.Time) error

	// Remove is equivalent to os.Remove.
	Remove(path string) error

	// RemoveAll is equivalent to os.RemoveAll.
	RemoveAll(path string) error

	// Mkdir is equivalent to os.Mkdir.
	Mkdir(path string, perm os.FileMode) error

	// MkdirAll is equivalent to os.MkdirAll.
	MkdirAll(path string, perm os.FileMode) error

	// Mknod is equivalent to unix.Mknod.
	Mknod(path string, mode os.FileMode, dev uint64) error

	// Llistxattr is equivalent to system.Llistxattr
	Llistxattr(path string) ([]string, error)

	// Lremovexattr is equivalent to system.Lremovexattr
	Lremovexattr(path, name string) error

	// Lsetxattr is equivalent to system.Lsetxattr
	Lsetxattr(path, name string, value []byte, flags int) error

	// Lgetxattr is equivalent to system.Lgetxattr
	Lgetxattr(path string, name string) ([]byte, error)

	// Lclearxattrs is equivalent to system.Lclearxattrs
	Lclearxattrs(path string, except map[string]struct{}) error

	// KeywordFunc returns a wrapper around the given mtree.KeywordFunc.
	KeywordFunc(fn mtree.KeywordFunc) mtree.KeywordFunc

	// Walk is equivalent to filepath.Walk.
	Walk(root string, fn filepath.WalkFunc) error
}
