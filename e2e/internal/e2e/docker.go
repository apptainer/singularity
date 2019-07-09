// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import (
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

const dockerInstanceName = "e2e-docker-instance"

var registrySetup struct {
	sync.Once
	up uint32 // 1 if the registry is running, 0 otherwise
}

// PrepRegistry run a docker registry and push a busybox
// image and the test image with oras transport.
func PrepRegistry(t *testing.T, env TestEnv) {
	registrySetup.Do(func() {
		atomic.StoreUint32(&registrySetup.up, 1)

		EnsureImage(t, env)

		dockerDefinition := "testdata/Docker_registry.def"
		dockerImage := filepath.Join(env.TestDir, "docker-e2e.sif")

		RunSingularity(
			t,
			"BuildDockerRegistry",
			WithoutSubTest(),
			WithPrivileges(true),
			WithCommand("build"),
			WithArgs("-s", dockerImage, dockerDefinition),
			ExpectExit(0),
		)

		RunSingularity(
			t,
			"RunDockerRegistry",
			WithoutSubTest(),
			WithPrivileges(true),
			WithCommand("instance start"),
			WithArgs("-w", "-B", "/sys", dockerImage, dockerInstanceName),
			ExpectExit(0),
		)

		RunSingularity(
			t,
			"OrasPushTestImage",
			WithoutSubTest(),
			WithCommand("push"),
			WithArgs(env.ImagePath, env.OrasTestImage),
			ExpectExit(0),
		)
	})
}

// KillRegistry stop and cleanup docker registry.
func KillRegistry(t *testing.T) {
	if !atomic.CompareAndSwapUint32(&registrySetup.up, 1, 0) {
		return
	}

	RunSingularity(
		t,
		"KillDockerRegistry",
		WithoutSubTest(),
		WithPrivileges(true),
		WithCommand("instance stop"),
		WithArgs("-s", "KILL", dockerInstanceName),
		ExpectExit(0),
	)
}
