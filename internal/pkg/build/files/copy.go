// Copyright (c) 2019-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package files

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/hpcng/singularity/internal/pkg/util/fs"
)

// makeParentDir ensures existence of the expected destination directory for the cp command
// based on the supplied path.
func makeParentDir(path string) error {
	_, err := os.Stat(path)
	if !os.IsNotExist(err) {
		return nil
	}

	// if path ends with a trailing '/' always ensure the full path exists as a directory
	// because 'cp' is expecting a dir in these cases
	if strings.HasSuffix(path, "/") {
		if err := os.MkdirAll(filepath.Clean(path), 0755); err != nil {
			return fmt.Errorf("while creating full path: %s", err)
		}
		return nil
	}

	// only make parent directory
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("while creating parent of path: %s", err)
	}

	return nil
}

// CopyFromHost should be used to copy files into the rootfs from the host fs.
// src is a path relative to CWD on the host, or an absolute path on the host.
// dstRel is a destination path inside dstRootfs.
// An empty dstRel "" means copy the src file to the same path in the rootfs.
// All symlinks encountered in the copy will be dereferenced (cp -L behavior).
func CopyFromHost(src, dstRel, dstRootfs string) error {
	// resolve any bash globbing in filepath
	paths, err := expandPath(src)
	if err != nil {
		return fmt.Errorf("while expanding source path with bash: %s: %s", src, err)
	}

	for _, srcGlobbed := range paths {
		// If the dstRel is "" then we are copying to the full source path, appended to the rootfs prefix
		dstRelGlobbed := dstRel
		if dstRel == "" {
			dstRelGlobbed = srcGlobbed
		}

		// Resolve our destination within the container rootfs
		dstResolved, err := secureJoinKeepSlash(dstRootfs, dstRelGlobbed)
		if err != nil {
			return fmt.Errorf("while resolving destination: %s: %s", dstRelGlobbed, err)
		}

		// Create any parent dirs for dst that don't already exist
		if err := makeParentDir(dstResolved); err != nil {
			return fmt.Errorf("while creating parent dir: %v", err)
		}

		args := []string{"-fLr", srcGlobbed, dstResolved}
		var output, stderr bytes.Buffer
		// copy each file into bundle rootfs
		copy := exec.Command("/bin/cp", args...)
		copy.Stdout = &output
		copy.Stderr = &stderr
		if err := copy.Run(); err != nil {
			return fmt.Errorf("while copying %s to %s: %v: %s", paths, dstResolved, args, stderr.String())
		}

	}
	return nil
}

// CopyFromStage should be used to copy files into the rootfs from a previous stage.
// The src and dst are paths relative to the srcRootfs and dstRootfs.
// An empty dst "" means copy the src file to the same path in the dst rootfs.
// Symlinks are only dereferenced for the specified source or files that resolve
// directly from a specified glob pattern. Any additional links inside a directory
// being copied are not dereferenced.
func CopyFromStage(src, dst, srcRootfs, dstRootfs string) error {
	// An absolute path is required for globbing... but with no symlink resolution or
	// path cleaning yet.
	srcAbs := joinKeepSlash(srcRootfs, src)

	// resolve any bash globbing in filepath
	paths, err := expandPath(srcAbs)
	if err != nil {
		return fmt.Errorf("while expanding source path with bash: %s: %s", srcAbs, err)
	}

	// We manually dereference first-level src symlinks only.
	for _, srcGlobbed := range paths {
		// Now re-resolve the source files after globbing by using securejoin,
		// so that absolute symlinks are dereferenced relative to the source rootfs,
		// and the source is enforced to be inside the rootfs.
		srcGlobbedRel := strings.TrimPrefix(srcGlobbed, srcRootfs)
		srcResolved, err := secureJoinKeepSlash(srcRootfs, srcGlobbedRel)
		if err != nil {
			return fmt.Errorf("while resolving source: %s: %s", srcGlobbedRel, err)
		}

		// If the dst is "" then we are copying to the same path in dstRootfs, as src is in srcRootfs.
		dstGlobbed := dst
		if dst == "" {
			dstGlobbed = srcGlobbedRel
		}
		// Resolve the destination path, keeping any final slash
		dstResolved, err := secureJoinKeepSlash(dstRootfs, dstGlobbed)
		if err != nil {
			return fmt.Errorf("while resolving destination: %s: %s", dstGlobbed, err)
		}
		// Create any parent dirs for dstResolved that don't already exist.
		if err := makeParentDir(dstResolved); err != nil {
			return fmt.Errorf("while creating parent dir: %v", err)
		}

		// If we are copying into a directory then we must use the original source filename,
		// for the destination filename, not the one that was resolved out.
		// I.E. if copying `/opt/view` to `/opt/` where `/opt/view links-> /opt/.view/abc123`
		// we want to create `/opt/view` in the dest, not `/opt/abc123`.
		if fs.IsDir(dstResolved) {
			_, srcName := path.Split(srcGlobbedRel)
			dstResolved = path.Join(dstResolved, srcName)
		}

		// Set flags for cp to perform a recursive copy without further symlink dereference.
		args := []string{"-fr", srcResolved, dstResolved}
		var output, stderr bytes.Buffer
		copy := exec.Command("/bin/cp", args...)
		copy.Stdout = &output
		copy.Stderr = &stderr
		if err := copy.Run(); err != nil {
			return fmt.Errorf("while copying %s to %s: %s: %s", paths, dstResolved, err, stderr.String())
		}
	}
	return nil
}
