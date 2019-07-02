// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package instance

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
)

type testingEnv struct {
	// base env for running tests
	CmdPath     string `split_words:"true"`
	TestDir     string `split_words:"true"`
	RunDisabled bool   `default:"false"`
	//  base image for tests
	ImagePath string `split_words:"true"`
}

var testenv testingEnv

// Sends a deterministic message to an echo server and expects the same message
// in response.
func echo(t *testing.T, port int) {
	const message = "b40cbeaaea293f7e8bd40fb61f389cfca9823467\n"
	sock, sockErr := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if sockErr != nil {
		t.Fatalf("Failed to dial echo server: %v", sockErr)
	}
	fmt.Fprintf(sock, message)
	response, responseErr := bufio.NewReader(sock).ReadString('\n')
	if responseErr != nil || response != message {
		t.Fatalf("Bad response: err = %v, response = %v", responseErr, response)
	}
}

// Return the number of currently running instances.
func getNumberOfInstances(t *testing.T) int {
	stdout, _, err := listInstance(listOpts{json: true})
	if err != nil {
		t.Fatalf("Error listing instances: %v", err)
	}
	var instances instanceList
	if err = json.Unmarshal([]byte(stdout), &instances); err != nil {
		t.Fatalf("Error decoding JSON from listInstance: %v", err)
	}
	return len(instances.Instances)
}

// Test that no instances are running.
func testNoInstances(t *testing.T) {
	if n := getNumberOfInstances(t); n != 0 {
		t.Fatalf("There are %d instances running, but there should be 0.\n", n)
	}
}

// Test that a basic echo server instance can be started, communicated with,
// and stopped.
func testBasicEchoServer(t *testing.T) {
	const instanceName = "echo1"
	// Start the instance.
	_, _, err := startInstance(startOpts{}, testenv.ImagePath, instanceName, strconv.Itoa(instanceStartPort))
	if err != nil {
		t.Fatalf("Failed to start instance %s: %v", instanceName, err)
	}
	// Try to contact the instance.
	echo(t, instanceStartPort)
	// Stop the instance.
	_, _, err = stopInstance(stopOpts{}, instanceName)
	if err != nil {
		t.Fatalf("Failed to stop instance %s: %v", instanceName, err)
	}
}

// Test creating many instances, but don't stop them.
func testCreateManyInstances(t *testing.T) {
	const n = 10
	// Start n instances.
	for i := 0; i < n; i++ {
		instanceName := "echo" + strconv.Itoa(i+1)
		_, _, err := startInstance(startOpts{}, testenv.ImagePath, instanceName, strconv.Itoa(instanceStartPort+i))
		if err != nil {
			t.Fatalf("Failed to start instance %s: %v", instanceName, err)
		}
	}
	// Verify all instances started.
	if numStarted := getNumberOfInstances(t); numStarted != n {
		t.Fatalf("Expected %d instances, but see %d.", n, numStarted)
	}
	// Echo all n instances.
	for i := 0; i < n; i++ {
		echo(t, instanceStartPort+i)
	}
}

// Test stopping all running instances.
func testStopAll(t *testing.T) {
	_, _, err := stopInstance(stopOpts{all: true}, "")
	if err != nil {
		t.Fatalf("Failed to stop all instances: %v", err)
	}
}

// Test basic options like mounting a custom home directory, changing the
// hostname, etc.
func testBasicOptions(t *testing.T) {
	const fileName = "hello"
	const instanceName = "testbasic"
	const testHostname = "echoserver99"
	fileContents := []byte("world")

	// Create a temporary directory to serve as a home directory.
	dir, err := ioutil.TempDir(testenv.TestDir, "TestInstance")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(dir)
	// Create and populate a temporary file.
	tempFile := filepath.Join(dir, fileName)
	err = ioutil.WriteFile(tempFile, fileContents, 0644)
	if err != nil {
		t.Fatalf("Failed to create file %s: %v", tempFile, err)
	}
	instanceOpts := startOpts{
		home:     dir + ":/home/temp",
		hostname: testHostname,
		cleanenv: true,
	}
	// Start an instance with the temporary directory as the home directory.
	_, _, err = startInstance(instanceOpts, testenv.ImagePath, instanceName, strconv.Itoa(instanceStartPort))
	if err != nil {
		t.Fatalf("Failed to start instance %s: %v", instanceName, err)
	}
	// Verify we can see the file's contents from within the container.
	stdout, _, err := execInstance(instanceName, "cat", "/home/temp/"+fileName)
	if err != nil {
		t.Fatalf("Error executing command on instance %s: %v", instanceName, err)
	}
	if !bytes.Equal(fileContents, []byte(stdout)) {
		t.Fatalf("File contents were %s, but expected %s", stdout, string(fileContents))
	}
	// Verify that the hostname has been set correctly.
	stdout, _, err = execInstance(instanceName, "hostname")
	if err != nil {
		t.Fatalf("Error executing command on instance %s: %v", instanceName, err)
	}
	if !bytes.Equal([]byte(testHostname+"\n"), []byte(stdout)) {
		t.Fatalf("Hostname is %s, but expected %s", stdout, testHostname)
	}
	// Stop the container.
	_, _, err = stopInstance(stopOpts{}, instanceName)
	if err != nil {
		t.Fatalf("Failed to stop instance %s: %v", instanceName, err)
	}
}

// Test that contain works.
func testContain(t *testing.T) {
	const instanceName = "testcontain"
	const fileName = "thegreattestfile"
	// Create a temporary directory to serve as a contain directory.
	dir, err := ioutil.TempDir("", "TestInstance")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(dir)
	instanceOpts := startOpts{
		contain: true,
		workdir: dir,
	}
	// Start an instance with the temporary directory as the home directory.
	_, _, err = startInstance(instanceOpts, testenv.ImagePath, instanceName, strconv.Itoa(instanceStartPort))
	if err != nil {
		t.Fatalf("Failed to start instance %s: %v", instanceName, err)
	}
	// Touch a file within /tmp.
	_, _, err = execInstance(instanceName, "touch", "/tmp/"+fileName)
	if err != nil {
		t.Fatalf("Failed to touch a file: %v", err)
	}
	// Stop the container.
	_, _, err = stopInstance(stopOpts{}, instanceName)
	if err != nil {
		t.Fatalf("Failed to stop instance %s: %v", instanceName, err)
	}
	// Verify that the touched file exists outside the container.
	if _, err = os.Stat(filepath.Join(dir, "tmp", fileName)); os.IsNotExist(err) {
		t.Fatal("The temp file doesn't exist.")
	}
}

// Test by running directly from URI
func testInstanceFromURI(t *testing.T) {
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
		{
			name: "test_from_shub",
			uri:  "shub://singularityhub/busybox",
		},
	}

	for _, i := range instances {
		// Start an instance with the temporary directory as the home directory.
		_, _, err := startInstance(startOpts{}, i.uri, i.name)
		if err != nil {
			t.Fatalf("Failed to start instance %s: %v", i.name, err)
		}
		// Exec id command.
		_, _, err = execInstance(i.name, "id")
		if err != nil {
			t.Fatalf("Failed to run id command: %v", err)
		}
		// Stop the container.
		_, _, err = stopInstance(stopOpts{}, i.name)
		if err != nil {
			t.Fatalf("Failed to stop instance %s: %v", i.name, err)
		}
	}
}

// Bootstrap to run all instance tests.
func legacyInstanceTests(t *testing.T) {
	e2e.EnsureImage(t)

	// Define and loop through tests.
	tests := []struct {
		name       string
		function   func(*testing.T)
		privileged bool
	}{
		{"InitialNoInstances", testNoInstances, false},
		{"BasicEchoServer", testBasicEchoServer, false},
		{"BasicOptions", testBasicOptions, false},
		{"Contain", testContain, false},
		{"InstanceFromURI", testInstanceFromURI, false},
		{"CreateManyInstances", testCreateManyInstances, false},
		{"StopAll", testStopAll, false},
		{"FinalNoInstances", testNoInstances, false},
	}
	for _, tt := range tests {
		var wrappedFn func(*testing.T)
		if tt.privileged {
			wrappedFn = e2e.Privileged(tt.function)
		} else {
			wrappedFn = tt.function
		}
		t.Run(tt.name, wrappedFn)
	}
}

// RunE2ETests is the bootstrap to run all instance tests.
func RunE2ETests(t *testing.T) {
	e2e.LoadEnv(t, &testenv)

	t.Run("legacy", legacyInstanceTests)
}
