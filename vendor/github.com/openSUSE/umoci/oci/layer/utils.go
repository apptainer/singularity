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
	"os"
	"path/filepath"

	"github.com/apex/log"
	"github.com/golang/protobuf/proto"
	"github.com/openSUSE/umoci/pkg/idtools"
	rspec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
	rootlesscontainers "github.com/rootless-containers/proto/go-proto"
)

// MapOptions specifies the UID and GID mappings used when unpacking and
// repacking images.
type MapOptions struct {
	// UIDMappings and GIDMappings are the UID and GID mappings to apply when
	// packing and unpacking image rootfs layers.
	UIDMappings []rspec.LinuxIDMapping `json:"uid_mappings"`
	GIDMappings []rspec.LinuxIDMapping `json:"gid_mappings"`

	// Rootless specifies whether any to error out if chown fails.
	Rootless bool `json:"rootless"`

	// KeepDirlinks is essentially the same as rsync's optio
	// --keep-dirlinks: if, on extraction, a directory would be created
	// where a symlink to a directory previously existed, KeepDirlinks
	// doesn't create that directory, but instead just uses the existing
	// symlink.
	KeepDirlinks bool `json:"-"`
}

// mapHeader maps a tar.Header generated from the filesystem so that it
// describes the inode as it would be observed by a container process. In
// particular this involves apply an ID mapping from the host filesystem to the
// container mappings. Returns an error if it's not possible to map the given
// UID.
func mapHeader(hdr *tar.Header, mapOptions MapOptions) error {
	var newUID, newGID int

	// It only makes sense to do un-mapping if we're not rootless. If we're
	// rootless then all of the files will be owned by us anyway.
	if !mapOptions.Rootless {
		var err error
		newUID, err = idtools.ToContainer(hdr.Uid, mapOptions.UIDMappings)
		if err != nil {
			return errors.Wrap(err, "map uid to container")
		}
		newGID, err = idtools.ToContainer(hdr.Gid, mapOptions.GIDMappings)
		if err != nil {
			return errors.Wrap(err, "map gid to container")
		}
	}

	// We have special handling for the "user.rootlesscontainers" xattr. If
	// we're rootless then we override the owner of the file we're currently
	// parsing (and then remove the xattr). If we're not rootless then the user
	// is doing something strange, so we log a warning but just ignore the
	// xattr otherwise.
	//
	// TODO: We should probably add a flag to opt-out of this (though I'm not
	//       sure why anyone would intentionally use this incorrectly).
	if value, ok := hdr.Xattrs[rootlesscontainers.Keyname]; !ok {
		// noop
	} else if !mapOptions.Rootless {
		log.Warnf("suspicious filesystem: saw special rootless xattr %s in non-rootless invocation", rootlesscontainers.Keyname)
	} else {
		var payload rootlesscontainers.Resource
		if err := proto.Unmarshal([]byte(value), &payload); err != nil {
			return errors.Wrap(err, "unmarshal rootlesscontainers payload")
		}

		// If the payload isn't uint32(-1) we apply it. The xattr includes the
		// *in-container* owner so we don't want to map it.
		if uid := payload.GetUid(); uid != rootlesscontainers.NoopID {
			newUID = int(uid)
		}
		if gid := payload.GetGid(); gid != rootlesscontainers.NoopID {
			newGID = int(gid)
		}

		// Drop the xattr since it's just a marker for us and shouldn't be in
		// layers. This is technically out-of-spec, but so is
		// "user.rootlesscontainers".
		delete(hdr.Xattrs, rootlesscontainers.Keyname)
	}

	hdr.Uid = newUID
	hdr.Gid = newGID
	return nil
}

// unmapHeader maps a tar.Header from a tar layer stream so that it describes
// the inode as it would be exist on the host filesystem. In particular this
// involves applying an ID mapping from the container filesystem to the host
// mappings. Returns an error if it's not possible to map the given UID.
func unmapHeader(hdr *tar.Header, mapOptions MapOptions) error {
	// To avoid nil references.
	if hdr.Xattrs == nil {
		hdr.Xattrs = make(map[string]string)
	}

	// If there is already a "user.rootlesscontainers" we give a warning in
	// both rootless and root cases -- but in rootless we explicitly delete the
	// entry because we might replace it.
	if _, ok := hdr.Xattrs[rootlesscontainers.Keyname]; ok {
		if mapOptions.Rootless {
			log.Warnf("rootless{%s} ignoring special xattr %s stored in layer", hdr.Name, rootlesscontainers.Keyname)
			delete(hdr.Xattrs, rootlesscontainers.Keyname)
		} else {
			log.Warnf("suspicious layer: saw special xattr %s in non-rootless invocation", rootlesscontainers.Keyname)
		}
	}

	// In rootless mode there are a few things we need to do. We need to map
	// all of the files in the layer to have an owner of (0, 0) because we
	// cannot lchown(2) anything -- and then if the owner was non-root we have
	// to create a "user.rootlesscontainers" xattr for it.
	if mapOptions.Rootless {
		// Fill the rootlesscontainers payload with the original (uid, gid). If
		// either is 0, we replace it with uint32(-1). Technically we could
		// just leave it as 0 (since that is what the source of truth told us
		// the owner was), but this would result in a massive increase in
		// xattrs with no real benefit.
		payload := rootlesscontainers.Resource{
			Uid: rootlesscontainers.NoopID,
			Gid: rootlesscontainers.NoopID,
		}
		if uid := hdr.Uid; uid != 0 {
			payload.Uid = uint32(uid)
		}
		if gid := hdr.Gid; gid != 0 {
			payload.Gid = uint32(gid)
		}

		// Don't add the xattr if the owner isn't just (0, 0) because that's a
		// waste of space.
		if !rootlesscontainers.IsDefault(payload) {
			valueBytes, err := proto.Marshal(&payload)
			if err != nil {
				return errors.Wrap(err, "marshal rootlesscontainers payload")
			}
			// While the payload is almost certainly not UTF-8, Go strings can
			// actually be arbitrary bytes (in case you didn't know this and
			// were confused like me when this worked). See
			// <https://blog.golang.org/strings> for more detail.
			hdr.Xattrs[rootlesscontainers.Keyname] = string(valueBytes)
		}

		hdr.Uid = 0
		hdr.Gid = 0
	}

	newUID, err := idtools.ToHost(hdr.Uid, mapOptions.UIDMappings)
	if err != nil {
		return errors.Wrap(err, "map uid to host")
	}
	newGID, err := idtools.ToHost(hdr.Gid, mapOptions.GIDMappings)
	if err != nil {
		return errors.Wrap(err, "map gid to host")
	}

	hdr.Uid = newUID
	hdr.Gid = newGID
	return nil
}

// CleanPath makes a path safe for use with filepath.Join. This is done by not
// only cleaning the path, but also (if the path is relative) adding a leading
// '/' and cleaning it (then removing the leading '/'). This ensures that a
// path resulting from prepending another path will always resolve to lexically
// be a subdirectory of the prefixed path. This is all done lexically, so paths
// that include symlinks won't be safe as a result of using CleanPath.
//
// This function comes from runC (libcontainer/utils/utils.go).
func CleanPath(path string) string {
	// Deal with empty strings nicely.
	if path == "" {
		return ""
	}

	// Ensure that all paths are cleaned (especially problematic ones like
	// "/../../../../../" which can cause lots of issues).
	path = filepath.Clean(path)

	// If the path isn't absolute, we need to do more processing to fix paths
	// such as "../../../../<etc>/some/path". We also shouldn't convert absolute
	// paths to relative ones.
	if !filepath.IsAbs(path) {
		path = filepath.Clean(string(os.PathSeparator) + path)
		// This can't fail, as (by definition) all paths are relative to root.
		path, _ = filepath.Rel(string(os.PathSeparator), path)
	}

	// Clean the path again for good measure.
	return filepath.Clean(path)
}

// InnerErrno returns the "real" system error from an error that originally
// came from the "os" package. The returned error can be compared directly with
// unix.* (or syscall.*) errno values. If the type could not be detected we just return
func InnerErrno(err error) error {
	// All of the os.* cases as well as an explicit
	errno := errors.Cause(err)
	switch err := errno.(type) {
	case *os.PathError:
		errno = err.Err
	case *os.LinkError:
		errno = err.Err
	case *os.SyscallError:
		errno = err.Err
	}
	return errno
}
