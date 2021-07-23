// Copyright (c) 2020-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package instance

import (
	"fmt"
	"testing"

	"github.com/hpcng/singularity/e2e/internal/e2e"
	uuid "github.com/satori/go.uuid"
)

func (c *ctx) issue5033(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	c.profile = e2e.RootProfile

	// pick up a random name
<<<<<<< HEAD
	instanceName := uuid.Must(uuid.NewV4()).String()
=======
	instanceName := randomName(t)
>>>>>>> 78f8778... fix: update code to account for uuid module breaking changes
	joinName := fmt.Sprintf("instance://%s", instanceName)

	c.env.RunSingularity(
		t,
		e2e.WithProfile(c.profile),
		e2e.WithCommand("instance start"),
		e2e.WithArgs("--boot", c.env.ImagePath, instanceName),
		e2e.ExpectExit(0),
	)

	c.env.RunSingularity(
		t,
		e2e.WithProfile(c.profile),
		e2e.WithCommand("exec"),
		e2e.WithArgs(joinName, "/bin/true"),
		e2e.ExpectExit(0),
	)

	c.stopInstance(t, instanceName)
}
