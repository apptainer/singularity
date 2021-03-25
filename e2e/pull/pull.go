// Copyright (c) 2019-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// This file has been migrated from cmd/singularity/pull_test.go

package pull

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/containerd/containerd/reference"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/deislabs/oras/pkg/content"
	"github.com/deislabs/oras/pkg/context"
	"github.com/deislabs/oras/pkg/oras"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/e2e/internal/testhelper"
	syoras "github.com/sylabs/singularity/internal/pkg/client/oras"
	"github.com/sylabs/singularity/internal/pkg/util/uri"
	"golang.org/x/sys/unix"
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
		srcURI:           "library://alpine:3.11.5",
		force:            true,
		createDst:        true,
		unauthenticated:  true,
		expectedExitCode: 0,
	},
	{
		desc:             "force non-existing file",
		srcURI:           "library://alpine:3.11.5",
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
		expectedExitCode: 0,
	},

	// test version specifications
	{
		desc:             "image with specific hash",
		srcURI:           "library://alpine:sha256.03883ca565b32e58fa0a496316d69de35741f2ef34b5b4658a6fec04ed8149a8",
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
		srcURI:           "library://alpine:3.11.5",
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
		srcURI:           "library://alpine:3.11.5",
		unauthenticated:  true,
		setPullDir:       true,
		setImagePath:     true,
		expectedExitCode: 255,
	},

	// transport tests
	{
		desc:             "bare image name",
		srcURI:           "alpine:3.11.5",
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
	// Finalized v1 layer mediaType (3.7 and onward)
	{
		desc:             "oras transport for SIF from registry",
		srcURI:           "oras://localhost:5000/pull_test_sif:latest", // TODO(mem): obtain registry from context
		force:            true,
		unauthenticated:  false,
		expectedExitCode: 0,
	},
	// Original/prototype layer mediaType (<3.7)
	{
		desc:             "oras transport for SIF from registry (SifLayerMediaTypeProto)",
		srcURI:           "oras://localhost:5000/pull_test_sif_mediatypeproto:latest", // TODO(mem): obtain registry from context
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
		srcURI:           "library://busybox:1.31.1",
		library:          "https://bad-library.sylabs.io",
		expectedExitCode: 255,
	},
	{
		desc:             "default library URI",
		srcURI:           "library://busybox:1.31.1",
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
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("pull"),
		e2e.WithArgs(strings.Split(argv, " ")...),
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
	e2e.EnsureRegistry(t)

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
		srcPath        string
		uri            string
		layerMediaType string
	}{
		{
			srcPath:        c.env.ImagePath,
			uri:            fmt.Sprintf("%s/pull_test_sif:latest", c.env.TestRegistry),
			layerMediaType: syoras.SifLayerMediaTypeV1,
		},
		{
			srcPath:        c.env.ImagePath,
			uri:            fmt.Sprintf("%s/pull_test_sif_mediatypeproto:latest", c.env.TestRegistry),
			layerMediaType: syoras.SifLayerMediaTypeProto,
		},
		{
			srcPath:        orasInvalidDir,
			uri:            fmt.Sprintf("%s/pull_test_dir:latest", c.env.TestRegistry),
			layerMediaType: syoras.SifLayerMediaTypeV1,
		},
		{
			srcPath:        orasInvalidFile,
			uri:            fmt.Sprintf("%s/pull_test_invalid_file:latest", c.env.TestRegistry),
			layerMediaType: syoras.SifLayerMediaTypeV1,
		},
	}

	for _, i := range orasImages {
		err = orasPushNoCheck(i.srcPath, i.uri, i.layerMediaType)
		if err != nil {
			t.Fatalf("while prepping registry for oras tests: %v", err)
		}
	}
}

func (c ctx) testPullCmd(t *testing.T) {
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
// We can also set the layer mediaType - so we can push images with older media types
// to verify that they can still be pulled.
func orasPushNoCheck(file, ref, layerMediaType string) error {
	ref = strings.TrimPrefix(ref, "//")

	spec, err := reference.Parse(ref)
	if err != nil {
		err = errors.Wrapf(err, "parse OCI reference %s", ref)
		return fmt.Errorf("unable to parse oci reference: %+v", err)
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

	conf, err := store.Add("$config", syoras.SifConfigMediaTypeV1, "/dev/null")
	if err != nil {
		err = errors.Wrap(err, "adding manifest config to file store")
		return fmt.Errorf("unable to add manifest config to FileStore: %+v", err)
	}
	conf.Annotations = nil

	// use last element of filepath as file name in annotation
	fileName := filepath.Base(file)
	desc, err := store.Add(fileName, layerMediaType, file)
	if err != nil {
		err = errors.Wrap(err, "adding manifest SIF file to file store")
		return fmt.Errorf("unable to add SIF file to FileStore: %+v", err)
	}

	descriptors := []ocispec.Descriptor{desc}

	if _, err := oras.Push(context.Background(), resolver, spec.String(), store, descriptors, oras.WithConfig(conf)); err != nil {
		err = errors.Wrap(err, "pushing to oras")
		return fmt.Errorf("unable to push: %+v", err)
	}

	return nil
}

func (c ctx) testPullDisableCacheCmd(t *testing.T) {
	cacheDir, err := ioutil.TempDir("", "e2e-imgcache-")
	if err != nil {
		t.Fatalf("failed to create temporary directory: %s", err)
	}
	defer func() {
		err := os.RemoveAll(cacheDir)
		if err != nil {
			t.Fatalf("failed to delete temporary directory %s: %s", cacheDir, err)
		}
	}()

	c.env.ImgCacheDir = cacheDir

	disableCacheTests := []struct {
		name      string
		imagePath string
		imageSrc  string
	}{
		{
			name:      "library",
			imagePath: filepath.Join(c.env.TestDir, "library.sif"),
			imageSrc:  "library://alpine:latest",
		},
		{
			name:      "docker",
			imagePath: filepath.Join(c.env.TestDir, "docker.sif"),
			imageSrc:  "docker://alpine:latest",
		},
		{
			name:      "oras",
			imagePath: filepath.Join(c.env.TestDir, "oras.sif"),
			imageSrc:  "oras://localhost:5000/pull_test_sif:latest",
		},
	}

	for _, tt := range disableCacheTests {
		cmdArgs := []string{"--disable-cache", tt.imagePath, tt.imageSrc}
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("pull"),
			e2e.WithArgs(cmdArgs...),
			e2e.ExpectExit(0),
			e2e.PostRun(func(t *testing.T) {
				// Cache entry must not have been created
				cacheEntryPath := filepath.Join(cacheDir, "cache")
				if _, err := os.Stat(cacheEntryPath); !os.IsNotExist(err) {
					t.Errorf("cache created while disabled (%s exists)", cacheEntryPath)
				}
				// We also need to check the image pulled is in the correct place!
				// Issue #5628s
				_, err := os.Stat(tt.imagePath)
				if os.IsNotExist(err) {
					t.Errorf("image does not exist at %s", tt.imagePath)
				}
			}),
		)
	}
}

// testPullUmask will run some pull tests with different umasks, and
// ensure the output file hase the correct permissions.
func (c ctx) testPullUmask(t *testing.T) {
	umask22Image := "0022-umask-pull"
	umask77Image := "0077-umask-pull"
	umask27Image := "0027-umask-pull"

	umaskTests := []struct {
		name       string
		imagePath  string
		umask      int
		expectPerm uint32
		force      bool
	}{
		{
			name:       "0022 umask pull",
			imagePath:  filepath.Join(c.env.TestDir, umask22Image),
			umask:      0022,
			expectPerm: 0755,
		},
		{
			name:       "0077 umask pull",
			imagePath:  filepath.Join(c.env.TestDir, umask77Image),
			umask:      0077,
			expectPerm: 0700,
		},
		{
			name:       "0027 umask pull",
			imagePath:  filepath.Join(c.env.TestDir, umask27Image),
			umask:      0027,
			expectPerm: 0750,
		},

		// With the force flag, and overide the image. The permission will
		// reset to 0666 after every test.
		{
			name:       "0022 umask pull overide",
			imagePath:  filepath.Join(c.env.TestDir, umask22Image),
			umask:      0022,
			expectPerm: 0755,
			force:      true,
		},
		{
			name:       "0077 umask pull overide",
			imagePath:  filepath.Join(c.env.TestDir, umask77Image),
			umask:      0077,
			expectPerm: 0700,
			force:      true,
		},
		{
			name:       "0027 umask pull overide",
			imagePath:  filepath.Join(c.env.TestDir, umask27Image),
			umask:      0027,
			expectPerm: 0750,
			force:      true,
		},
	}

	// Helper function to get the file mode for a file.
	getFilePerm := func(t *testing.T, path string) uint32 {
		finfo, err := os.Stat(path)
		if err != nil {
			t.Fatalf("failed while getting file permission: %s", err)
		}
		return uint32(finfo.Mode().Perm())
	}

	// Set a common umask, then reset it back later.
	oldUmask := unix.Umask(0022)
	defer unix.Umask(oldUmask)

	// TODO: should also check the cache umask.
	for _, tc := range umaskTests {
		var cmdArgs []string
		if tc.force {
			cmdArgs = append(cmdArgs, "--force")
		}
		cmdArgs = append(cmdArgs, tc.imagePath, "library://alpine")

		c.env.RunSingularity(
			t,
			e2e.WithProfile(e2e.UserProfile),
			e2e.PreRun(func(t *testing.T) {
				// Reset the file permission after every pull.
				err := os.Chmod(tc.imagePath, 0666)
				if !os.IsNotExist(err) && err != nil {
					t.Fatalf("failed chmod-ing file: %s", err)
				}

				// Set the test umask.
				unix.Umask(tc.umask)
			}),
			e2e.PostRun(func(t *testing.T) {
				// Check the file permission.
				permOut := getFilePerm(t, tc.imagePath)
				if tc.expectPerm != permOut {
					t.Fatalf("Unexpected failure: expecting file perm: %o, got: %o", tc.expectPerm, permOut)
				}
			}),
			e2e.WithCommand("pull"),
			e2e.WithArgs(cmdArgs...),
			e2e.ExpectExit(0),
		)
	}
}

// E2ETests is the main func to trigger the test suite
func E2ETests(env e2e.TestEnv) testhelper.Tests {
	c := ctx{
		env: env,
	}

	// FIX: should run in parallel but the use of Chdir conflicts
	// with other tests and can lead to test failures
	return testhelper.Tests{
		"ordered": testhelper.NoParallel(func(t *testing.T) {
			// Run the tests the do not require setup.
			t.Run("pullUmaskCheck", c.testPullUmask)

			// Setup a test registry to pull from (for oras).
			c.setup(t)

			t.Run("pull", c.testPullCmd)
			t.Run("pullDisableCache", c.testPullDisableCacheCmd)

			// Regressions
			t.Run("issue5808", c.issue5808)
		}),
	}
}
