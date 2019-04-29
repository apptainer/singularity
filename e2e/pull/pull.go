// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// This file has been migrated from cmd/singularity/pull_test.go

package pull

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/internal/pkg/test"
	"github.com/sylabs/singularity/internal/pkg/util/uri"
)

type testingEnv struct {
	CmdPath     string `split_words:"true"`
	TestDir     string `split_words:"true"`
	RunDisabled bool   `default:"false"`
}

var testenv testingEnv

var tests = []struct {
	desc            string // case description
	srcURI          string // source URI for image
	library         string // use specific library, XXX(mem): not tested yet
	force           bool   // pass --force
	createDst       bool   // create destination file before pull
	unauthenticated bool   // pass --allow-unauthenticated
	setImagePath    bool   // pass destination path
	setPullDir      bool   // pass --dir
	expectSuccess   bool   // singularity should exit with code 0
}{
	{
		desc:          "non existent image",
		srcURI:        "library://sylabs/tests/does_not_exist:0",
		expectSuccess: false,
	},

	// --allow-unauthenticated tests
	{
		desc:            "signed image known signature require authenticated",
		srcURI:          "library://alpine:3.8",
		unauthenticated: false,
		expectSuccess:   true,
	},
	{
		desc:            "signed image known signature allow unauthenticated",
		srcURI:          "library://alpine:3.8",
		unauthenticated: true,
		expectSuccess:   true,
	},
	{
		desc:            "signed image unknown signature require authenticated",
		srcURI:          "library://sylabs/tests/not-default:1.0.0",
		unauthenticated: false,
		expectSuccess:   false,
	},
	{
		desc:            "signed image unknown signature allow unauthenticated",
		srcURI:          "library://sylabs/tests/not-default:1.0.0",
		unauthenticated: true,
		expectSuccess:   true,
	},
	{
		desc:            "unsigned image require authenticated",
		srcURI:          "library://sylabs/tests/unsigned:1.0.0",
		unauthenticated: false,
		expectSuccess:   false,
	},
	{
		desc:            "unsigned image allow unauthenticated",
		srcURI:          "library://sylabs/tests/unsigned:1.0.0",
		unauthenticated: true,
		expectSuccess:   true,
	},

	// --force tests
	{
		desc:            "force existing file",
		srcURI:          "library://alpine:3.8",
		force:           true,
		createDst:       true,
		unauthenticated: true,
		expectSuccess:   true,
	},
	{
		desc:            "force non-existing file",
		srcURI:          "library://alpine:3.8",
		force:           true,
		createDst:       false,
		unauthenticated: true,
		expectSuccess:   true,
	},
	{
		// --force should not have an effect on --allow-unauthenticated=false
		desc:            "unsigned image force require authenticated",
		srcURI:          "library://sylabs/tests/unsigned:1.0.0",
		force:           true,
		unauthenticated: false,
		expectSuccess:   false,
	},
	{
		// --force should not have an effect on --allow-unauthenticated=false
		desc:            "signed image unknown signature force require authenticated",
		srcURI:          "library://sylabs/tests/not-default:1.0.0",
		force:           true,
		unauthenticated: false,
		expectSuccess:   false,
	},

	// test version specifications
	{
		desc:            "image with specific hash",
		srcURI:          "library://sylabs/tests/signed:sha256.5c439fd262095766693dae95fb81334c3a02a7f0e4dc6291e0648ed4ddc61c6c",
		unauthenticated: true,
		expectSuccess:   true,
	},
	{
		desc:            "latest tag",
		srcURI:          "library://alpine:latest",
		unauthenticated: true,
		expectSuccess:   true,
	},

	// --dir tests
	{
		desc:            "dir no image path",
		srcURI:          "library://alpine:3.9",
		unauthenticated: true,
		setPullDir:      true,
		setImagePath:    false,
		expectSuccess:   true,
	},
	{
		// XXX(mem): this specific test is passing both --path and an image path to
		// singularity pull. The current behavior is that the code is joining both paths and
		// failing to find the image in the expected location indicated by image path
		// because image path is absolute, so after joining /tmp/a/b/c and
		// /tmp/a/b/image.sif, the code expects to find /tmp/a/b/c/tmp/a/b/image.sif. Since
		// the directory /tmp/a/b/c/tmp/a/b does not exist, it fails to create the file
		// image.sif in there.
		desc:            "dir image path",
		srcURI:          "library://alpine:3.9",
		unauthenticated: true,
		setPullDir:      true,
		setImagePath:    true,
		expectSuccess:   false,
	},

	// transport tests
	{
		desc:            "bare image name",
		srcURI:          "alpine:3.8",
		force:           true,
		unauthenticated: true,
		expectSuccess:   true,
	},

	{
		desc:            "image from docker",
		srcURI:          "docker://alpine:3.8",
		force:           true,
		unauthenticated: false,
		expectSuccess:   true,
	},
	{
		desc:            "image from shub",
		srcURI:          "shub://GodloveD/busybox",
		force:           true,
		unauthenticated: false,
		expectSuccess:   true,
	},
}

func imagePull(t *testing.T, imgURI, library, pullDir, imagePath string, force, unauthenticated bool) (string, []byte, error) {
	argv := []string{"pull"}

	if force {
		argv = append(argv, "--force")
	}

	if unauthenticated {
		argv = append(argv, "--allow-unauthenticated")
	}

	if pullDir != "" {
		argv = append(argv, "--dir", pullDir)
	}

	if library != "" {
		argv = append(argv, "--library", library)
	}

	if imagePath != "" {
		argv = append(argv, imagePath)
	}

	argv = append(argv, imgURI)

	cmd := fmt.Sprintf("%s %s", testenv.CmdPath, strings.Join(argv, " "))
	out, err := exec.Command(testenv.CmdPath, argv...).CombinedOutput()

	return cmd, out, err
}

func getImageNameFromURI(imgURI string) string {
	// XXX(mem): this function should be part of the code, not the test
	switch transport, ref := uri.Split(imgURI); {
	case ref == "":
		return "" //, errInvalidURI

	case transport == "":
		imgURI = "library://" + imgURI
	}

	return uri.GetName(imgURI)
}

func testPullCmd(t *testing.T) {
	test.WithoutPrivilege(func(t *testing.T) {
		// XXX(mem): this should come from the environment
		sylabsAdminFingerprint := "8883491F4268F173C6E5DC49EDECE4F3F38D871E"
		// XXX(mem): we should not be modifying the
		// configuration of the user that is running the test,
		// this should use a temporary configuration directory
		// (set via environment variable, maybe?)
		argv := []string{"key", "pull", sylabsAdminFingerprint}
		out, err := exec.Command(testenv.CmdPath, argv...).CombinedOutput()
		if err != nil {
			t.Fatalf("Cannot pull key %q: %+v\nCommand:\n%s %s\nOutput:\n%s\n",
				sylabsAdminFingerprint,
				err,
				testenv.CmdPath, strings.Join(argv, " "),
				out)
		}
	})(t)

	for _, tt := range tests {
		t.Run(tt.desc, test.WithoutPrivilege(func(t *testing.T) {
			tmpdir, err := ioutil.TempDir(testenv.TestDir, "pull_test.")
			if err != nil {
				t.Fatalf("Failed to create temporary directory for pull test: %+v", err)
			}
			defer os.RemoveAll(tmpdir)

			var pullDir string
			if tt.setPullDir {
				pullDir, err = ioutil.TempDir(tmpdir, "pull_dir.")
				if err != nil {
					t.Fatalf("Failed to create temporary directory for pull dir: %+v", err)
				}
			}

			var imagePath, expectedImage string
			if tt.setImagePath {
				imagePath = filepath.Join(tmpdir, "image.sif")
				expectedImage = imagePath
			} else {
				// Since we are not passing an image name, change the current
				// working directory to the temporary directory we just created so
				// that we know it's clean. We don't do this for the other case in
				// order to catch spurious files showing up. Maybe later we can
				// examine the directory and assert that it only contains what we
				// expect.
				oldwd, err := os.Getwd()
				if err != nil {
					t.Fatalf("Failed to get working directory for pull test: %+v", err)
				}
				defer os.Chdir(oldwd)

				os.Chdir(tmpdir)

				// if there's a pullDir, that's where we expect to find the image
				if pullDir != "" {
					os.Chdir(pullDir)
				}

				expectedImage = getImageNameFromURI(tt.srcURI)
			}

			// In order to actually test force, there must already be a file present in
			// the expected location
			if tt.createDst {
				fh, err := os.Create(expectedImage)
				if err != nil {
					t.Fatalf("Failed to create file %q: %+v\n", expectedImage, err)
				}
				fh.Close()
			}

			cmd, out, err := imagePull(t, tt.srcURI, tt.library, pullDir, imagePath, tt.force, tt.unauthenticated)
			switch {
			case tt.expectSuccess && err == nil:
				// MAYBE PASS: expecting success, succeeded

				_, err := os.Stat(expectedImage)
				switch err {
				case nil:
					// PASS
					return

				case os.ErrNotExist:
					// FAIL
					t.Logf("Running command:\n%s\nOutput:\n%s\n", cmd, out)
					t.Errorf("expecting image at %q, not found: %+v\n", expectedImage, err)

				default:
					// FAIL
					t.Logf("Running command:\n%s\nOutput:\n%s\n", cmd, out)
					t.Errorf("unable to stat image at %q: %+v\n", expectedImage, err)
				}

				// XXX(mem): This is running a bunch of commands in the downloaded
				// images. Do we really want this here? If yes, we need to have a
				// way to do this in a generic fashion, as it's going to be shared
				// with build as well.

				// imageVerify(t, tt.imagePath, false)

			case !tt.expectSuccess && err != nil:
				// PASS: expecting failure, failed

			case tt.expectSuccess && err != nil:
				// FAIL: expecting success, failed

				t.Logf("Running command:\n%s\nOutput:\n%s\n", cmd, out)
				t.Errorf("unexpected failure: %v", err)

			case !tt.expectSuccess && err == nil:
				// FAIL: expecting failure, succeeded

				t.Logf("Running command:\n%s\nOutput:\n%s\n", cmd, out)
				t.Errorf("unexpected success: command should have failed")
			}
		}))
	}
}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(t *testing.T) {
	e2e.LoadEnv(t, &testenv)

	t.Run("pull", testPullCmd)
}
