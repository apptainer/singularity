// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build integration_test
// +build seccomp

package main

import (
	"os"
	"path"
	"strings"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/test"
)

// Path for image used in these security tests
var securityImagePath string

// testSecurityUnpriv tests security flag fuctionality for singularity exec without elevated privileges
func testSecurityUnpriv(t *testing.T) {
	tests := []struct {
		name   string
		image  string
		action string
		argv   []string
		opts
		exit          int
		expectSuccess bool
	}{
		// taget UID/GID
		{"Set_uid", securityImagePath, "exec", []string{"id", "-u", "|", "grep", "99"}, opts{security: []string{"uid:99"}}, 1, false},
		{"Set_gid", securityImagePath, "exec", []string{"id", "-g", "|", "grep", "99"}, opts{security: []string{"gid:99"}}, 1, false},
		// seccomp from json file
		{"SecComp_BlackList", securityImagePath, "exec", []string{"mkdir", "/tmp/foo"}, opts{security: []string{"seccomp:./testdata/seccomp-profile.json"}}, 1, false},
		{"SecComp_true", securityImagePath, "exec", []string{"true"}, opts{security: []string{"seccomp:./testdata/seccomp-profile.json"}}, 0, true},
		// capabilities
		{"capabilities_keep_true", securityImagePath, "exec", []string{"ping", "-c", "1", "8.8.8.8"}, opts{keepPrivs: true}, 1, false},
		{"capabilities_keep-false", securityImagePath, "exec", []string{"ping", "-c", "1", "8.8.8.8"}, opts{keepPrivs: false}, 1, false},
		{"capabilities_drop", securityImagePath, "exec", []string{"ping", "-c", "1", "8.8.8.8"}, opts{dropCaps: "CAP_NET_RAW"}, 1, false},
	}

	for _, tt := range tests {
		t.Run("unpriv "+tt.name, test.WithoutPrivilege(func(t *testing.T) {
			stdout, stderr, exitCode, err := imageExec(t, tt.action, tt.opts, tt.image, tt.argv)
			if tt.expectSuccess && (exitCode != 0) {
				t.Log(stdout, stderr, exitCode)
				t.Fatalf("unexpected failure running '%v': %v", strings.Join(tt.argv, " "), err)
			} else if !tt.expectSuccess && (exitCode != 1) {
				t.Log(stdout, stderr, exitCode)
				t.Fatalf("unexpected success running '%v'", strings.Join(tt.argv, " "))
			}
		}))
	}
}

// testSecurityPriv tests security flag fuctionality for singularity exec with elevated privileges
func testSecurityPriv(t *testing.T) {
	tests := []struct {
		name   string
		image  string
		action string
		argv   []string
		opts
		exit          int
		expectSuccess bool
	}{
		// taget UID/GID
		{"Set_uid", securityImagePath, "exec", []string{"id", "-u", "|", "grep", "99"}, opts{security: []string{"uid:99"}}, 1, false},
		{"Set_gid", securityImagePath, "exec", []string{"id", "-g", "|", "grep", "99"}, opts{security: []string{"gid:99"}}, 1, false},
		// seccomp from json file
		{"SecComp_BlackList", securityImagePath, "exec", []string{"mkdir", "/tmp/foo"}, opts{security: []string{"seccomp:./testdata/seccomp-profile.json"}}, 1, false},
		{"SecComp_true", securityImagePath, "exec", []string{"true"}, opts{security: []string{"seccomp:./testdata/seccomp-profile.json"}}, 0, true},
		// capabilities
		{"capabilities_keep", securityImagePath, "exec", []string{"ping", "-c", "1", "8.8.8.8"}, opts{keepPrivs: true}, 0, true},
		{"capabilities_drop", securityImagePath, "exec", []string{"ping", "-c", "1", "8.8.8.8"}, opts{dropCaps: "CAP_NET_RAW"}, 1, false},
	}

	for _, tt := range tests {
		t.Run("priv "+tt.name, test.WithPrivilege(func(t *testing.T) {
			stdout, stderr, exitCode, err := imageExec(t, tt.action, tt.opts, tt.image, tt.argv)
			if tt.expectSuccess && (exitCode != 0) {
				t.Log(stdout, stderr, exitCode)
				t.Fatalf("unexpected failure running '%v': %v", strings.Join(tt.argv, " "), err)
			} else if !tt.expectSuccess && (exitCode != 1) {
				t.Log(stdout, stderr, exitCode)
				t.Fatalf("unexpected success running '%v'", strings.Join(tt.argv, " "))
			}
		}))
	}
}

// testSecurityConfOwnership tests checks on config files ownerships
func testSecurityConfOwnership(t *testing.T) {
	configFile := buildcfg.SINGULARITY_CONF_FILE
	// Change file ownership (do not try this at home)
	err := os.Chown(configFile, 1001, 0)
	if err != nil {
		t.Fatal(err)
	}

	// try to run
	t.Run("non_root_config", test.WithoutPrivilege(func(t *testing.T) {
		_, stderr, exitCode, err := imageExec(t, "exec", opts{}, securityImagePath, []string{"/bin/true"})
		if exitCode != 1 {
			t.Log(stderr, err)
			t.Fatalf("unexpected success running /bin/true")
		}
	}))

	// return file ownership to normal
	err = os.Chown(configFile, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSecurity(t *testing.T) {
	test.EnsurePrivilege(t)
	opts := buildOpts{
		force:   true,
		sandbox: false,
	}

	// Create a clean image cache
	imgCache, cleanup := setupCache(t)
	defer cleanup()

	// Was previously using imagePath that got set in TestBuild. This resulted
	// in a failure on systems with cores to run in parallel, and the CWD of
	// this test being removed if this code ran before TestBuild set a path.
	securityImagePath = path.Join(testDir, "security-container")
	defer os.Remove(imagePath)

	if b, err := imageBuild(imgCache, opts, securityImagePath, "library://busybox:1.31.1"); err != nil {
		t.Log(string(b))
		t.Fatalf("unexpected failure: %v", err)
	}

	// Security
	t.Run("Security_priv", testSecurityPriv)
	t.Run("Security_unpriv", testSecurityUnpriv)
	t.Run("Security_config_ownerships", testSecurityConfOwnership)

}
