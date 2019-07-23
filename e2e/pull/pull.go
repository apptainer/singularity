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

	"github.com/containerd/containerd/reference"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/deislabs/oras/pkg/content"
	"github.com/deislabs/oras/pkg/context"
	"github.com/deislabs/oras/pkg/oras"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/internal/pkg/util/uri"
)

type ctx struct {
	env e2e.TestEnv
}

type testStruct struct {
	desc             string // case description
	srcURI           string // source URI for image
	library          string // use specific library, XXX(mem): not tested yet
	force            bool   // pass --force
	createDst        bool   // create destination file before pull
	unauthenticated  bool   // pass --allow-unauthenticated
	setImagePath     bool   // pass destination path
	setPullDir       bool   // pass --dir
	expectedExitCode int
	pullDir          string
	imagePath        string
	expectedImage    string
}

var tests = []testStruct{
	{
		desc:             "non existent image",
		srcURI:           "library://sylabs/tests/does_not_exist:0",
		expectedExitCode: 255,
	},

	// --allow-unauthenticated tests
	{
		desc:             "unsigned image allow unauthenticated",
		srcURI:           "library://sylabs/tests/unsigned:1.0.0",
		unauthenticated:  true,
		expectedExitCode: 0,
	},

	// --force tests
	{
		desc:             "force existing file",
		srcURI:           "library://alpine:3.8",
		force:            true,
		createDst:        true,
		unauthenticated:  true,
		expectedExitCode: 0,
	},
	{
		desc:             "force non-existing file",
		srcURI:           "library://alpine:3.8",
		force:            true,
		createDst:        false,
		unauthenticated:  true,
		expectedExitCode: 0,
	},
	{
		// --force should not have an effect on --allow-unauthenticated=false
		desc:             "unsigned image force require authenticated",
		srcURI:           "library://sylabs/tests/unsigned:1.0.0",
		force:            true,
		unauthenticated:  false,
		expectedExitCode: 10,
	},

	// test version specifications
	{
		desc:             "image with specific hash",
		srcURI:           "library://sylabs/tests/signed:sha256.5c439fd262095766693dae95fb81334c3a02a7f0e4dc6291e0648ed4ddc61c6c",
		unauthenticated:  true,
		expectedExitCode: 0,
	},
	{
		desc:             "latest tag",
		srcURI:           "library://alpine:latest",
		unauthenticated:  true,
		expectedExitCode: 0,
	},

	// --dir tests
	{
		desc:             "dir no image path",
		srcURI:           "library://alpine:3.9",
		unauthenticated:  true,
		setPullDir:       true,
		setImagePath:     false,
		expectedExitCode: 0,
	},
	{
		// XXX(mem): this specific test is passing both --path and an image path to
		// singularity pull. The current behavior is that the code is joining both paths and
		// failing to find the image in the expected location indicated by image path
		// because image path is absolute, so after joining /tmp/a/b/c and
		// /tmp/a/b/image.sif, the code expects to find /tmp/a/b/c/tmp/a/b/image.sif. Since
		// the directory /tmp/a/b/c/tmp/a/b does not exist, it fails to create the file
		// image.sif in there.
		desc:             "dir image path",
		srcURI:           "library://alpine:3.9",
		unauthenticated:  true,
		setPullDir:       true,
		setImagePath:     true,
		expectedExitCode: 255,
	},

	// transport tests
	{
		desc:             "bare image name",
		srcURI:           "alpine:3.8",
		force:            true,
		unauthenticated:  true,
		expectedExitCode: 0,
	},

	{
		desc:             "image from docker",
		srcURI:           "docker://alpine:3.8",
		force:            true,
		unauthenticated:  false,
		expectedExitCode: 0,
	},
	// TODO(mem): reenable this; disabled while shub is down
	// {
	// 	desc:            "image from shub",
	// 	srcURI:          "shub://GodloveD/busybox",
	// 	force:           true,
	// 	unauthenticated: false,
	// 	expectSuccess:   true,
	// },
	{
		desc:             "oras transport for SIF from registry",
		srcURI:           "oras://localhost:5000/pull_test_sif:latest", // TODO(mem): obtain registry from context
		force:            true,
		unauthenticated:  false,
		expectedExitCode: 0,
	},

	// pulling of invalid images with oras
	{
		desc:             "oras pull of non SIF file",
		srcURI:           "oras://localhost:5000/pull_test_:latest", // TODO(mem): obtain registry from context
		force:            true,
		expectedExitCode: 255,
	},
	{
		desc:             "oras pull of packed dir",
		srcURI:           "oras://localhost:5000/pull_test_invalid_file:latest", // TODO(mem): obtain registry from context
		force:            true,
		expectedExitCode: 255,
	},

	// pulling with library URI argument
	{
		desc:             "bad library URI",
		srcURI:           "library://busybox",
		library:          "https://bad-library.sylabs.io",
		expectedExitCode: 255,
	},
	{
		desc:             "default library URI",
		srcURI:           "library://busybox",
		library:          "https://library.sylabs.io",
		force:            true,
		expectedExitCode: 0,
	},
}

func (c *ctx) imagePull(t *testing.T, tt testStruct) {
	// We use a string rather than a slice of strings to avoid having an empty
	// element in the slice, which would cause the command to fail, wihtout
	// over-complicating the code.
	argv := ""

	if tt.force {
		argv += "--force "
	}

	if tt.unauthenticated {
		argv += "--allow-unauthenticated "
	}

	if tt.pullDir != "" {
		argv += "--dir " + tt.pullDir + " "
	}

	if tt.library != "" {
		argv += "--library " + tt.library + " "
	}

	if tt.imagePath != "" {
		argv += tt.imagePath + " "
	}

	argv += tt.srcURI

	c.env.RunSingularity(
		t,
		e2e.AsSubtest(tt.desc),
		e2e.WithPrivileges(false),
		e2e.WithCommand("pull"),
		e2e.WithArgs(strings.Split(argv, " ")...),
		// We make sure to include os.Environ() in addition of KeyringDir to find utilities such as mksquashfs.
		// If nothing is specified with WithEnv(), the framework will pick up os.Environ(); if we need to had a
		// new environment variable, we need to explicitly also add os.Environ()
		e2e.WithEnv(append(os.Environ(), c.env.KeyringDir)),
		e2e.ExpectExit(tt.expectedExitCode))

	checkPullResult(t, tt)
}

func getImageNameFromURI(imgURI string) string {
	// XXX(mem): this function should be part of the code, not the test
	switch transport, ref := uri.Split(imgURI); {
	case ref == "":
		return "" // Invalid URI

	case transport == "":
		imgURI = "library://" + imgURI
	}

	return uri.GetName(imgURI)
}

func (c *ctx) setup(t *testing.T) {
	e2e.EnsureImage(t, c.env)
	e2e.PrepRegistry(t, c.env)

	// setup file and dir to use as invalid images
	orasInvalidDir, err := ioutil.TempDir(c.env.TestDir, "oras_push_dir-")
	if err != nil {
		t.Fatalf("unable to create src dir for push tests: %v", err)
	}

	orasInvalidFile, err := e2e.WriteTempFile(orasInvalidDir, "oras_invalid_image-", "Invalid Image Contents")
	if err != nil {
		t.Fatalf("unable to create src file for push tests: %v", err)
	}

	// prep local registry with oras generated artifacts
	// Note: the image name prevents collisions by using a package specific name
	// as the registry is shared between different test packages
	orasImages := []struct {
		srcPath string
		uri     string
	}{
		{
			srcPath: c.env.ImagePath,
			uri:     fmt.Sprintf("%s/pull_test_sif:latest", c.env.TestRegistry),
		},
		{
			srcPath: orasInvalidDir,
			uri:     fmt.Sprintf("%s/pull_test_dir:latest", c.env.TestRegistry),
		},
		{
			srcPath: orasInvalidFile,
			uri:     fmt.Sprintf("%s/pull_test_invalid_file:latest", c.env.TestRegistry),
		},
	}

	for _, i := range orasImages {
		err = orasPushNoCheck(i.srcPath, i.uri)
		if err != nil {
			t.Fatalf("while prepping registry for oras tests: %v", err)
		}
	}
}

func (c *ctx) testPullCmd(t *testing.T) {
	// XXX(mem): this should come from the environment
	sylabsAdminFingerprint := "8883491F4268F173C6E5DC49EDECE4F3F38D871E"
	tempKeyringDir, err := ioutil.TempDir("", "tempKeyringDir-")
	if err != nil {
		t.Fatalf("failed to create temporary directory: %s", err)
	}
	defer os.RemoveAll(tempKeyringDir)

	keyringEnv := "SINGULARITY_SYPGPDIR=" + tempKeyringDir
	c.env.KeyringDir = keyringEnv
	argv := []string{"key", "pull", sylabsAdminFingerprint}
	out, err := exec.Command(c.env.CmdPath, argv...).CombinedOutput()
	if err != nil {
		t.Fatalf("Cannot pull key %q: %+v\nCommand:\n%s %s\nOutput:\n%s\n",
			sylabsAdminFingerprint,
			err,
			c.env.CmdPath, strings.Join(argv, " "),
			out)
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			tmpdir, err := ioutil.TempDir(c.env.TestDir, "pull_test.")
			if err != nil {
				t.Fatalf("Failed to create temporary directory for pull test: %+v", err)
			}
			defer os.RemoveAll(tmpdir)

			if tt.setPullDir {
				tt.pullDir, err = ioutil.TempDir(tmpdir, "pull_dir.")
				if err != nil {
					t.Fatalf("Failed to create temporary directory for pull dir: %+v", err)
				}
			}

			if tt.setImagePath {
				tt.imagePath = filepath.Join(tmpdir, "image.sif")
				tt.expectedImage = tt.imagePath
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
				if tt.pullDir != "" {
					os.Chdir(tt.pullDir)
				}

				tt.expectedImage = getImageNameFromURI(tt.srcURI)
			}

			// In order to actually test force, there must already be a file present in
			// the expected location
			if tt.createDst {
				fh, err := os.Create(tt.expectedImage)
				if err != nil {
					t.Fatalf("failed to create file %q: %+v\n", tt.expectedImage, err)
				}
				fh.Close()
			}

			c.imagePull(t, tt)
		})
	}
}

func checkPullResult(t *testing.T, tt testStruct) {
	if tt.expectedExitCode == 0 {
		_, err := os.Stat(tt.expectedImage)
		switch err {
		case nil:
			// PASS
			return

		case os.ErrNotExist:
			// FAIL
			t.Errorf("expecting image at %q, not found: %+v\n", tt.expectedImage, err)

		default:
			// FAIL
			t.Errorf("unable to stat image at %q: %+v\n", tt.expectedImage, err)
		}

		// XXX(mem): This is running a bunch of commands in the downloaded
		// images. Do we really want this here? If yes, we need to have a
		// way to do this in a generic fashion, as it's going to be shared
		// with build as well.

		// imageVerify(t, tt.imagePath, false)
	}
}

// this is a version of the oras push functionality that does not check that given the
// file is a valid SIF, this allows us to push arbitrary objects to the local registry
// to test the pull validation
func orasPushNoCheck(file, ref string) error {
	ref = strings.TrimPrefix(ref, "//")

	spec, err := reference.Parse(ref)
	if err != nil {
		return fmt.Errorf("unable to parse oci reference: %s", err)
	}

	// Hostname() will panic if there is no '/' in the locator
	// explicitly check for this and fail in order to prevent panic
	// this case will only occur for incorrect uris
	if !strings.Contains(spec.Locator, "/") {
		return fmt.Errorf("not a valid oci object uri: %s", ref)
	}

	// append default tag if no object exists
	if spec.Object == "" {
		spec.Object = "latest"
	}

	resolver := docker.NewResolver(docker.ResolverOptions{})

	store := content.NewFileStore("")
	defer store.Close()

	conf, err := store.Add("$config", "application/vnd.sylabs.sif.config.v1+json", "/dev/null")
	if err != nil {
		return fmt.Errorf("unable to add manifest config to FileStore: %s", err)
	}
	conf.Annotations = nil

	// use last element of filepath as file name in annotation
	fileName := filepath.Base(file)
	desc, err := store.Add(fileName, "appliciation/vnd.sylabs.sif.layer.tar", file)
	if err != nil {
		return fmt.Errorf("unable to add SIF file to FileStore: %s", err)
	}

	descriptors := []ocispec.Descriptor{desc}

	if _, err := oras.Push(context.Background(), resolver, spec.String(), store, descriptors, oras.WithConfig(conf)); err != nil {
		return fmt.Errorf("unable to push: %s", err)
	}

	return nil
}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(env e2e.TestEnv) func(*testing.T) {
	c := &ctx{
		env: env,
	}

	return func(t *testing.T) {
		c.setup(t)
		t.Run("pull", c.testPullCmd)
	}
}
