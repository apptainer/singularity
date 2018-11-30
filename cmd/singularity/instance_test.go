// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

const instanceStartPort = 11372
const instanceDefinition = "../../examples/instances/Singularity"
const instanceImagePath = "./instance_tests.sif"

type startOpts struct {
	add_caps       string
	allow_setuid   bool
	apply_cgroups  string
	bind           string
	boot           bool
	cleanenv       bool
	contain        bool
	containall     bool
	dns            string
	drop_caps      string
	home           string
	hostname       string
	keep_privs     bool
	net            bool
	network        string
	network_args   string
	no_home        bool
	no_privs       bool
	nv             bool
	overlay        string
	scratch        string
	security       string
	userns         bool
	uts            bool
	workdir        string
	writable       bool
	writable_tmpfs bool
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

func startInstance(image string, instance string, opts startOpts) ([]byte, error) {
	args := []string{"instance", "start"}
	if opts.add_caps != "" {
		args = append(args, "--add-caps", opts.add_caps)
	}
	if opts.allow_setuid {
		args = append(args, "--allow-setuid")
	}
	if opts.apply_cgroups != "" {
		args = append(args, "--apply-cgroups", opts.apply_cgroups)
	}
	if opts.bind != "" {
		args = append(args, "--bind", opts.bind)
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
	if opts.drop_caps != "" {
		args = append(args, "--drop-caps", opts.drop_caps)
	}
	if opts.home != "" {
		args = append(args, "--home", opts.home)
	}
	if opts.hostname != "" {
		args = append(args, "--hostname", opts.hostname)
	}
	if opts.keep_privs {
		args = append(args, "--keep-privs")
	}
	if opts.net {
		args = append(args, "--net")
	}
	if opts.network != "" {
		args = append(args, "--network", opts.network)
	}
	if opts.network_args != "" {
		args = append(args, "--network-args", opts.network_args)
	}
	if opts.no_home {
		args = append(args, "--no-home")
	}
	if opts.no_privs {
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
	if opts.writable_tmpfs {
		args = append(args, "--writable-tmpfs")
	}
	args = append(args, image, instance)
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

// Sends a deterministic message to an echo server and expects the same message
// in response.
func echo(t *testing.T, port int) {
	const message = "b40cbeaaea293f7e8bd40fb61f389cfca9823467\n"
	sock, sock_err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if sock_err != nil {
		t.Fatalf("Failed to dial echo server: %v", sock_err)
	}
	fmt.Fprintf(sock, message)
	response, response_err := bufio.NewReader(sock).ReadString('\n')
	if response_err != nil || response != message {
		t.Fatalf("Bad response: err = %v, response = %v", response_err, response)
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
	_, err := startInstance(instanceImagePath, instanceName, startOpts{})
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
		_, err := startInstance(instanceImagePath, instanceName, startOpts{})
		if err != nil {
			t.Fatalf("Failed to start instance %s: %v", instanceName, err)
		}
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
		{"CreateManyInstances", testCreateManyInstances, false},
		{"StopAll", testStopAll, false},
		{"FinalNoInstances", testNoInstances, false},
	}
	for _, currentTest := range tests {
		var wrappedFn func(*testing.T)
		if currentTest.privileged {
			wrappedFn = test.WithPrivilege(currentTest.function)
		} else {
			wrappedFn = test.WithoutPrivilege(currentTest.function)
		}
		t.Run(currentTest.name, wrappedFn)
	}
}
