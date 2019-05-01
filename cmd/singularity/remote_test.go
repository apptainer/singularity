// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

// TODO: Tests for remote are not implemented because there is not a great way to handle
// valid authentication tokens for testing at the moment

func TestRemoteAdd(t *testing.T) {
	test.DropPrivilege(t)

	config, err := ioutil.TempFile(testDir, "testConfig-")
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
			if b, err := exec.Command(cmdPath, argv...).CombinedOutput(); err != nil {
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
			if b, err := exec.Command(cmdPath, argv...).CombinedOutput(); err == nil {
				t.Log(string(b))
				t.Fatalf("unexpected success: %v", err)
			}
		}))
	}
}

func TestRemoteRemove(t *testing.T) {
	test.DropPrivilege(t)

	config, err := ioutil.TempFile(testDir, "testConfig-")
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
			if b, err := exec.Command(cmdPath, argv...).CombinedOutput(); err != nil {
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
			if b, err := exec.Command(cmdPath, argv...).CombinedOutput(); err != nil {
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
			if b, err := exec.Command(cmdPath, argv...).CombinedOutput(); err == nil {
				t.Log(string(b))
				t.Fatalf("unexpected success: %v", err)
			}
		}))
	}
}

func TestRemoteUse(t *testing.T) {
	test.DropPrivilege(t)

	config, err := ioutil.TempFile(testDir, "testConfig-")
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
			if b, err := exec.Command(cmdPath, argv...).CombinedOutput(); err == nil {
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
			if b, err := exec.Command(cmdPath, argv...).CombinedOutput(); err != nil {
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
			if b, err := exec.Command(cmdPath, argv...).CombinedOutput(); err != nil {
				t.Log(string(b))
				t.Fatalf("unexpected failure: %v", err)
			}
		}))
	}
}

func TestRemoteStatus(t *testing.T) {
	test.DropPrivilege(t)

	config, err := ioutil.TempFile(testDir, "testConfig-")
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
			if b, err := exec.Command(cmdPath, argv...).CombinedOutput(); err != nil {
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
			if b, err := exec.Command(cmdPath, argv...).CombinedOutput(); err != nil {
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
			if b, err := exec.Command(cmdPath, argv...).CombinedOutput(); err == nil {
				t.Log(string(b))
				t.Fatalf("unexpected success: %v", err)
			}
		}))
	}
}

func TestRemoteList(t *testing.T) {
	test.DropPrivilege(t)

	config, err := ioutil.TempFile(testDir, "testConfig-")
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
			if b, err := exec.Command(cmdPath, argv...).CombinedOutput(); err != nil {
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
			if b, err := exec.Command(cmdPath, argv...).CombinedOutput(); err != nil {
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
			if b, err := exec.Command(cmdPath, argv...).CombinedOutput(); err != nil {
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
			if b, err := exec.Command(cmdPath, argv...).CombinedOutput(); err != nil {
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
			if b, err := exec.Command(cmdPath, argv...).CombinedOutput(); err != nil {
				t.Log(string(b))
				t.Fatalf("unexpected failure: %v", err)
			}
		}))
	}
}
