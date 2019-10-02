// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// TODO(ian): The build package should be refactored to make each conveyorpacker
// its own separate package. With that change, this file should be grouped with the
// OCIConveyorPacker code

package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/containers/image/types"
	"github.com/openSUSE/umoci"
	umocilayer "github.com/openSUSE/umoci/oci/layer"
	"github.com/openSUSE/umoci/pkg/idtools"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	sytypes "github.com/sylabs/singularity/pkg/build/types"
)

// unpackRootfs extracts all of the layers of the given image reference into the rootfs of the provided bundle
func unpackRootfs(b *sytypes.Bundle, tmpfsRef types.ImageReference, sysCtx *types.SystemContext) (err error) {
	var mapOptions umocilayer.MapOptions

	// Allow unpacking as non-root
	if os.Geteuid() != 0 {
		mapOptions.Rootless = true

		uidMap, err := idtools.ParseMapping(fmt.Sprintf("0:%d:1", os.Geteuid()))
		if err != nil {
			return fmt.Errorf("error parsing uidmap: %s", err)
		}
		mapOptions.UIDMappings = append(mapOptions.UIDMappings, uidMap)

		gidMap, err := idtools.ParseMapping(fmt.Sprintf("0:%d:1", os.Getegid()))
		if err != nil {
			return fmt.Errorf("error parsing gidmap: %s", err)
		}
		mapOptions.GIDMappings = append(mapOptions.GIDMappings, gidMap)
	}

	engineExt, err := umoci.OpenLayout(b.Path)
	if err != nil {
		return fmt.Errorf("error opening layout: %s", err)
	}

	// Obtain the manifest
	imageSource, err := tmpfsRef.NewImageSource(context.Background(), sysCtx)
	if err != nil {
		return fmt.Errorf("error creating image source: %s", err)
	}
	manifestData, mediaType, err := imageSource.GetManifest(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("error obtaining manifest source: %s", err)
	}
	if mediaType != imgspecv1.MediaTypeImageManifest {
		return fmt.Errorf("error verifying manifest media type: %s", mediaType)
	}
	var manifest imgspecv1.Manifest
	json.Unmarshal(manifestData, &manifest)

	// UnpackRootfs from umoci v0.4.2 expects a path to a non-existing directory
	os.RemoveAll(b.Rootfs())

	// Unpack root filesystem
	err = umocilayer.UnpackRootfs(context.Background(), engineExt, b.Rootfs(), manifest, &mapOptions)
	if err != nil {
		return fmt.Errorf("error unpacking rootfs: %s", err)
	}

	// If this is a rootless extraction we need to mangle permissions to fix #4524. This
	// returns to the <=3.3 permissions on the rootfs, with the exception that umoci
	// correctly applies permission changes across layers when extracting.
	if mapOptions.Rootless {
		sylog.Debugf("Modifying rootless permissions on temporary rootfs")
		return fixPermsRootless(b.Rootfs())
	}

	return nil
}

// fixPermsRootless forces permissions on the rootfs so that it can be easily
// moved and deleted by a non-root user owner.
func fixPermsRootless(rootfs string) (err error) {
	errors := 0
	err = permWalk(rootfs, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			sylog.Errorf("Unable to access sandbox path %s: %s", path, err)
			errors++
			return nil
		}
		// Directories must have the owner 'rx' bits to allow traversal and reading on move, and the 'w' bit
		// so their content can be deleted by the user when the rootfs/sandbox is deleted
		switch mode := f.Mode(); {
		case mode.IsDir():
			if err := os.Chmod(path, f.Mode().Perm()|0700); err != nil {
				sylog.Errorf("Error setting rootless permission for %s: %s", path, err)
				errors++
			}
		case mode.IsRegular():
			// Regular files must have the owner 'r' bit so that everything can be read in order to
			// copy or move the rootfs/sandbox around. Also, the `w` bit as the build does write into
			// some files (e.g. resolv.conf) in the container rootfs.
			if err := os.Chmod(path, f.Mode().Perm()|0600); err != nil {
				sylog.Errorf("Error setting rootless permission for %s: %s", path, err)
				errors++
			}
		}
		return nil
	})

	if errors > 0 {
		err = fmt.Errorf("%d errors were encountered when setting permissions", errors)
	}

	return err
}

// permWalk is similar to os.Walk - but:
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
