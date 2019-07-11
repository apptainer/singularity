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

package layer

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/openSUSE/umoci/pkg/fseval"
	"github.com/openSUSE/umoci/pkg/testutils"
	"github.com/pkg/errors"
)

// ignoreXattrs is a list of xattr names that should be ignored when
// creating a new image layer, because they are host-specific and/or would be a
// bad idea to unpack. They are also excluded from Lclearxattr when extracting
// an archive.
// XXX: Maybe we should make this configurable so users can manually blacklist
//      (or even whitelist) xattrs that they actually want included? Like how
//      GNU tar's xattr setup works.
var ignoreXattrs = map[string]struct{}{
	// SELinux doesn't allow you to set SELinux policies generically. They're
	// also host-specific. So just ignore them during extraction.
	"security.selinux": {},

	// NFSv4 ACLs are very system-specific and shouldn't be touched by us, nor
	// should they be included in images.
	"system.nfs4_acl": {},
}

func init() {
	// For test purposes we add a fake forbidden attribute that an unprivileged
	// user can easily write to (and thus we can test it).
	if testutils.IsTestBinary() {
		ignoreXattrs["user.UMOCI:forbidden_xattr"] = struct{}{}
	}
}

// tarGenerator is a helper for generating layer diff tars. It should be noted
// that when using tarGenerator.Add{Path,Whiteout} it is recommended to do it
// in lexicographic order.
type tarGenerator struct {
	tw *tar.Writer

	// mapOptions is the set of mapping options for modifying entries before
	// they're added to the layer.
	mapOptions MapOptions

	// Hardlink mapping.
	inodes map[uint64]string

	// fsEval is an fseval.FsEval used for extraction.
	fsEval fseval.FsEval

	// XXX: Should we add a saftey check to make sure we don't generate two of
	//      the same path in a tar archive? This is not permitted by the spec.
}

// newTarGenerator creates a new tarGenerator using the provided writer as the
// output writer.
func newTarGenerator(w io.Writer, opt MapOptions) *tarGenerator {
	fsEval := fseval.DefaultFsEval
	if opt.Rootless {
		fsEval = fseval.RootlessFsEval
	}

	return &tarGenerator{
		tw:         tar.NewWriter(w),
		mapOptions: opt,
		inodes:     map[uint64]string{},
		fsEval:     fsEval,
	}
}

// normalise converts the provided pathname to a POSIX-compliant pathname. It also will provide an error if a path looks unsafe.
func normalise(rawPath string, isDir bool) (string, error) {
	// Clean up the path.
	path := CleanPath(rawPath)

	// Nothing to do.
	if path == "." {
		return ".", nil
	}

	if filepath.IsAbs(path) {
		path = strings.TrimPrefix(path, "/")
	}

	// Check that the path is "safe", meaning that it doesn't resolve outside
	// of the tar archive. While this might seem paranoid, it is a legitimate
	// concern.
	if "/"+path != filepath.Join("/", path) {
		return "", errors.Errorf("escape warning: generated path is outside tar root: %s", rawPath)
	}

	// With some other tar formats, you needed to have a '/' at the end of a
	// pathname in order to state that it is a directory. While this is no
	// longer necessary, some older tooling may assume that.
	if isDir {
		path += "/"
	}

	return path, nil
}

// AddFile adds a file from the filesystem to the tar archive. It copies all of
// the relevant stat information about the file, and also attempts to track
// hardlinks. This should be functionally equivalent to adding entries with GNU
// tar.
func (tg *tarGenerator) AddFile(name, path string) error {
	fi, err := tg.fsEval.Lstat(path)
	if err != nil {
		return errors.Wrap(err, "add file lstat")
	}

	linkname := ""
	if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
		if linkname, err = tg.fsEval.Readlink(path); err != nil {
			return errors.Wrap(err, "add file readlink")
		}
	}

	hdr, err := tar.FileInfoHeader(fi, linkname)
	if err != nil {
		return errors.Wrap(err, "convert fi to hdr")
	}
	hdr.Xattrs = map[string]string{}
	// Usually incorrect for containers and was added in Go 1.10 causing
	// changes to our output on a compiler bump...
	hdr.Uname = ""
	hdr.Gname = ""

	name, err = normalise(name, fi.IsDir())
	if err != nil {
		return errors.Wrap(err, "normalise path")
	}
	hdr.Name = name

	// Make sure that we don't include any files with the name ".wh.". This
	// will almost certainly confuse some users (unfortunately) but there's
	// nothing we can do to store such files on-disk.
	if strings.HasPrefix(filepath.Base(name), whPrefix) {
		return errors.Errorf("invalid path has whiteout prefix %q: %s", whPrefix, name)
	}

	// FIXME: Do we need to ensure that the parent paths have all been added to
	//        the archive? I haven't found any tar specification that makes
	//        this mandatory, but I have a feeling that some people might rely
	//        on it. The issue with implementing it is that we'd have to get
	//        the FileInfo about the directory from somewhere (and we don't
	//        want to waste space by adding an entry that will be overwritten
	//        later).

	// Different systems have different special things they need to set within
	// a tar header. For example, device numbers are quite important to be set
	// by us.
	statx, err := tg.fsEval.Lstatx(path)
	if err != nil {
		return errors.Wrapf(err, "lstatx %q", path)
	}
	updateHeader(hdr, statx)

	// Set up xattrs externally to updateHeader because the function signature
	// would look really dumb otherwise.
	// XXX: This should probably be moved to a function in tar_unix.go.
	names, err := tg.fsEval.Llistxattr(path)
	if err != nil {
		return errors.Wrap(err, "get xattr list")
	}
	for _, name := range names {
		// Some xattrs need to be skipped for sanity reasons, such as
		// security.selinux, because they are very much host-specific and
		// carrying them to other hosts would be a really bad idea.
		if _, ignore := ignoreXattrs[name]; ignore {
			continue
		}
		// TODO: We should translate all v3 capabilities into root-owned
		//       capabilities here. But we don't have Go code for that yet
		//       (we'd need to use libcap to parse it).
		value, err := tg.fsEval.Lgetxattr(path, name)
		if err != nil {
			// XXX: I'm not sure if we're unprivileged whether Lgetxattr can
			//      fail with EPERM. If it can, we should ignore it (like when
			//      we try to clear xattrs).
			return errors.Wrapf(err, "get xattr: %s", name)
		}
		// https://golang.org/issues/20698 -- We don't just error out here
		// because it's not _really_ a fatal error. Currently it's unclear
		// whether the stdlib will correctly handle reading or disable writing
		// of these PAX headers so we have to track this ourselves.
		if len(value) <= 0 {
			log.Warnf("ignoring empty-valued xattr %s: disallowed by PAX standard", name)
			continue
		}
		// Note that Go strings can actually be arbitrary byte sequences, so
		// this conversion (while it might look a bit wrong) is actually fine.
		hdr.Xattrs[name] = string(value)
	}

	// Not all systems have the concept of an inode, but I'm not in the mood to
	// handle this in a way that makes anything other than GNU/Linux happy
	// right now. Handle hardlinks.
	if oldpath, ok := tg.inodes[statx.Ino]; ok {
		// We just hit a hardlink, so we just have to change the header.
		hdr.Typeflag = tar.TypeLink
		hdr.Linkname = oldpath
		hdr.Size = 0
	} else {
		tg.inodes[statx.Ino] = name
	}

	// Apply any header mappings.
	if err := mapHeader(hdr, tg.mapOptions); err != nil {
		return errors.Wrap(err, "map header")
	}
	if err := tg.tw.WriteHeader(hdr); err != nil {
		return errors.Wrap(err, "write header")
	}

	// Write the contents of regular files.
	if hdr.Typeflag == tar.TypeReg {
		fh, err := tg.fsEval.Open(path)
		if err != nil {
			return errors.Wrap(err, "open file")
		}
		defer fh.Close()

		n, err := io.Copy(tg.tw, fh)
		if err != nil {
			return errors.Wrap(err, "copy to layer")
		}
		if n != hdr.Size {
			return errors.Wrap(io.ErrShortWrite, "copy to layer")
		}
	}

	return nil
}

// whPrefix is the whiteout prefix, which is used to signify "special" files in
// an OCI image layer archive. An expanded filesystem image cannot contain
// files that have a basename starting with this prefix.
const whPrefix = ".wh."

// whOpaque is the *full* basename of a special file which indicates that all
// siblings in a directory are to be dropped in the "lower" layer.
const whOpaque = whPrefix + whPrefix + ".opq"

// addWhiteout adds a whiteout file for the given name inside the tar archive.
// It's not recommended to add a file with AddFile and then white it out. If
// you specify opaque, then the whiteout created is an opaque whiteout *for the
// directory path* given.
func (tg *tarGenerator) addWhiteout(name string, opaque bool) error {
	name, err := normalise(name, false)
	if err != nil {
		return errors.Wrap(err, "normalise path")
	}

	// Disallow having a whiteout of a whiteout, purely for our own sanity.
	dir, file := filepath.Split(name)
	if strings.HasPrefix(file, whPrefix) {
		return errors.Errorf("invalid path has whiteout prefix %q: %s", whPrefix, name)
	}

	// Figure out the whiteout name.
	whiteout := filepath.Join(dir, whPrefix+file)
	if opaque {
		whiteout = filepath.Join(name, whOpaque)
	}

	// Add a dummy header for the whiteout file.
	return errors.Wrap(tg.tw.WriteHeader(&tar.Header{
		Name: whiteout,
		Size: 0,
	}), "write whiteout header")
}

// AddWhiteout creates a whiteout for the provided path.
func (tg *tarGenerator) AddWhiteout(name string) error {
	return tg.addWhiteout(name, false)
}

// AddOpaqueWhiteout creates a whiteout for the provided path.
func (tg *tarGenerator) AddOpaqueWhiteout(name string) error {
	return tg.addWhiteout(name, true)
}
