// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build seccomp

package security

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/kelseyhightower/envconfig"
	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/test"
)

type testingEnv struct {
	// base env for running tests
	CmdPath     string `split_words:"true"`
	TestDir     string `split_words:"true"`
	RunDisabled bool   `default:"false"`
}

var testenv testingEnv
var imagePath string

// testSecurityUnpriv tests security flag fuctionality for singularity exec without elevated privileges
func testSecurityUnpriv(t *testing.T) {
	tests := []struct {
		name          string
		image         string
		action        string
		argv          []string
		opts          e2e.ExecOpts
		exit          int
		expectSuccess bool
	}{
		// taget UID/GID
		{
			name:          "Set_uid",
			image:         imagePath,
			action:        "exec",
			argv:          []string{"id", "-u", "|", "grep", "99"},
			opts:          e2e.ExecOpts{Security: []string{"uid:99"}},
			exit:          1,
			expectSuccess: false,
		},
		{
			name:          "Set_gid",
			image:         imagePath,
			action:        "exec",
			argv:          []string{"id", "-g", "|", "grep", "99"},
			opts:          e2e.ExecOpts{Security: []string{"gid:99"}},
			exit:          1,
			expectSuccess: false,
		},
		// seccomp from json file
		{
			name:          "SecComp_BlackList",
			image:         imagePath,
			action:        "exec",
			argv:          []string{"mkdir", "/tmp/foo"},
			opts:          e2e.ExecOpts{Security: []string{"seccomp:./testdata/seccomp-profile.json"}},
			exit:          1,
			expectSuccess: false,
		},
		{
			name:          "SecComp_true",
			image:         imagePath,
			action:        "exec",
			argv:          []string{"true"},
			opts:          e2e.ExecOpts{Security: []string{"seccomp:./testdata/seccomp-profile.json"}},
			exit:          0,
			expectSuccess: true,
		},
		// capabilities
		{
			name:          "capabilities_keep_true",
			image:         imagePath,
			action:        "exec",
			argv:          []string{"ping", "-c", "1", "8.8.8.8"},
			opts:          e2e.ExecOpts{KeepPrivs: true},
			exit:          1,
			expectSuccess: false,
		},
		{
			name:          "capabilities_keep-false",
			image:         imagePath,
			action:        "exec",
			argv:          []string{"ping", "-c", "1", "8.8.8.8"},
			opts:          e2e.ExecOpts{KeepPrivs: false},
			exit:          1,
			expectSuccess: false,
		},
		{
			name:          "capabilities_drop",
			image:         imagePath,
			action:        "exec",
			argv:          []string{"ping", "-c", "1", "8.8.8.8"},
			opts:          e2e.ExecOpts{DropCaps: "CAP_NET_RAW"},
			exit:          1,
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run("unpriv "+tt.name, test.WithoutPrivilege(func(t *testing.T) {
			_, stderr, exitCode, err := e2e.ImageExec(t, testenv.CmdPath, tt.action, tt.opts, tt.image, tt.argv)
			//stdout, stderr, exitCode, err := e2e.ImageExec(t, tt.action, tt.opts, tt.image, tt.argv)
			if tt.expectSuccess && (exitCode != 0) {
				//t.Log(stdout, stderr, exitCode)
				t.Log(stderr, err, exitCode)
				t.Fatalf("unexpected failure running %q: %v", strings.Join(tt.argv, " "), err)
			} else if !tt.expectSuccess && (exitCode != 1) {
				t.Log(stderr, err, exitCode)
				t.Fatalf("unexpected success running %q", strings.Join(tt.argv, " "))
			}
		}))
	}
}

// testSecurityPriv tests security flag fuctionality for singularity exec with elevated privileges
func testSecurityPriv(t *testing.T) {
	tests := []struct {
		name          string
		image         string
		action        string
		argv          []string
		opts          e2e.ExecOpts
		exit          int
		expectSuccess bool
	}{
		// taget UID/GID
		{
			name:          "Set_uid",
			image:         imagePath,
			action:        "exec",
			argv:          []string{"id", "-u", "|", "grep", "99"},
			opts:          e2e.ExecOpts{Security: []string{"uid:99"}},
			exit:          1,
			expectSuccess: false,
		},
		{
			name:          "Set_gid",
			image:         imagePath,
			action:        "exec",
			argv:          []string{"id", "-g", "|", "grep", "99"},
			opts:          e2e.ExecOpts{Security: []string{"gid:99"}},
			exit:          1,
			expectSuccess: false,
		},
		// seccomp from json file
		{
			name:          "SecComp_BlackList",
			image:         imagePath,
			action:        "exec",
			argv:          []string{"mkdir", "/tmp/foo"},
			opts:          e2e.ExecOpts{Security: []string{"seccomp:./testdata/seccomp-profile.json"}},
			exit:          1,
			expectSuccess: false,
		},
		{
			name:          "SecComp_true",
			image:         imagePath,
			action:        "exec",
			argv:          []string{"true"},
			opts:          e2e.ExecOpts{Security: []string{"seccomp:./testdata/seccomp-profile.json"}},
			exit:          0,
			expectSuccess: true,
		},
		// capabilities
		{
			name:          "capabilities_keep",
			image:         imagePath,
			action:        "exec",
			argv:          []string{"ping", "-c", "1", "8.8.8.8"},
			opts:          e2e.ExecOpts{KeepPrivs: true},
			exit:          0,
			expectSuccess: true,
		},
		{
			name:          "capabilities_drop",
			image:         imagePath,
			action:        "exec",
			argv:          []string{"ping", "-c", "1", "8.8.8.8"},
			opts:          e2e.ExecOpts{DropCaps: "CAP_NET_RAW"},
			exit:          1,
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run("priv "+tt.name, test.WithPrivilege(func(t *testing.T) {
			//			stdout, stderr, exitCode, err := imageExec(t, tt.action, tt.opts, tt.image, tt.argv)
			_, stderr, exitCode, err := e2e.ImageExec(t, testenv.CmdPath, tt.action, tt.opts, tt.image, tt.argv)
			if tt.expectSuccess && (exitCode != 0) {
				t.Log(stderr, err, exitCode)
				t.Fatalf("unexpected failure running '%v': %v", strings.Join(tt.argv, " "), err)
			} else if !tt.expectSuccess && (exitCode != 1) {
				t.Log(stderr, err, exitCode)
				t.Fatalf("unexpected success running '%v'", strings.Join(tt.argv, " "))
			}
		}))
	}
}

// testSecurityConfOwnership tests checks on config files ownerships
func testSecurityConfOwnership(t *testing.T) {
	configFile := buildcfg.SYSCONFDIR + "/singularity/singularity.conf"
	// Change file ownership (do not try this at home)
	err := os.Chown(configFile, 1001, 0)
	if err != nil {
		t.Fatal(err)
	}

	// try to run
	t.Run("non_root_config", test.WithoutPrivilege(func(t *testing.T) {
		//_, stderr, exitCode, err := imageExec(t, "exec", opts{}, imagePath, []string{"/bin/true"})
		_, stderr, exitCode, err := e2e.ImageExec(t, testenv.CmdPath, "exec", e2e.ExecOpts{}, imagePath, []string{"/bin/true"})
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

func testSecurity(t *testing.T) {
	test.EnsurePrivilege(t)
	opts := e2e.BuildOpts{
		Force:   true,
		Sandbox: false,
	}
	if b, err := e2e.ImageBuild(testenv.CmdPath, opts, imagePath, "../../examples/busybox/Singularity"); err == nil {
		//if b, err := imageBuild(opts, imagePath, "../../examples/busybox/Singularity"); err != nil {
		t.Log(string(b))
		t.Fatalf("Unexpected failure: %v", err)
	}
	defer os.Remove(imagePath)

	// Security
	t.Run("Security_unpriv", testSecurityPriv)
	t.Run("Security_priv", testSecurityUnpriv)
	t.Run("Security_config_ownerships", testSecurityConfOwnership)

}

// pullTestContainer ...
func pullTestContainer(t *testing.T) {

	//argv := []string{"pull", "-U", "--dir", imagePath, "library://alpine:latest"}
	argv := []string{"pull", "-U", imagePath, "library://alpine:latest"}

	fmt.Println("IMAGEPATH: ", imagePath)

	cmd := exec.Command(testenv.CmdPath, argv...)
	b, err := cmd.CombinedOutput()

	if err != nil {
		t.Log(string(b))
		t.Fatalf("Unable to pull test container: %s", err)
	}
}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(t *testing.T) {
	err := envconfig.Process("E2E", &testenv)
	if err != nil {
		t.Fatal(err.Error())
	}

	imagePath = e2e.MakeTmpDir(t)
	imagePath += "test_container.sif"

	t.Run("pulling_test_contianer", pullTestContainer)
	t.Run("testSecurity", testSecurity)
}
