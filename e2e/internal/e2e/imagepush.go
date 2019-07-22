// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

// ImagePush executes a singularity push command to push
// an image to the specified URI.
func ImagePush(t *testing.T, cmdPath SingularityCmdPath, imagePath, imgURI string) (string, []byte, error) {
	argv := []string{"push"}

	if imagePath != "" {
		argv = append(argv, imagePath)
	}

	argv = append(argv, imgURI)

	cmd := fmt.Sprintf("%s %s", cmdPath, strings.Join(argv, " "))
	out, err := exec.Command(string(cmdPath), argv...).CombinedOutput()

	return cmd, out, err

}
