// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

const (
	instanceStartPort  = 11372
	instanceDefinition = "../../examples/instances/Singularity"
	instanceImagePath  = "./instance_tests.sif"
)

type startOpts struct {
	addCaps       string
	allowSetuid   bool
	applyCgroups  string
	binds         []string
	boot          bool
	cleanenv      bool
	contain       bool
	containall    bool
	dns           string
	dropCaps      string
	home          string
	hostname      string
	keepPrivs     bool
	net           bool
	network       string
	networkArgs   string
	noHome        bool
	noPrivs       bool
	nv            bool
	overlay       string
	scratch       string
	security      string
	userns        bool
	uts           bool
	workdir       string
	writable      bool
	writableTmpfs bool
}

type listOpts struct {
	json      bool
	user      string
	container string
}

type stopOpts struct {
	all      bool
	force    bool
	signal   string
	timeout  string
	user     string
	instance string
}

type instance struct {
	Instance string `json:"instance"`
	Pid      int    `json:"pid"`
	Image    string `json:"img"`
}

type instanceList struct {
	Instances []instance `json:"instances"`
}

func startInstance(image string, instance string, portOffset int, opts startOpts) ([]byte, error) {
	args := []string{"instance", "start"}
	if opts.addCaps != "" {
		args = append(args, "--add-caps", opts.addCaps)
	}
	if opts.allowSetuid {
		args = append(args, "--allow-setuid")
	}
	if opts.applyCgroups != "" {
		args = append(args, "--apply-cgroups", opts.applyCgroups)
	}
	for _, bind := range opts.binds {
		args = append(args, "--bind", bind)
	}
	if opts.boot {
		args = append(args, "--boot")
	}
	if opts.cleanenv {
		args = append(args, "--cleanenv")
	}
	if opts.contain {
		args = append(args, "--contain")
	}
	if opts.containall {
		args = append(args, "--containall")
	}
	if opts.dns != "" {
		args = append(args, "--dns", opts.dns)
	}
	if opts.dropCaps != "" {
		args = append(args, "--drop-caps", opts.dropCaps)
	}
	if opts.home != "" {
		args = append(args, "--home", opts.home)
	}
	if opts.hostname != "" {
		args = append(args, "--hostname", opts.hostname)
	}
	if opts.keepPrivs {
		args = append(args, "--keep-privs")
	}
	if opts.net {
		args = append(args, "--net")
	}
	if opts.network != "" {
		args = append(args, "--network", opts.network)
	}
	if opts.networkArgs != "" {
		args = append(args, "--network-args", opts.networkArgs)
	}
	if opts.noHome {
		args = append(args, "--no-home")
	}
	if opts.noPrivs {
		args = append(args, "--no-privs")
	}
	if opts.nv {
		args = append(args, "--nv")
	}
	if opts.overlay != "" {
		args = append(args, "--overlay", opts.overlay)
	}
	if opts.scratch != "" {
		args = append(args, "--scratch", opts.scratch)
	}
	if opts.security != "" {
		args = append(args, "--security", opts.security)
	}
	if opts.userns {
		args = append(args, "--userns")
	}
	if opts.uts {
		args = append(args, "--uts")
	}
	if opts.workdir != "" {
		args = append(args, "--workdir", opts.workdir)
	}
	if opts.writable {
		args = append(args, "--writable")
	}
	if opts.writableTmpfs {
		args = append(args, "--writable-tmpfs")
	}
	args = append(args, image, instance, strconv.Itoa(instanceStartPort+portOffset))
	cmd := exec.Command(cmdPath, args...)
	return cmd.CombinedOutput()
}

func listInstance(opts listOpts) ([]byte, error) {
	args := []string{"instance", "list"}
	if opts.json {
		args = append(args, "--json")
	}
	if opts.user != "" {
		args = append(args, "--user", opts.user)
	}
	if opts.container != "" {
		args = append(args, opts.container)
	}
	cmd := exec.Command(cmdPath, args...)
	return cmd.CombinedOutput()
}

func stopInstance(opts stopOpts) ([]byte, error) {
	args := []string{"instance", "stop"}
	if opts.all {
		args = append(args, "--all")
	}
	if opts.force {
		args = append(args, "--force")
	}
	if opts.signal != "" {
		args = append(args, "--signal", opts.signal)
	}
	if opts.timeout != "" {
		args = append(args, "--timeout", opts.timeout)
	}
	if opts.user != "" {
		args = append(args, "--user", opts.user)
	}
	if opts.instance != "" {
		args = append(args, opts.instance)
	}
	cmd := exec.Command(cmdPath, args...)
	return cmd.CombinedOutput()
}

func execInstance(instance string, execCmd ...string) ([]byte, error) {
	args := []string{"exec", "instance://" + instance}
	args = append(args, execCmd...)
	cmd := exec.Command(cmdPath, args...)
	return cmd.CombinedOutput()
}

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
	output, err := listInstance(listOpts{json: true})
	if err != nil {
		t.Fatalf("Error listing instances: %v. Output:\n%s", err, string(output))
	}
	var instances instanceList
	if err = json.Unmarshal(output, &instances); err != nil {
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
	_, err := startInstance(instanceImagePath, instanceName, 0, startOpts{})
	if err != nil {
		t.Fatalf("Failed to start instance %s: %v", instanceName, err)
	}
	// Try to contact the instance.
	echo(t, instanceStartPort)
	// Stop the instance.
	_, err = stopInstance(stopOpts{instance: instanceName})
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
		_, err := startInstance(instanceImagePath, instanceName, i, startOpts{})
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
	_, err := stopInstance(stopOpts{all: true})
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
	dir, err := ioutil.TempDir("", "TestInstance")
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
	_, err = startInstance(instanceImagePath, instanceName, 0, instanceOpts)
	if err != nil {
		t.Fatalf("Failed to start instance %s: %v", instanceName, err)
	}
	// Verify we can see the file's contents from within the container.
	output, err := execInstance(instanceName, "cat", "/home/temp/"+fileName)
	if err != nil {
		t.Fatalf("Error executing command on instance %s: %v", instanceName, err)
	}
	if !bytes.Equal(fileContents, output) {
		t.Fatalf("File contents were %s, but expected %s", output, fileContents)
	}
	// Verify that the hostname has been set correctly.
	output, err = execInstance(instanceName, "hostname")
	if err != nil {
		t.Fatalf("Error executing command on instance %s: %v", instanceName, err)
	}
	if !bytes.Equal([]byte(testHostname+"\n"), output) {
		t.Fatalf("Hostname is %s, but expected %s", output, testHostname)
	}
	// Stop the container.
	_, err = stopInstance(stopOpts{instance: instanceName})
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
	_, err = startInstance(instanceImagePath, instanceName, 0, instanceOpts)
	if err != nil {
		t.Fatalf("Failed to start instance %s: %v", instanceName, err)
	}
	// Touch a file within /tmp.
	_, err = execInstance(instanceName, "touch", "/tmp/"+fileName)
	if err != nil {
		t.Fatalf("Failed to touch a file: %v", err)
	}
	// Stop the container.
	_, err = stopInstance(stopOpts{instance: instanceName})
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
		_, err := startInstance(i.uri, i.name, 0, startOpts{})
		if err != nil {
			t.Fatalf("Failed to start instance %s: %v", i.name, err)
		}
		// Exec id command.
		_, err = execInstance(i.name, "id")
		if err != nil {
			t.Fatalf("Failed to run id command: %v", err)
		}
		// Stop the container.
		_, err = stopInstance(stopOpts{instance: i.name})
		if err != nil {
			t.Fatalf("Failed to stop instance %s: %v", i.name, err)
		}
	}
}

// Bootstrap to run all instance tests.
func TestInstance(t *testing.T) {
	// Build a basic Singularity image to test instances.
	if b, err := imageBuild(buildOpts{force: true, sandbox: false}, instanceImagePath, instanceDefinition); err != nil {
		t.Log(string(b))
		t.Fatalf("unexpected failure: %v", err)
	}
	imageVerify(t, instanceImagePath, true)
	defer os.RemoveAll(instanceImagePath)
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
			wrappedFn = test.WithPrivilege(tt.function)
		} else {
			wrappedFn = test.WithoutPrivilege(tt.function)
		}
		t.Run(tt.name, wrappedFn)
	}
}
