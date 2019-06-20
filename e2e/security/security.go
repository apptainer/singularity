// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package security

import (
	"bytes"
	"io/ioutil"
	"os"
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
		expectID      string
		argv          []string
		opts          e2e.ExecOpts
		exit          int
		expectSuccess bool
	}{
		// taget UID/GID
		{
			name:          "Set_uid",
			argv:          []string{"id", "-u"},
			opts:          e2e.ExecOpts{Security: []string{"uid:99"}},
			expectID:      "99",
			exit:          1,
			expectSuccess: false,
		},
		{
			name:          "Set_gid",
			argv:          []string{"id", "-g"},
			opts:          e2e.ExecOpts{Security: []string{"gid:99"}},
			expectID:      "99",
			exit:          1,
			expectSuccess: false,
		},
		// seccomp from json file
		{
			name:          "SecComp_BlackList",
			argv:          []string{"mkdir", "/tmp/foo"},
			opts:          e2e.ExecOpts{Security: []string{"seccomp:./testdata/seccomp-profile.json"}},
			exit:          1,
			expectSuccess: false,
		},
		{
			name:          "SecComp_true",
			argv:          []string{"true"},
			opts:          e2e.ExecOpts{Security: []string{"seccomp:./testdata/seccomp-profile.json"}},
			exit:          0,
			expectSuccess: true,
		},
		// capabilities
		{
			name:          "capabilities_keep_true",
			argv:          []string{"ping", "-c", "1", "8.8.8.8"},
			opts:          e2e.ExecOpts{KeepPrivs: true},
			exit:          1,
			expectSuccess: false,
		},
		{
			name:          "capabilities_keep-false",
			argv:          []string{"ping", "-c", "1", "8.8.8.8"},
			opts:          e2e.ExecOpts{KeepPrivs: false},
			exit:          1,
			expectSuccess: false,
		},
		{
			name:          "capabilities_drop",
			argv:          []string{"ping", "-c", "1", "8.8.8.8"},
			opts:          e2e.ExecOpts{DropCaps: "CAP_NET_RAW"},
			exit:          1,
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run("unpriv "+tt.name, test.WithoutPrivilege(func(t *testing.T) {
			stdout, stderr, exitCode, err := e2e.ImageExec(t, testenv.CmdPath, "exec", tt.opts, imagePath, tt.argv)

			switch {
			case tt.expectSuccess && tt.exit == 0:
				// expect success, command succeeded
				lines := bytes.Fields([]byte(stdout))
				if len(lines) == 1 && string(lines[0]) != tt.expectID {
					t.Fatal("test failed? expecting: 99, got: ", string(lines[0]))
				}

			case !tt.expectSuccess && tt.exit != 0:
				// expect failure, command failed

			case tt.expectSuccess && tt.exit != 0:
				// expect success, command failed
				t.Log(stderr, err, exitCode)
				t.Fatalf("unexpected failure running %q: %v", strings.Join(tt.argv, " "), err)

			case !tt.expectSuccess && tt.exit == 0:
				// expect failure, command succeeded
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
		expectID      string
		argv          []string
		opts          e2e.ExecOpts
		exit          int
		expectSuccess bool
	}{
		// taget UID/GID
		{
			name:          "Set_uid",
			argv:          []string{"id", "-u"},
			opts:          e2e.ExecOpts{Security: []string{"uid:99"}},
			expectID:      "99",
			exit:          0,
			expectSuccess: true,
		},
		{
			name:          "Set_gid",
			argv:          []string{"id", "-g"},
			opts:          e2e.ExecOpts{Security: []string{"gid:99"}},
			expectID:      "99",
			exit:          0,
			expectSuccess: true,
		},
		// seccomp from json file
		{
			name:          "SecComp_BlackList",
			argv:          []string{"mkdir", "/tmp/foo"},
			opts:          e2e.ExecOpts{Security: []string{"seccomp:./testdata/seccomp-profile.json"}},
			exit:          1,
			expectSuccess: false,
		},
		{
			name:          "SecComp_true",
			argv:          []string{"true"},
			opts:          e2e.ExecOpts{Security: []string{"seccomp:./testdata/seccomp-profile.json"}},
			exit:          0,
			expectSuccess: true,
		},
		// capabilities
		{
			name:          "capabilities_keep",
			argv:          []string{"ping", "-c", "1", "8.8.8.8"},
			opts:          e2e.ExecOpts{KeepPrivs: true},
			exit:          0,
			expectSuccess: true,
		},
		{
			name:          "capabilities_drop",
			argv:          []string{"ping", "-c", "1", "8.8.8.8"},
			opts:          e2e.ExecOpts{DropCaps: "CAP_NET_RAW"},
			exit:          1,
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run("priv "+tt.name, test.WithPrivilege(func(t *testing.T) {
			stdout, stderr, exitCode, err := e2e.ImageExec(t, testenv.CmdPath, "exec", tt.opts, imagePath, tt.argv)

			switch {
			case tt.expectSuccess && tt.exit == 0:
				// expect success, command succeeded
				lines := bytes.Fields([]byte(stdout))
				if len(lines) == 1 && string(lines[0]) != tt.expectID {
					t.Fatal("test failed? expecting: 99, got: ", string(lines[0]))
				}

			case !tt.expectSuccess && tt.exit != 0:
				// expect failure, command failed

			case tt.expectSuccess && tt.exit != 0:
				// expect success, command failed
				t.Log(stderr, err, exitCode)
				t.Fatalf("unexpected failure running %q: %v", strings.Join(tt.argv, " "), err)

			case !tt.expectSuccess && tt.exit == 0:
				// expect failure, command succeeded
				t.Log(stderr, err, exitCode)
				t.Fatalf("unexpected success running %q", strings.Join(tt.argv, " "))
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

	// Security
	t.Run("Security_unpriv", testSecurityPriv)
	t.Run("Security_priv", testSecurityUnpriv)
	t.Run("Security_config_ownerships", testSecurityConfOwnership)

}

func pullAlpineTest(t *testing.T) {
	// Make a tmp file
	file, err := ioutil.TempFile(testenv.TestDir, "test_container.sif")
	if err != nil {
		t.Fatal("unable to make tmp file: ", err)
	}
	defer file.Close()

	imagePath = file.Name()

	t.Log("FFFFFFFFFFFFFFFFFFFF: ", imagePath)

	b, err := e2e.PullTestAlpineContainer(testenv.CmdPath, imagePath)
	if err != nil {
		t.Log(string(b))
		t.Fatalf("Unable to pull test alpine container: %s\n", err)
	}

}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(t *testing.T) {
	err := envconfig.Process("E2E", &testenv)
	if err != nil {
		t.Fatal(err.Error())
	}

	pullAlpineTest(t)

	// pull a test image to that directory
	//t.Run("non_root_config",
	/*	test.WithoutPrivilege(func(t *testing.T) {

			// Make a tmp file
			file, err := ioutil.TempFile(testenv.TestDir, "test_container.sif")
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()

			imagePath = file.Name()

			t.Log("FFFFFFFFFFFFFFFFFFFF: ", imagePath)

			b, err := e2e.PullTestAlpineContainer(testenv.CmdPath, imagePath)
			if err != nil {
				t.Log(string(b))
				t.Fatalf("Unable to pull test alpine container: %s", err)
			}
		}) //)

	*/

	t.Log("#####################: ", imagePath)

	t.Run("testSecurity", testSecurity)
}
