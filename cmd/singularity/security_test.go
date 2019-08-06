// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build seccomp

package main

import (
	"os"
	"strings"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
	"github.com/sylabs/singularity/internal/pkg/util/fs/files"
)

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
		{"Set_uid", imagePath, "exec", []string{"id", "-u", "|", "grep", "99"}, opts{security: []string{"uid:99"}}, 1, false},
		{"Set_gid", imagePath, "exec", []string{"id", "-g", "|", "grep", "99"}, opts{security: []string{"gid:99"}}, 1, false},
		// seccomp from json file
		{"SecComp_BlackList", imagePath, "exec", []string{"mkdir", "/tmp/foo"}, opts{security: []string{"seccomp:./testdata/seccomp-profile.json"}}, 1, false},
		{"SecComp_true", imagePath, "exec", []string{"true"}, opts{security: []string{"seccomp:./testdata/seccomp-profile.json"}}, 0, true},
		// capabilities
		{"capabilities_keep_true", imagePath, "exec", []string{"ping", "-c", "1", "8.8.8.8"}, opts{keepPrivs: true}, 1, false},
		{"capabilities_keep-false", imagePath, "exec", []string{"ping", "-c", "1", "8.8.8.8"}, opts{keepPrivs: false}, 1, false},
		{"capabilities_drop", imagePath, "exec", []string{"ping", "-c", "1", "8.8.8.8"}, opts{dropCaps: "CAP_NET_RAW"}, 1, false},
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
		{"Set_uid", imagePath, "exec", []string{"id", "-u", "|", "grep", "99"}, opts{security: []string{"uid:99"}}, 1, false},
		{"Set_gid", imagePath, "exec", []string{"id", "-g", "|", "grep", "99"}, opts{security: []string{"gid:99"}}, 1, false},
		// seccomp from json file
		{"SecComp_BlackList", imagePath, "exec", []string{"mkdir", "/tmp/foo"}, opts{security: []string{"seccomp:./testdata/seccomp-profile.json"}}, 1, false},
		{"SecComp_true", imagePath, "exec", []string{"true"}, opts{security: []string{"seccomp:./testdata/seccomp-profile.json"}}, 0, true},
		// capabilities
		{"capabilities_keep", imagePath, "exec", []string{"ping", "-c", "1", "8.8.8.8"}, opts{keepPrivs: true}, 0, true},
		{"capabilities_drop", imagePath, "exec", []string{"ping", "-c", "1", "8.8.8.8"}, opts{dropCaps: "CAP_NET_RAW"}, 1, false},
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
	configFile := files.GetSysConfigFile()
	// Change file ownership (do not try this at home)
	err := os.Chown(configFile, 1001, 0)
	if err != nil {
		t.Fatal(err)
	}

	// try to run
	t.Run("non_root_config", test.WithoutPrivilege(func(t *testing.T) {
		_, stderr, exitCode, err := imageExec(t, "exec", opts{}, imagePath, []string{"/bin/true"})
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

	if b, err := imageBuild(imgCache, opts, imagePath, "../../examples/busybox/Singularity"); err != nil {
		t.Log(string(b))
		t.Fatalf("unexpected failure: %v", err)
	}
	defer os.Remove(imagePath)

	// Security
	t.Run("Security_unpriv", testSecurityPriv)
	t.Run("Security_priv", testSecurityUnpriv)
	t.Run("Security_config_ownerships", testSecurityConfOwnership)

}
