// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package docker

import (
	"fmt"
	"os"
//	"io/ioutil"
//	"path/filepath"
	"os/exec"
	"testing"

	"github.com/kelseyhightower/envconfig"
	//	"github.com/sylabs/singularity/e2e/testutils"
//	"github.com/sylabs/singularity/e2e/imgbuild"
	"github.com/sylabs/singularity/internal/pkg/test"
	//	"golang.org/x/sys/unix"
)

type testingEnv struct {
	// base env for running tests
	CmdPath     string `split_words:"true"`
	TestDir     string `split_words:"true"`
	RunDisabled bool   `default:"false"`
}

//type buildOpts struct {
//	force   bool
//	sandbox bool
//	env     []string
//}

var testenv testingEnv
var testDir string

func testDockerPulls(t *testing.T) {
	tests := []struct {
		desc      string
		srcURI    string
		imageName string
		imagePath string
		force     bool
		sandBox       bool
		expectSuccess bool
	}{
		{
			desc:          "alpine latest pull",
			srcURI:        "docker://alpine:latest",
			imageName:     "",
			imagePath:     "",
			force:         false,
			sandBox:       false,
			expectSuccess: true,
		},
		{
			desc:          "alpine 3.9 pull",
			srcURI:        "docker://alpine:3.9",
			imageName:     "",
			imagePath:     "",
			force:         true,
			sandBox:       false,
			expectSuccess: true,
		},
		{
			desc:          "busybox pull",
			srcURI:        "docker://busybox:latest",
			imageName:     "",
			imagePath:     "",
			force:         true,
			sandBox:       false,
			expectSuccess: true,
		},
		{
			desc:          "busybox pull",
			srcURI:        "docker://busybox:1.28",
			imageName:     "",
			imagePath:     "",
			force:         false,
			sandBox:       false,
			expectSuccess: true,
		},
		{
			desc:          "busybox pull fail",
			srcURI:        "docker://busybox:1.28",
			imageName:     "",
			imagePath:     "",
			force:         false,
			sandBox:       false,
			expectSuccess: true, // TODO: WTF!! this should fail...
		},
		{
			desc:          "busybox pull",
			srcURI:        "docker://busybox:1.28",
			imageName:     "",
			imagePath:     "",
			force:         true,
			sandBox:       false,
			expectSuccess: true,
		},
	}



//	tmpdir, err := ioutil.TempDir(testenv.TestDir, "pull_test")
//	tmpImagePath, err := ioutil.TempDir(testenv.TestDir, "pull_test")
//	if err != nil {
//		t.Fatalf("Failed to create temporary directory for pull test: %+v", err)
//	}
//	tmpImagePath := filepath.Join(tmpdir, "image")
//	defer os.RemoveAll(tmpdir)

	tmpImagePath := ""

	imagePull := func(imgURI, imageName, imagePath string, force, sandBox bool) ([]byte, error) {
		argv := []string{"pull"}

		if force {
			argv = append(argv, "--force")
		}

		if sandBox {
			argv = append(argv, "--sandbox")
		}

		if imagePath == "" {
			argv = append(argv, "--path", tmpImagePath)
		} else {
			argv = append(argv, "--path", imagePath)
		}

		if imageName != "" {
			argv = append(argv, imageName)
		}

		argv = append(argv, imgURI)

		fmt.Printf("COMMAND: %s %s\n", testenv.CmdPath, argv)

		cmd := exec.Command(testenv.CmdPath, argv...)

		b, err := cmd.CombinedOutput()

		return b, err
	}

	for _, tt := range tests {
		t.Run(tt.desc, test.WithoutPrivilege(func(t *testing.T) {
//			tmpdir, err := ioutil.TempDir(testenv.TestDir, "pull_test")
//			if err != nil {
//				t.Fatalf("Failed to create temporary directory for pull test: %+v", err)
//			}
//			tmpImagePath = filepath.Join(tmpdir, "image")
//			defer os.RemoveAll(tmpdir)

			tmpImagePath := "/tmp/docker_tests"
			if err := os.RemoveAll(tmpImagePath); err != nil {
				t.Fatal("%v", err)
			}
			if err := os.MkdirAll(tmpImagePath, os.ModePerm); err != nil {
				t.Fatal("%v", err)
			}

			b, err := imagePull(tt.srcURI, tt.imageName, tmpImagePath, tt.force, tt.sandBox)
			switch {
			case tt.expectSuccess && err == nil:
				// PASS: expecting success, succeeded

				// imageVerify(t, tt.imagePath, false)

			case !tt.expectSuccess && err != nil:
				// PASS: expecting failure, failed

			case tt.expectSuccess && err != nil:
				// FAIL: expecting success, failed

				t.Log(string(b))
				t.Fatalf("unexpected failure: %v", err)

			case !tt.expectSuccess && err == nil:
				// FAIL: expecting failure, succeeded

				t.Log(string(b))
				t.Fatalf("unexpected success: command should have failed")
			}
		}))
//		defer os.RemoveAll(tmpdir)
	}
}

/*
func testDockerPulls(t *testing.T) {
	tests := []struct {
		name          string
		imagePath     string
		expectSuccess bool
	}{
		{"BusyBox", "docker://busybox", true},
		{"DoesNotExist", "docker://something_that_doesnt_exist_ever", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithPrivilege(func(t *testing.T) {
			imagePath := path.Join(testDir, "container")
			defer os.Remove(imagePath)

			opts := imgbuild.Opts{}
			//				Sandbox: tt.sandbox,
			//			}

			//			b, err := imgbuild.ImageBuild(buildOpts{}, imagePath, tt.imagePath)
			b, err := imgbuild.ImageBuild(testenv.CmdPath, opts, imagePath, tt.imagePath)
			if tt.expectSuccess {
				if err != nil {
					t.Log(string(b))
					t.Fatalf("unexpected failure: %v", err)
				}
				//				imgbuild.ImageVerify(t, imagePath, false)
				imgbuild.ImageVerify(t, testenv.CmdPath, imagePath, false, testenv.RunDisabled)
			} else if !tt.expectSuccess && err == nil {
				t.Log(string(b))
				t.Fatal("unexpected success")
			}
		}))
	}
}*/

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(t *testing.T) {
	err := envconfig.Process("E2E", &testenv)
	if err != nil {
		t.Fatal(err.Error())
	}

	t.Run("dockerPulls", testDockerPulls)
	//	t.Run("docker", testDocker)
	//	t.Run("docker", testDockerAUFS)
	//	t.Run("docker", testDockerPermissions)
	//	t.Run("docker", testDockerWhiteoutSymlink)
	//	t.Run("docker", testDockerDefFile)
}
