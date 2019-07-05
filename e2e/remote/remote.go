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
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/internal/pkg/test"
	"github.com/sylabs/singularity/internal/pkg/test/exec"
)

type ctx struct {
	env e2e.TestEnv
}

// remoteAdd checks the functionality of "singularity remote add" command.
// It Verifies that adding valid endpoints results in success and invalid
// one's results in failure.
func (c *ctx) remoteAdd(t *testing.T) {

	test.DropPrivilege(t)

	config, err := ioutil.TempFile(c.env.TestDir, "testConfig-")
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
		t.Run(tt.name, func(t *testing.T) {
			if res := exec.Command(c.env.CmdPath, argv...).Run(t); res.Error != nil {
				t.Fatalf("Unexpected failure.\n%s", res)
			}
		})
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
		t.Run(tt.name, func(t *testing.T) {
			if res := exec.Command(c.env.CmdPath, argv...).Run(t); res.Error == nil {
				t.Fatalf("Unexpected success.\n%s", res)
			}
		})
	}
}

// remoteRemove tests the functionality of "singularity remote remove" command.
// 1. Adds remote endpoints
// 2. Deletes the already added entries
// 3. Verfies that removing an invalid entry results in a failure
func (c *ctx) remoteRemove(t *testing.T) {
	test.DropPrivilege(t)

	config, err := ioutil.TempFile(c.env.TestDir, "testConfig-")
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
		t.Run(tt.name, func(t *testing.T) {
			if res := exec.Command(c.env.CmdPath, argv...).Run(t); res.Error != nil {
				t.Fatalf("Unexpected failure.\n%s", res)
			}
		})
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
		t.Run(tt.name, func(t *testing.T) {
			if res := exec.Command(c.env.CmdPath, argv...).Run(t); res.Error != nil {
				t.Fatalf("Unexpected failure.\n%s", res)
			}
		})
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
		t.Run(tt.name, func(t *testing.T) {
			if res := exec.Command(c.env.CmdPath, argv...).Run(t); res.Error == nil {
				t.Fatalf("Unexpected success.\n%s", res)
			}
		})
	}
}

// remoteUse tests the functionality of "singularity remote use" command.
// 1. Tries to use non-existing remote entry
// 2. Adds remote entries and tries to use those
func (c *ctx) remoteUse(t *testing.T) {
	test.DropPrivilege(t)

	config, err := ioutil.TempFile(c.env.TestDir, "testConfig-")
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
		t.Run(tt.name, func(t *testing.T) {
			if res := exec.Command(c.env.CmdPath, argv...).Run(t); res.Error == nil {
				t.Fatalf("Unexpected success.\n%s", res)
			}
		})
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
		t.Run(tt.name, func(t *testing.T) {
			if res := exec.Command(c.env.CmdPath, argv...).Run(t); res.Error != nil {
				t.Fatalf("Unexpected failure.\n%s", res)
			}
		})
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
		t.Run(tt.name, func(t *testing.T) {
			if res := exec.Command(c.env.CmdPath, argv...).Run(t); res.Error != nil {
				t.Fatalf("Unexpected failure.\n%s", res)
			}
		})
	}
}

// remoteStatus tests the functionality of "singularity remote status" command.
// 1. Adds remote endpoints
// 2. Verifies that remote status command succeeds on existing endpoints
// 3. Verifies that remote status command fails on non-existing endpoints
func (c *ctx) remoteStatus(t *testing.T) {
	test.DropPrivilege(t)

	config, err := ioutil.TempFile(c.env.TestDir, "testConfig-")
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
		t.Run(tt.name, func(t *testing.T) {
			if res := exec.Command(c.env.CmdPath, argv...).Run(t); res.Error != nil {
				t.Fatalf("Unexpected failure.\n%s", res)
			}
		})
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
		t.Run(tt.name, func(t *testing.T) {
			if res := exec.Command(c.env.CmdPath, argv...).Run(t); res.Error != nil {
				t.Fatalf("Unexpected failure.\n%s", res)
			}
		})
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
		t.Run(tt.name, func(t *testing.T) {
			if res := exec.Command(c.env.CmdPath, argv...).Run(t); res.Error == nil {
				t.Fatalf("Unexpected success.\n%s", res)
			}
		})
	}
}

// remoteList tests the functionality of "singularity remote list" command
func (c *ctx) remoteList(t *testing.T) {
	test.DropPrivilege(t)

	config, err := ioutil.TempFile(c.env.TestDir, "testConfig-")
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
		t.Run(tt.name, func(t *testing.T) {
			fmt.Println("config.name is ", config.Name())
			if res := exec.Command(c.env.CmdPath, argv...).Run(t); res.Error != nil {
				t.Fatalf("Unexpected failure.\n%s", res)
			}
		})
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
		t.Run(tt.name, func(t *testing.T) {
			if res := exec.Command(c.env.CmdPath, argv...).Run(t); res.Error != nil {
				t.Fatalf("Unexpected failure.\n%s", res)
			}
		})
	}

	testPass = []struct {
		name string
	}{
		{"PopulatedConfig"},
	}

	for _, tt := range testPass {
		argv := []string{"remote", "--config", config.Name(), "list"}
		t.Run(tt.name, func(t *testing.T) {
			if res := exec.Command(c.env.CmdPath, argv...).Run(t); res.Error != nil {
				t.Fatalf("Unexpected failure.\n%s", res)
			}
		})
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
		t.Run(tt.name, func(t *testing.T) {
			if res := exec.Command(c.env.CmdPath, argv...).Run(t); res.Error != nil {
				t.Fatalf("Unexpected failure.\n%s", res)
			}
		})
	}

	testPass = []struct {
		name string
	}{
		{"PopulatedConfigWithDefault"},
	}

	for _, tt := range testPass {
		argv := []string{"remote", "--config", config.Name(), "list"}
		t.Run(tt.name, func(t *testing.T) {
			if res := exec.Command(c.env.CmdPath, argv...).Run(t); res.Error != nil {
				t.Fatalf("Unexpected failure.\n%s", res)
			}
		})
	}
}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(env e2e.TestEnv) func(*testing.T) {
	c := &ctx{
		env: env,
	}

	return func(t *testing.T) {
		t.Run("add", c.remoteAdd)
		t.Run("remove", c.remoteRemove)
		t.Run("use", c.remoteUse)
		t.Run("status", c.remoteStatus)
		t.Run("list", c.remoteList)
	}
}
