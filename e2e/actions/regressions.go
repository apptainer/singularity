// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package actions

import (
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
)

// Check there is no file descriptor leaked in the container
// process. This test expect 4 file descriptors, 3 for stdin,
// stdout, stderr and one opened by the ls command.
func (c actionTests) issue4488(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("exec"),
		e2e.WithArgs(c.env.ImagePath, "ls", "-1", "/proc/self/fd"),
		e2e.ExpectExit(
			0,
			e2e.ExpectOutput(e2e.ExactMatch, "0\n1\n2\n3"),
		),
	)
}
