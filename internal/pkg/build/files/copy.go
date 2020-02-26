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
	"path/filepath"
	"strings"
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

// Copy calls cp with src and dst as its arguments
// Checks dst and creates parent directories if they do not exist
// before calling cp.
// If followLinks is true, the -L flag to cp will follow all symlinks
// If followLinks is false, the -H flag to cp will only follow links for specified
// files or files that resolve directly from a glob pattern. It will not follow
// links found during directory traversal.
func Copy(src, dst string, followLinks bool) error {
	// resolve any bash globbing in filepath
	paths, err := expandPath(src)
	if err != nil {
		return fmt.Errorf("while expanding source path with bash: %s: %s", src, err)
	}

	if err := makeParentDir(dst, len(paths)); err != nil {
		return fmt.Errorf("while creating parent dir: %v", err)
	}

	// set flags for cp
	args := []string{"-fHr"}
	if followLinks {
		args = []string{"-fLr"}
	}
	// append file(s) to be copied
	args = append(args, paths...)
	// append dst as last arg
	args = append(args, dst)

	var output, stderr bytes.Buffer
	// copy each file into bundle rootfs
	copy := exec.Command("/bin/cp", args...)
	copy.Stdout = &output
	copy.Stderr = &stderr
	if err := copy.Run(); err != nil {
		return fmt.Errorf("while copying %s to %s: %s: %s", paths, dst, err, stderr.String())
	}
	return nil
}
