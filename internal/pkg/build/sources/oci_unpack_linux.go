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
	"errors"
	"fmt"
	"os"

	"github.com/containers/image/types"
	"github.com/openSUSE/umoci"
	umocilayer "github.com/openSUSE/umoci/oci/layer"
	"github.com/openSUSE/umoci/pkg/idtools"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	sytypes "github.com/sylabs/singularity/pkg/build/types"
)

// unpackRootfs extracts all of the layers of the given image reference into the rootfs of the provided bundle
func unpackRootfs(ctx context.Context, b *sytypes.Bundle, tmpfsRef types.ImageReference, sysCtx *types.SystemContext) (err error) {
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

	engineExt, err := umoci.OpenLayout(b.TmpDir)
	if err != nil {
		return fmt.Errorf("error opening layout: %s", err)
	}

	// Obtain the manifest
	imageSource, err := tmpfsRef.NewImageSource(ctx, sysCtx)
	if err != nil {
		return fmt.Errorf("error creating image source: %s", err)
	}
	manifestData, mediaType, err := imageSource.GetManifest(ctx, nil)
	if err != nil {
		return fmt.Errorf("error obtaining manifest source: %s", err)
	}
	if mediaType != imgspecv1.MediaTypeImageManifest {
		return fmt.Errorf("error verifying manifest media type: %s", mediaType)
	}
	var manifest imgspecv1.Manifest
	json.Unmarshal(manifestData, &manifest)

	// UnpackRootfs from umoci v0.4.2 expects a path to a non-existing directory
	os.RemoveAll(b.RootfsPath)

	// Unpack root filesystem
	err = umocilayer.UnpackRootfs(ctx, engineExt, b.RootfsPath, manifest, &mapOptions)
	if err != nil {
		return fmt.Errorf("error unpacking rootfs: %s", err)
	}

	// If the `--fix-perms` flag was used, then modify the permissions so that
	// content has owner rwX and we're done
	if b.Opts.FixPerms {
		sylog.Warningf("The --fix-perms option modifies the filesystem permissions on the resulting container.")
		sylog.Debugf("Modifying permissions for file/directory owners")
		return fixPerms(b.RootfsPath)
	}

	// If `--fix-perms` was not used and this is a sandbox, scan for restrictive
	// perms that would stop the user doing an `rm` without a chmod first,
	// and warn if they exist
	if b.Opts.SandboxTarget {
		sylog.Debugf("Scanning for restrictive permissions")
		return checkPerms(b.RootfsPath)
	}

	// No `--fix-perms` and no sandbox... we are fine
	return err

}

// fixPerms will work through the rootfs of this bundle, making sure that all
// files and directories have permissions set such that the owner can read,
// modify, delete. This brings us to the situation of <=3.4
func fixPerms(rootfs string) (err error) {
	errors := 0
	err = fs.PermWalk(rootfs, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			sylog.Errorf("Unable to access rootfs path %s: %s", path, err)
			errors++
			return nil
		}

		switch mode := f.Mode(); {
		// Directories must have the owner 'rx' bits to allow traversal and reading on move, and the 'w' bit
		// so their content can be deleted by the user when the rootfs/sandbox is deleted
		case mode.IsDir():
			if err := os.Chmod(path, f.Mode().Perm()|0700); err != nil {
				sylog.Errorf("Error setting permission for %s: %s", path, err)
				errors++
			}
		case mode.IsRegular():
			// Regular files must have the owner 'r' bit so that everything can be read in order to
			// copy or move the rootfs/sandbox around. Also, the `w` bit as the build does write into
			// some files (e.g. resolv.conf) in the container rootfs.
			if err := os.Chmod(path, f.Mode().Perm()|0600); err != nil {
				sylog.Errorf("Error setting permission for %s: %s", path, err)
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

// checkPerms will work through the rootfs of this bundle, and find if any
// directory does not have owner rwX - which may cause unexpected issues for a
// user trying to look through, or delete a sandbox
func checkPerms(rootfs string) (err error) {
	// This is a locally defined error we can bubble up to cancel our recursive
	// structure.
	var errRestrictivePerm = errors.New("restrictive file permission found")

	err = fs.PermWalkRaiseError(rootfs, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			// If the walk function cannot access a directory at all, that's an
			// obvious restrictive permission we need to warn on
			if os.IsPermission(err) {
				sylog.Debugf("Path %q has restrictive permissions", path)
				return errRestrictivePerm
			}
			return fmt.Errorf("unable to access rootfs path %s: %s", path, err)
		}
		// Warn on any directory not `rwX` - technically other combinations may
		// be traversable / removable... but are confusing to the user vs
		// the Singularity 3.4 behavior.
		if f.Mode().IsDir() && f.Mode().Perm()&0700 != 0700 {
			sylog.Debugf("Path %q has restrictive permissions", path)
			return errRestrictivePerm
		}
		return nil
	})

	if errors.Is(err, errRestrictivePerm) {
		sylog.Warningf("Permission handling has changed in Singularity 3.5 for improved OCI compatibility")
		sylog.Warningf("The sandbox will contain files/dirs that cannot be removed until permissions are modified")
		sylog.Warningf("Use 'chmod -R u+rwX' to set permissions that allow removal")
		sylog.Warningf("Use the '--fix-perms' option to 'singularity build' to modify permissions at build time")
		sylog.Warningf("You can provide feedback about this change at https://github.com/sylabs/singularity/issues/4671")
		// It's not an error any further up... the rootfs is still usable
		return nil
	}
	return err
}
