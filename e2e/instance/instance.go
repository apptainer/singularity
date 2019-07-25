// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package instance

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/pkg/errors"
	"github.com/sylabs/singularity/e2e/internal/e2e"
)

// Test that no instances are running.
func testNoInstances(ec *e2e.TestContext) {
	expectedNumberOfInstances(ec, 0)
}

// Test that a basic echo server instance can be started, communicated with,
// and stopped.
func testBasicEchoServer(ctx *e2e.TestContext) {
	t, env, profile := ctx.Get()

	const instanceName = "echo1"

	// Start the instance.
	env.RunSingularity(
		t,
		e2e.WithProfile(profile),
		e2e.WithCommand("instance start"),
		e2e.WithArgs(env.ImagePath, instanceName, strconv.Itoa(instanceStartPort)),
		e2e.PostRun(func(t *testing.T) {
			if t.Failed() {
				return
			}
			// Try to contact the instance.
			echo(t, instanceStartPort)
			stopInstance(ctx, instanceName)
		}),
		e2e.ExpectExit(0),
	)
}

// Test creating many instances, but don't stop them.
func testCreateManyInstances(ctx *e2e.TestContext) {
	t, env, profile := ctx.Get()

	const n = 10

	// Start n instances.
	for i := 0; i < n; i++ {
		port := instanceStartPort + i
		instanceName := "echo" + strconv.Itoa(i+1)

		env.RunSingularity(
			t,
			e2e.WithProfile(profile),
			e2e.WithCommand("instance start"),
			e2e.WithArgs(env.ImagePath, instanceName, strconv.Itoa(port)),
			e2e.PostRun(func(t *testing.T) {
				echo(t, port)
			}),
			e2e.ExpectExit(0),
		)
	}

	// Verify all instances started.
	expectedNumberOfInstances(ctx, n)
}

// Test stopping all running instances.
func testStopAll(ctx *e2e.TestContext) {
	stopInstance(ctx, "", "--all")
}

// Test basic options like mounting a custom home directory, changing the
// hostname, etc.
func testBasicOptions(ctx *e2e.TestContext) {
	t, env, profile := ctx.Get()

	const fileName = "hello"
	const instanceName = "testbasic"
	const testHostname = "echoserver99"
	fileContents := []byte("world")

	// Create a temporary directory to serve as a home directory.
	dir, err := ioutil.TempDir(env.TestDir, "TestInstance")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(dir)

	// Create and populate a temporary file.
	tempFile := filepath.Join(dir, fileName)
	err = ioutil.WriteFile(tempFile, fileContents, 0644)
	err = errors.Wrapf(err, "creating temporary test file %s", tempFile)
	if err != nil {
		t.Fatalf("Failed to create file: %+v", err)
	}

	// Start an instance with the temporary directory as the home directory.
	env.RunSingularity(
		t,
		e2e.WithProfile(profile),
		e2e.WithCommand("instance start"),
		e2e.WithArgs(
			"-H", dir+":/home/temp",
			"--hostname", testHostname,
			"-e",
			env.ImagePath,
			instanceName,
			strconv.Itoa(instanceStartPort),
		),
		e2e.PostRun(func(t *testing.T) {
			if t.Failed() {
				return
			}

			// Verify we can see the file's contents from within the container.
			stdout, _, success := execInstance(ctx, instanceName, "cat", "/home/temp/"+fileName)
			if success && !bytes.Equal(fileContents, []byte(stdout)) {
				t.Errorf("File contents were %s, but expected %s", stdout, string(fileContents))
			}

			// Verify that the hostname has been set correctly.
			stdout, _, success = execInstance(ctx, instanceName, "hostname")
			if success && !bytes.Equal([]byte(testHostname+"\n"), []byte(stdout)) {
				t.Errorf("Hostname is %s, but expected %s", stdout, testHostname)
			}

			// Stop the instance.
			stopInstance(ctx, instanceName)
		}),
		e2e.ExpectExit(0),
	)
}

// Test that contain works.
func testContain(ctx *e2e.TestContext) {
	t, env, profile := ctx.Get()

	const instanceName = "testcontain"
	const fileName = "thegreattestfile"

	// Create a temporary directory to serve as a contain directory.
	dir, err := ioutil.TempDir("", "TestInstance")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(dir)

	// Start the instance.
	env.RunSingularity(
		t,
		e2e.WithProfile(profile),
		e2e.WithCommand("instance start"),
		e2e.WithArgs(
			"-c",
			"-W", dir,
			env.ImagePath,
			instanceName,
			strconv.Itoa(instanceStartPort),
		),
		e2e.PostRun(func(t *testing.T) {
			if t.Failed() {
				return
			}

			// Touch a file within /tmp.
			_, _, success := execInstance(ctx, instanceName, "touch", "/tmp/"+fileName)
			if success {
				// Verify that the touched file exists outside the container.
				if _, err = os.Stat(filepath.Join(dir, "tmp", fileName)); os.IsNotExist(err) {
					t.Errorf("The temp file doesn't exist.")
				}
			}

			// Stop the container.
			stopInstance(ctx, instanceName)
		}),
		e2e.ExpectExit(0),
	)
}

// Test by running directly from URI
func testInstanceFromURI(ctx *e2e.TestContext) {
	t, env, profile := ctx.Get()

	instances := []struct {
		name string
		uri  string
	}{
		{
			name: "test_from_docker",
			uri:  "docker://busybox",
		},
		{
			name: "test_from_library",
			uri:  "library://busybox",
		},
		// TODO(mem): reenable this; disabled while shub is down
		// {
		// 	name: "test_from_shub",
		// 	uri:  "shub://singularityhub/busybox",
		// },
	}

	for _, i := range instances {
		env.RunSingularity(
			t,
			e2e.WithProfile(profile),
			e2e.WithCommand("instance start"),
			e2e.WithArgs(i.uri, i.name),
			e2e.PostRun(func(t *testing.T) {
				if t.Failed() {
					return
				}
				execInstance(ctx, i.name, "id")
				stopInstance(ctx, i.name)
			}),
			e2e.ExpectExit(0),
		)
	}
}

// RunE2ETests is the bootstrap to run all instance tests.
func RunE2ETests(env e2e.TestEnv) func(*testing.T) {
	return func(t *testing.T) {
		e2e.EnsureImage(t, env)

		// Define and loop through tests.
		// Execution order matter.
		tests := []struct {
			name string
			fn   func(*e2e.TestContext)
		}{
			{
				name: "InitialNoInstances",
				fn:   testNoInstances,
			},
			{
				name: "BasicEchoServer",
				fn:   testBasicEchoServer,
			},
			{
				name: "BasicOptions",
				fn:   testBasicOptions,
			},
			{
				name: "Contain",
				fn:   testContain,
			},
			{
				name: "InstanceFromURI",
				fn:   testInstanceFromURI,
			},
			{
				name: "CreateManyInstances",
				fn:   testCreateManyInstances,
			},
			{
				name: "StopAll",
				fn:   testStopAll,
			},
			{
				name: "FinalNoInstances",
				fn:   testNoInstances,
			},
		}

		for _, profile := range e2e.Profiles {
			t.Run(profile.Name(), func(t *testing.T) {
				profile.Require(t)

				for _, tt := range tests {
					t.Run(tt.name, func(t *testing.T) {
						ctx := e2e.NewTestContext(t, env, profile)
						tt.fn(ctx)
					})
				}
			})
		}
	}
}
