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

type ctx struct {
	env     e2e.TestEnv
	profile e2e.Profile
}

// Test that no instances are running.
func (c *ctx) testNoInstances(t *testing.T) {
	c.expectedNumberOfInstances(t, 0)
}

// Test that a basic echo server instance can be started, communicated with,
// and stopped.
func (c *ctx) testBasicEchoServer(t *testing.T) {
	const instanceName = "echo1"

	args := []string{c.env.ImagePath, instanceName, strconv.Itoa(instanceStartPort)}

	// Start the instance.
	c.env.RunSingularity(
		t,
		e2e.WithProfile(c.profile),
		e2e.WithCommand("instance start"),
		e2e.WithArgs(args...),
		e2e.PostRun(func(t *testing.T) {
			if t.Failed() {
				return
			}
			// Try to contact the instance.
			echo(t, instanceStartPort)
			c.stopInstance(t, instanceName)
		}),
		e2e.ExpectExit(0),
	)
}

// Test creating many instances, but don't stop them.
func (c *ctx) testCreateManyInstances(t *testing.T) {
	const n = 10

	// Start n instances.
	for i := 0; i < n; i++ {
		port := instanceStartPort + i
		instanceName := "echo" + strconv.Itoa(i+1)

		c.env.RunSingularity(
			t,
			e2e.WithProfile(c.profile),
			e2e.WithCommand("instance start"),
			e2e.WithArgs(c.env.ImagePath, instanceName, strconv.Itoa(port)),
			e2e.PostRun(func(t *testing.T) {
				echo(t, port)
			}),
			e2e.ExpectExit(0),
		)
	}

	// Verify all instances started.
	c.expectedNumberOfInstances(t, n)
}

// Test stopping all running instances.
func (c *ctx) testStopAll(t *testing.T) {
	c.stopInstance(t, "", "--all")
}

// Test basic options like mounting a custom home directory, changing the
// hostname, etc.
func (c *ctx) testBasicOptions(t *testing.T) {
	const fileName = "hello"
	const instanceName = "testbasic"
	const testHostname = "echoserver99"
	fileContents := []byte("world")

	// Create a temporary directory to serve as a home directory.
	dir, err := ioutil.TempDir(c.env.TestDir, "TestInstance")
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
	c.env.RunSingularity(
		t,
		e2e.WithProfile(c.profile),
		e2e.WithCommand("instance start"),
		e2e.WithArgs(
			"-H", dir+":/home/temp",
			"--hostname", testHostname,
			"-e",
			c.env.ImagePath,
			instanceName,
			strconv.Itoa(instanceStartPort),
		),
		e2e.PostRun(func(t *testing.T) {
			if t.Failed() {
				return
			}

			// Verify we can see the file's contents from within the container.
			stdout, _, success := c.execInstance(t, instanceName, "cat", "/home/temp/"+fileName)
			if success && !bytes.Equal(fileContents, []byte(stdout)) {
				t.Errorf("File contents were %s, but expected %s", stdout, string(fileContents))
			}

			// Verify that the hostname has been set correctly.
			stdout, _, success = c.execInstance(t, instanceName, "hostname")
			if success && !bytes.Equal([]byte(testHostname+"\n"), []byte(stdout)) {
				t.Errorf("Hostname is %s, but expected %s", stdout, testHostname)
			}

			// Stop the instance.
			c.stopInstance(t, instanceName)
		}),
		e2e.ExpectExit(0),
	)
}

// Test that contain works.
func (c *ctx) testContain(t *testing.T) {
	const instanceName = "testcontain"
	const fileName = "thegreattestfile"

	// Create a temporary directory to serve as a contain directory.
	dir, err := ioutil.TempDir("", "TestInstance")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(dir)

	// Start the instance.
	c.env.RunSingularity(
		t,
		e2e.WithProfile(c.profile),
		e2e.WithCommand("instance start"),
		e2e.WithArgs(
			"-c",
			"-W", dir,
			c.env.ImagePath,
			instanceName,
			strconv.Itoa(instanceStartPort),
		),
		e2e.PostRun(func(t *testing.T) {
			if t.Failed() {
				return
			}

			// Touch a file within /tmp.
			_, _, success := c.execInstance(t, instanceName, "touch", "/tmp/"+fileName)
			if success {
				// Verify that the touched file exists outside the container.
				if _, err = os.Stat(filepath.Join(dir, "tmp", fileName)); os.IsNotExist(err) {
					t.Errorf("The temp file doesn't exist.")
				}
			}

			// Stop the container.
			c.stopInstance(t, instanceName)
		}),
		e2e.ExpectExit(0),
	)
}

// Test by running directly from URI
func (c *ctx) testInstanceFromURI(t *testing.T) {
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
		args := []string{i.uri, i.name}
		c.env.RunSingularity(
			t,
			e2e.WithProfile(c.profile),
			e2e.WithCommand("instance start"),
			e2e.WithArgs(args...),
			e2e.PostRun(func(t *testing.T) {
				if t.Failed() {
					return
				}
				c.execInstance(t, i.name, "id")
				c.stopInstance(t, i.name)
			}),
			e2e.ExpectExit(0),
		)
	}
}

// E2ETests is the main func to trigger the test suite
func E2ETests(env e2e.TestEnv) func(*testing.T) {
	c := &ctx{
		env:     env,
		profile: e2e.UserProfile,
	}

	return func(t *testing.T) {
		e2e.EnsureImage(t, c.env)

		// Define and loop through tests.
		tests := []struct {
			name     string
			function func(*testing.T)
		}{
			{"InitialNoInstances", c.testNoInstances},
			{"BasicEchoServer", c.testBasicEchoServer},
			{"BasicOptions", c.testBasicOptions},
			{"Contain", c.testContain},
			{"InstanceFromURI", c.testInstanceFromURI},
			{"CreateManyInstances", c.testCreateManyInstances},
			{"StopAll", c.testStopAll},
			{"FinalNoInstances", c.testNoInstances},
		}

		// run unprivileged
		for _, tt := range tests {
			t.Run(tt.name, tt.function)
		}

		// run privileged
		c.profile = e2e.RootProfile
		for _, tt := range tests {
			t.Run("WithPriv"+tt.name, tt.function)
		}
	}
}
