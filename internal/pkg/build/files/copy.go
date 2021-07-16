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

	"github.com/sylabs/singularity/internal/pkg/util/fs"
)

// makeParentDir ensures existence of the expected destination directory for the cp command
// based on the supplied path and the number of source paths to copy
func makeParentDir(path string, numSrcPaths int) error {
	_, err := os.Stat(path)
	if !os.IsNotExist(err) {
		return nil
	}

	// if path ends with a trailing '/' or if there are multiple source paths to copy
	// always ensure the full path exists as a directory because 'cp' is expecting a
	// dir in these cases
	if strings.HasSuffix(path, "/") || numSrcPaths > 1 {
		if err := os.MkdirAll(filepath.Clean(path), 0755); err != nil {
			return fmt.Errorf("while creating full path: %s", err)
		}
	}

	// only make parent directory
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("while creating parent of path: %s", err)
	}

	return nil
}

// CopyFromHost should be used to copy files into the rootfs from the host fs.
// src is a path relative to CWD on the host, or an absolute path on the host.
// dstRel is a destination path inside dstRootfs
// All symlinks encountered in the copy will be dereferenced (cp -L behavior).
func CopyFromHost(src, dstRel, dstRootfs string) error {
	// resolve any bash globbing in filepath
	paths, err := expandPath(src)
	if err != nil {
		return fmt.Errorf("while expanding source path with bash: %s: %s", src, err)
	}

	// Resolve our destination within the container rootfs
	dstResolved, err := secureJoinKeepSlash(dstRootfs, dstRel)
	if err != nil {
		return fmt.Errorf("while resolving destination: %s: %s", dstRel, err)
	}

	// Create any parent dirs for dst that don't already exist
	if err := makeParentDir(dstResolved, len(paths)); err != nil {
		return fmt.Errorf("while creating parent dir: %v", err)
	}

	args := []string{"-fLr"}
	// append file(s) to be copied
	args = append(args, paths...)
	// append dst as last arg
	args = append(args, dstResolved)

	var output, stderr bytes.Buffer
	// copy each file into bundle rootfs
	copy := exec.Command("/bin/cp", args...)
	copy.Stdout = &output
	copy.Stderr = &stderr
	if err := copy.Run(); err != nil {
		return fmt.Errorf("while copying %s to %s: %s: %s", paths, dstResolved, err, stderr.String())
	}
	return nil
}

// CopyFromStage should be used to copy files into the rootfs from a previous stage.
// The srcRel and dstRel are src / dst paths relative to the srcRootfs and dstRootfs.
// Symlinks are only dereferenced for the specified source or files that resolve
// directly from a specified glob pattern. Any additional links inside a directory
// being copied are not dereferenced.
func CopyFromStage(srcRel, dstRel, srcRootfs, dstRootfs string) error {
	// An absolute path is required for globbing... but with no symlink resolution or
	// path cleaning yet.
	srcAbs := joinKeepSlash(srcRootfs, srcRel)

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
		// If we are copying into a directory then we must use the original source filename,
		// for the destination filename, not the one that was resolved out.
		// I.E. if copying `/opt/view` to `/opt/` where `/opt/view links-> /opt/.view/abc123`
		// we want to create `/opt/view` in the dest, not `/opt/abc123`.
		dstResolved, err := secureJoinKeepSlash(dstRootfs, dstRel)
		if err != nil {
			return fmt.Errorf("while resolving destination: %s: %s", dstRel, err)
		}
		if fs.IsDir(dstResolved) {
			_, srcName := path.Split(srcGlobbedRel)
			dstResolved = path.Join(dstResolved, srcName)
		}

		// Create any parent dirs for dst that don't already exist.
		if err := makeParentDir(dstResolved, len(paths)); err != nil {
			return fmt.Errorf("while creating parent dir: %v", err)
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
