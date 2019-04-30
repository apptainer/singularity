// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package remote

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"testing"

	"github.com/kelseyhightower/envconfig"
	"github.com/sylabs/singularity/internal/pkg/test"
)

type testingEnv struct {
	// base env for running tests
	CmdPath string `split_words:"true"`
	TestDir string `split_words:"true"`
}

var testenv testingEnv

// remoteAdd checks the functionality of "singularity remote add" command.
// It Verifies that adding valid endpoints results in success and invalid
// one's results in failure.
func remoteAdd(t *testing.T) {

	test.DropPrivilege(t)

	config, err := ioutil.TempFile(testenv.TestDir, "testConfig-")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(config.Name()) // clean up

	testPass := []struct {
		name   string
		remote string
		uri    string
	}{
		{"AddCloud", "cloud", "cloud.sylabs.io"},
		{"AddOtherCloud", "other", "cloud.sylabs.io"},
	}

	for _, tt := range testPass {
		argv := []string{"remote", "--config", config.Name(), "add"}
		argv = append(argv, tt.remote, tt.uri)
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			if b, err := exec.Command(testenv.CmdPath, argv...).CombinedOutput(); err != nil {
				t.Log(string(b))
				t.Fatalf("unexpected failure: %v", err)
			}
		}))
	}

	testFail := []struct {
		name   string
		remote string
		uri    string
	}{
		{"AddExistingRemote", "cloud", "cloud.sylabs.io"},
		{"AddExistingRemoteInvalidURI", "other", "anythingcangohere"},
	}

	for _, tt := range testFail {
		argv := []string{"remote", "--config", config.Name(), "add"}
		argv = append(argv, tt.remote, tt.uri)
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			if b, err := exec.Command(testenv.CmdPath, argv...).CombinedOutput(); err == nil {
				t.Log(string(b))
				t.Fatalf("unexpected success: %v", err)
			}
		}))
	}
}

// remoteRemove tests the functionality of "singularity remote remove" command.
// 1. Adds remote endpoints
// 2. Deletes the already added entries
// 3. Verfies that removing an invalid entry results in a failure
func remoteRemove(t *testing.T) {
	test.DropPrivilege(t)

	config, err := ioutil.TempFile(testenv.TestDir, "testConfig-")
	if err != nil {
		log.Fatal(err)
	}

	defer os.Remove(config.Name()) // clean up

	// Prep config by adding multiple remotes
	add := []struct {
		name   string
		remote string
		uri    string
	}{
		{"addCloud", "cloud", "cloud.sylabs.io"},
		{"addOther", "other", "cloud.sylabs.io"},
	}

	for _, tt := range add {
		argv := []string{"remote", "--config", config.Name(), "add"}
		argv = append(argv, tt.remote, tt.uri)
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			if b, err := exec.Command(testenv.CmdPath, argv...).CombinedOutput(); err != nil {
				t.Log(string(b))
				t.Fatalf("unexpected failure: %v", err)
			}
		}))
	}

	testPass := []struct {
		name   string
		remote string
	}{
		{"RemoveCloud", "cloud"},
		{"RemoveOther", "other"},
	}

	for _, tt := range testPass {
		argv := []string{"remote", "--config", config.Name(), "remove"}
		argv = append(argv, tt.remote)
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			if b, err := exec.Command(testenv.CmdPath, argv...).CombinedOutput(); err != nil {
				t.Log(string(b))
				t.Fatalf("unexpected failure: %v", err)
			}
		}))
	}

	testFail := []struct {
		name   string
		remote string
	}{
		{"RemoveNonExistingRemote", "cloud"},
	}

	for _, tt := range testFail {
		argv := []string{"remote", "--config", config.Name(), "remove"}
		argv = append(argv, tt.remote)
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			if b, err := exec.Command(testenv.CmdPath, argv...).CombinedOutput(); err == nil {
				t.Log(string(b))
				t.Fatalf("unexpected success: %v", err)
			}
		}))
	}
}

// remoteUse tests the functionality of "singularity remote use" command.
// 1. Tries to use non-existing remote entry
// 2. Adds remote entries and tries to use those
func remoteUse(t *testing.T) {
	test.DropPrivilege(t)

	config, err := ioutil.TempFile(testenv.TestDir, "testConfig-")
	if err != nil {
		log.Fatal(err)
	}

	defer os.Remove(config.Name()) // clean up

	testFail := []struct {
		name   string
		remote string
	}{
		{"UseNonExistingRemote", "cloud"},
	}

	for _, tt := range testFail {
		argv := []string{"remote", "--config", config.Name(), "use"}
		argv = append(argv, tt.remote)
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			if b, err := exec.Command(testenv.CmdPath, argv...).CombinedOutput(); err == nil {
				t.Log(string(b))
				t.Fatalf("unexpected success: %v", err)
			}
		}))
	}

	// Prep config by adding multiple remotes
	add := []struct {
		name   string
		remote string
		uri    string
	}{
		{"addCloud", "cloud", "cloud.sylabs.io"},
		{"addOther", "other", "cloud.sylabs.io"},
	}

	for _, tt := range add {
		argv := []string{"remote", "--config", config.Name(), "add"}
		argv = append(argv, tt.remote, tt.uri)
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			if b, err := exec.Command(testenv.CmdPath, argv...).CombinedOutput(); err != nil {
				t.Log(string(b))
				t.Fatalf("unexpected failure: %v", err)
			}
		}))
	}

	testPass := []struct {
		name   string
		remote string
	}{
		{"UseFromNothingToRemote", "cloud"},
		{"UseFromRemoteToRemote", "other"},
	}

	for _, tt := range testPass {
		argv := []string{"remote", "--config", config.Name(), "use"}
		argv = append(argv, tt.remote)
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			if b, err := exec.Command(testenv.CmdPath, argv...).CombinedOutput(); err != nil {
				t.Log(string(b))
				t.Fatalf("unexpected failure: %v", err)
			}
		}))
	}
}

// remoteStatus tests the functionality of "singularity remote status" command.
// 1. Adds remote endpoints
// 2. Verifies that remote status command succeeds on existing endpoints
// 3. Verifies that remote status command fails on non-existing endpoints
func remoteStatus(t *testing.T) {
	test.DropPrivilege(t)

	config, err := ioutil.TempFile(testenv.TestDir, "testConfig-")
	if err != nil {
		log.Fatal(err)
	}

	defer os.Remove(config.Name()) // clean up

	// Prep config by adding multiple remotes
	add := []struct {
		name   string
		remote string
		uri    string
	}{
		{"addCloud", "cloud", "cloud.sylabs.io"},
		{"addInvalidRemote", "invalid", "notarealendpoint"},
	}

	for _, tt := range add {
		argv := []string{"remote", "--config", config.Name(), "add"}
		argv = append(argv, tt.remote, tt.uri)
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			if b, err := exec.Command(testenv.CmdPath, argv...).CombinedOutput(); err != nil {
				t.Log(string(b))
				t.Fatalf("unexpected failure: %v", err)
			}
		}))
	}

	testPass := []struct {
		name   string
		remote string
	}{
		{"ValidRemote", "cloud"},
	}

	for _, tt := range testPass {
		argv := []string{"remote", "--config", config.Name(), "status"}
		argv = append(argv, tt.remote)
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			if b, err := exec.Command(testenv.CmdPath, argv...).CombinedOutput(); err != nil {
				t.Log(string(b))
				t.Fatalf("unexpected failure: %v", err)
			}
		}))
	}

	testFail := []struct {
		name   string
		remote string
	}{
		{"NonExistingRemote", "notaremote"},
		{"NonExistingEndpoint", "invalid"},
	}

	for _, tt := range testFail {
		argv := []string{"remote", "--config", config.Name(), "status"}
		argv = append(argv, tt.remote)
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			if b, err := exec.Command(testenv.CmdPath, argv...).CombinedOutput(); err == nil {
				t.Log(string(b))
				t.Fatalf("unexpected success: %v", err)
			}
		}))
	}
}

// remoteList tests the functionality of "singularity remote list" command
func remoteList(t *testing.T) {
	test.DropPrivilege(t)

	config, err := ioutil.TempFile(testenv.TestDir, "testConfig-")
	if err != nil {
		log.Fatal(err)
	}

	defer os.Remove(config.Name()) // clean up

	testPass := []struct {
		name string
	}{
		{"EmptyConfig"},
	}

	for _, tt := range testPass {
		argv := []string{"remote", "--config", config.Name(), "list"}
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			fmt.Println("config.name is ", config.Name())
			if b, err := exec.Command(testenv.CmdPath, argv...).CombinedOutput(); err != nil {
				t.Log(string(b))
				t.Fatalf("unexpected failure: %v", err)
			}
		}))
	}

	// Prep config by adding multiple remotes
	add := []struct {
		name   string
		remote string
		uri    string
	}{
		{"addCloud", "cloud", "cloud.sylabs.io"},
		{"addRemote", "remote", "cloud.sylabs.io"},
	}

	for _, tt := range add {
		argv := []string{"remote", "--config", config.Name(), "add"}
		argv = append(argv, tt.remote, tt.uri)
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			if b, err := exec.Command(testenv.CmdPath, argv...).CombinedOutput(); err != nil {
				t.Log(string(b))
				t.Fatalf("unexpected failure: %v", err)
			}
		}))
	}

	testPass = []struct {
		name string
	}{
		{"PopulatedConfig"},
	}

	for _, tt := range testPass {
		argv := []string{"remote", "--config", config.Name(), "list"}
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			if b, err := exec.Command(testenv.CmdPath, argv...).CombinedOutput(); err != nil {
				t.Log(string(b))
				t.Fatalf("unexpected failure: %v", err)
			}
		}))
	}

	// Prep config by selecting a remote to default to
	use := []struct {
		name   string
		remote string
	}{
		{"useCloud", "cloud"},
	}

	for _, tt := range use {
		argv := []string{"remote", "--config", config.Name(), "use"}
		argv = append(argv, tt.remote)
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			if b, err := exec.Command(testenv.CmdPath, argv...).CombinedOutput(); err != nil {
				t.Log(string(b))
				t.Fatalf("unexpected failure: %v", err)
			}
		}))
	}

	testPass = []struct {
		name string
	}{
		{"PopulatedConfigWithDefault"},
	}

	for _, tt := range testPass {
		argv := []string{"remote", "--config", config.Name(), "list"}
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			if b, err := exec.Command(testenv.CmdPath, argv...).CombinedOutput(); err != nil {
				t.Log(string(b))
				t.Fatalf("unexpected failure: %v", err)
			}
		}))
	}
}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(t *testing.T) {
	err := envconfig.Process("E2E", &testenv)

	if err != nil {
		t.Fatal(err.Error())
	}

	t.Run("remote_add", remoteAdd)
	t.Run("remote_remove", remoteRemove)
	t.Run("remote_use", remoteUse)
	t.Run("remote_status", remoteStatus)
	t.Run("remote_list", remoteList)
}
