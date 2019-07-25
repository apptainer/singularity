// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import (
	"net"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pkg/errors"
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

		env.RunSingularity(
			t,
			WithPrivileges(true),
			WithCommand("build"),
			WithArgs("-s", dockerImage, dockerDefinition),
			ExpectExit(0),
		)

		var umountFn func(*testing.T)

		env.RunSingularity(
			t,
			WithPrivileges(true),
			WithCommand("instance start"),
			WithArgs("-w", "-B", "/sys", dockerImage, dockerInstanceName),
			PreRun(func(t *testing.T) {
				umountFn = shadowInstanceDirectory(t, env)
			}),
			PostRun(func(t *testing.T) {
				if umountFn != nil {
					umountFn(t)
				}
			}),
			ExpectExit(0),
		)

		// start script in e2e/testdata/Docker_registry.def will listen
		// on port 5111 once docker registry is up and initialized, so
		// we are trying to connect to this port until we got a response,
		// without any response after 10 seconds we abort tests execution
		// because the start script probably failed
		retry := 0
		for {
			conn, err := net.Dial("tcp", "127.0.0.1:5111")
			err = errors.Wrap(err, "connecting to test endpoint in docker registry container")
			if err == nil {
				conn.Close()
				break
			}
			time.Sleep(100 * time.Millisecond)
			retry++
			if retry == 100 {
				t.Fatalf("docker registry unreachable after 10 seconds: %+v", err)
			}
		}

		env.RunSingularity(
			t,
			WithCommand("push"),
			WithArgs(env.ImagePath, env.OrasTestImage),
			ExpectExit(0),
		)
	})
}

// KillRegistry stop and cleanup docker registry.
func KillRegistry(t *testing.T, env TestEnv) {
	if !atomic.CompareAndSwapUint32(&registrySetup.up, 1, 0) {
		return
	}

	var umountFn func(*testing.T)

	env.RunSingularity(
		t,
		WithPrivileges(true),
		WithCommand("instance stop"),
		WithArgs("-s", "KILL", dockerInstanceName),
		PreRun(func(t *testing.T) {
			umountFn = shadowInstanceDirectory(t, env)
		}),
		PostRun(func(t *testing.T) {
			if umountFn != nil {
				umountFn(t)
			}
		}),
		ExpectExit(0),
	)
}
