// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package copy

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Copy calls cp with src and dst as its arguments
// checks dst and creates parent directories if they do not exist
// before calling cp
func Copy(src, dst string) error {
	_, err := os.Stat(dst)
	if os.IsNotExist(err) {
		// if destination doesn't exist, create parent directories
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return fmt.Errorf("while creating parent directories: %v", err)
		}
	}

	var output, stderr bytes.Buffer
	// copy each file into bundle rootfs
	copy := exec.Command("/bin/cp", "-fLr", src, dst)
	copy.Stdout = &output
	copy.Stderr = &stderr
	if err := copy.Run(); err != nil {
		return fmt.Errorf("while copying %v to %v: %v: %v", src, dst, err, stderr.String())
	}
	return nil
}
