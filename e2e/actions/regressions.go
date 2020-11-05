// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package actions

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/internal/pkg/cache"
	"github.com/sylabs/singularity/internal/pkg/test/tool/require"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
)

// Check there is no file descriptor leaked in the container
// process. This test expect 4 file descriptors, 3 for stdin,
// stdout, stderr and one opened by the ls command.
func (c actionTests) issue4488(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("exec"),
		e2e.WithArgs(c.env.ImagePath, "ls", "-1", "/proc/self/fd"),
		e2e.ExpectExit(
			0,
			e2e.ExpectOutput(e2e.ExactMatch, "0\n1\n2\n3"),
		),
	)
}

// Check that current working directory when is the user
// home directory doesn't override the custom home directory.
func (c actionTests) issue4587(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	u := e2e.UserProfile.HostUser(t)

	homeDir, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "homedir-", "")
	defer cleanup(t)

	canaryFile := filepath.Join(homeDir, "canary_file")
	if err := fs.Touch(canaryFile); err != nil {
		t.Fatalf("failed to create canary file: %s", err)
	}

	homeBind := homeDir + ":" + u.Dir

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithDir(u.Dir),
		e2e.WithCommand("exec"),
		e2e.WithArgs("--home", homeBind, c.env.ImagePath, "test", "-f", filepath.Join(u.Dir, "canary_file")),
		e2e.ExpectExit(0),
	)
}

// Check that current working directory doesn't interfere
// with image content when using underlay.
func (c actionTests) issue4755(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	sandbox, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "sandbox-", "")
	defer cleanup(t)

	// convert test image to sandbox
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("build"),
		e2e.WithArgs("--force", "--sandbox", sandbox, c.env.ImagePath),
		e2e.ExpectExit(0),
	)

	// create a file in image /tmp in order to trigger the issue
	// with underlay layer
	baseDir := filepath.Join(sandbox, filepath.Dir(c.env.TestDir))
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		t.Fatalf("can't create image directory %s: %s", baseDir, err)
	}
	path := filepath.Join(baseDir, "underlay-test")
	if err := fs.Touch(path); err != nil {
		t.Fatalf("can't create %s: %s", path, err)
	}

	// use of user namespace to force runtime to use underlay
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserNamespaceProfile),
		e2e.WithDir(c.env.TestDir),
		e2e.WithCommand("exec"),
		e2e.WithArgs(sandbox, "true"),
		e2e.ExpectExit(0),
	)
}

// Check that the last element of current working directory when it's
// a symlink pointing to a relative target is correctly handled by the
// runtime.
func (c actionTests) issue4768(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	homeDir, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "issue-4768-", "")
	defer cleanup(t)

	symCwdPath := filepath.Join(homeDir, "symlink")
	if err := os.Symlink(".", symCwdPath); err != nil {
		t.Fatalf("failed to create symlink %s: %s", symCwdPath, err)
	}

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithDir(symCwdPath),
		e2e.WithCommand("exec"),
		e2e.WithArgs(c.env.ImagePath, "pwd"),
		e2e.ExpectExit(
			0,
			e2e.ExpectOutput(e2e.ExactMatch, homeDir),
		),
	)
}

// Check that underlay layer handle relative/absolute symlinks
// when those are bind mount points.
func (c actionTests) issue4797(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	// /etc/relative-slink in the image point to ../usr/share/zoneinfo/Etc/UTC
	// /etc/absolute-slink in the image point to /usr/share/zoneinfo/Etc/UTC
	tests := []struct {
		name string
		args []string
		exit int
	}{
		{
			// check /usr/bin presence in the container
			name: "RelativeUsrBin",
			args: []string{"--bind", "/etc/passwd:/etc/relative-slink", c.env.ImagePath, "test", "-d", "/usr/bin"},
			exit: 0,
		},
		{
			// check /usr/share/zoneinfo/Etc/UTC presence in the container
			name: "RelativeUTC",
			args: []string{"--bind", "/etc/passwd:/etc/relative-slink", c.env.ImagePath, "test", "-f", "/usr/share/zoneinfo/Etc/UTC"},
			exit: 0,
		},
		{
			name: "AbsoluteUsrBin",
			args: []string{"--bind", "/etc/passwd:/etc/absolute-slink", c.env.ImagePath, "test", "-d", "/usr/bin"},
			exit: 0,
		},
		{
			name: "AbsoluteUTC",
			args: []string{"--bind", "/etc/passwd:/etc/absolute-slink", c.env.ImagePath, "test", "-f", "/usr/share/zoneinfo/Etc/UTC"},
			exit: 0,
		},
	}

	for _, tt := range tests {
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserNamespaceProfile),
			e2e.WithCommand("exec"),
			e2e.WithArgs(tt.args...),
			e2e.ExpectExit(tt.exit),
		)
	}
}

// Check that current working directory is correctly handled when an
// element of the path is a symlink.
func (c actionTests) issue4836(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	// $TMPDIR/issue-4836-XXXX directory
	issueDir, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "issue-4836-", "")
	defer cleanup(t)

	// $TMPDIR/issue-4836-XXXX/dir/child directory
	dir := filepath.Join(issueDir, "dir", "child")
	if err := os.MkdirAll(filepath.Join(issueDir, "dir", "child"), 0755); err != nil {
		t.Fatalf("failed to create dir %s: %s", dir, err)
	}

	// $TMPDIR/issue-4836-XXXX/symlink -> $TMPDIR/issue-4836-XXXX/dir
	symlink := filepath.Join(issueDir, "symlink")
	if err := os.Symlink(filepath.Join(issueDir, "dir"), symlink); err != nil {
		t.Fatalf("failed to create symlink %s: %s", symlink, err)
	}

	// will trigger the issue by traversing symlinked path into
	// the child directory :
	// PWD = $TMPDIR/issue-4836-XXXX/symlink/child
	cwd := filepath.Join(symlink, "child")

	// chdir will resolve the path so we check against dir, we could also
	// check $PWD content but that's enough
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("exec"),
		e2e.WithDir(cwd),
		e2e.WithArgs(c.env.ImagePath, "pwd"),
		e2e.ExpectExit(
			0,
			e2e.ExpectOutput(e2e.ExactMatch, dir),
		),
	)
}

// Check that image caching from an http source works correctly, using
// the HTTP Last-Modified header to invalidate previously pulled images,
// and that if we exec c1/v1.sif and then c1/v2.sif the latter is *not*
// run from the cached image of the former.
func (c actionTests) issue4823(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	// Copy image to a local tempdir so we can modify times on it
	issueDir, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "issue-4823-", "")
	defer cleanup(t)
	issueImage := path.Join(issueDir, "test.sif")
	if err := fs.CopyFile(c.env.ImagePath, issueImage, 0755); err != nil {
		t.Fatalf("Could not copy test image file: %v", err)
	}

	// Start an http server that always serves the same file from
	// whatever URL is requested
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, issueImage)
	}))
	defer srv.Close()

	tests := []struct {
		name         string
		urlPath      string
		touchFile    bool
		expectCached bool
	}{
		// First time we use an image at a URL it is not cached
		{
			name:         "http_c1v1.sif_uncached",
			urlPath:      "/c1/v1.sif",
			touchFile:    false,
			expectCached: false,
		},
		// Second time at same URL is cached
		{
			name:         "http_c1v1.sif_cached",
			urlPath:      "/c1/v1.sif",
			touchFile:    false,
			expectCached: true,
		},
		// Date of file at the URL is modified - not cached
		{
			name:         "http_c1v1.sif_touched_uncached",
			urlPath:      "/c1/v1.sif",
			touchFile:    true,
			expectCached: false,
		},
		// Different URL - not cached
		{
			name:         "http_c2v1.sif_uncached",
			urlPath:      "/c2/v1.sif",
			touchFile:    false,
			expectCached: false,
		},
	}

	// Share a cache dir for all of the subtests
	cacheDir, cleanup := e2e.MakeCacheDir(t, "")
	defer cleanup(t)
	_, err := cache.New(cache.Config{ParentDir: cacheDir})
	if err != nil {
		t.Fatalf("Could not create image cache handle: %v", err)
	}

	for _, tt := range tests {
		if tt.touchFile {
			// touch it into the future by a minute, in case tests
			// are running on fs with poor timestamp resolution
			newTime := time.Now().Add(time.Minute)
			err := os.Chtimes(issueImage, newTime, newTime)
			if err != nil {
				t.Fatalf("Error setting test file times: %v", err)
			}
		}

		expected := e2e.ExpectError(e2e.ContainMatch, "Downloading network image")
		if tt.expectCached {
			expected = e2e.ExpectError(e2e.ContainMatch, "Using image from cache")
		}

		c.env.ImgCacheDir = cacheDir
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithGlobalOptions("--debug"),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("exec"),
			e2e.WithArgs(srv.URL+tt.urlPath, "/bin/true"),
			e2e.ExpectExit(
				0,
				expected,
			),
		)
	}
}

// Check that we can run a container when the home mount is '/' as it is for 'nobody'
// We should just not do that mount which would clobber the whole container fs
func (c actionTests) issue5228(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	u := e2e.UserProfile.HostUser(t)

	// We don't actually switch user to one with `/` - we put this mount in using `--home`
	// which has the same effect.
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithDir(u.Dir),
		e2e.WithCommand("exec"),
		e2e.WithArgs("--home", "/", c.env.ImagePath, "/bin/true"),
		e2e.ExpectExit(0),
	)
}

// Check that home directory is not mounted under `/root` when --fakeroot and
// --contain are both given.
func (c actionTests) issue5211(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	u := e2e.UserProfile.HostUser(t)

	// Make non-hidden file in host home to check if it was mounted in container
	canaryDir, cleanup := e2e.MakeTempDir(t, u.Dir, "singularity-issue5211-dir-", "")
	defer cleanup(t)

	canaryBasename := filepath.Base(canaryDir)

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.FakerootProfile),
		e2e.WithDir(u.Dir),
		e2e.WithCommand("exec"),
		e2e.WithArgs("--contain", c.env.ImagePath, "test", "!", "-d", filepath.Join("/root", canaryBasename)),
		e2e.ExpectExit(0),
	)

	// Check we preserve `$HOME` as /root even when we `--contain` with `--fakeroot`
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.FakerootProfile),
		e2e.WithDir(u.Dir),
		e2e.WithCommand("exec"),
		e2e.WithArgs("--contain", c.env.ImagePath, "sh", "-c", "echo $HOME"),
		e2e.ExpectExit(
			0,
			e2e.ExpectOutput(e2e.ExactMatch, "/root"),
		),
	)

}

// Check that we can create a directory in container image with --writable-tmpfs.
func (c actionTests) issue5271(t *testing.T) {
	require.Filesystem(t, "overlay")

	e2e.EnsureImage(t, c.env)

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("exec"),
		e2e.WithArgs("--writable-tmpfs", c.env.ImagePath, "mkdir", "/e2e-dir"),
		e2e.ExpectExit(0),
	)
}

// Check that we get a warning when using --writable-tmpfs with underlay.
func (c actionTests) issue5307(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserNamespaceProfile),
		e2e.WithCommand("exec"),
		e2e.WithArgs("--writable-tmpfs", c.env.ImagePath, "true"),
		e2e.ExpectExit(
			0,
			e2e.ExpectError(e2e.ContainMatch, "Disabling --writable-tmpfs"),
		),
	)
}

// Check we can fakeroot exec an image containing a system xattr, which we may
// not be able to set in the SIF -> sandbox extraction.
func (c actionTests) issue5399(t *testing.T) {

	dir, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "issue5399-", "")
	defer e2e.Privileged(cleanup)(t)
	image := filepath.Join(dir, "issue_5399.sif")

	// Build as root to guarantee no issue setting the system xattr
	// Certain config may not allow us to do it as fakeroot e.g. it failed
	// in Ubuntu1604 CI.
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("build"),
		e2e.WithArgs(image, "testdata/regressions/issue_5399.def"),
		e2e.ExpectExit(0),
	)

	// Fakeroot will extract to a sandbox using mksquashfs as the user.
	// Should succeed, though it can't set a system xattr.
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.FakerootProfile),
		e2e.WithCommand("exec"),
		e2e.WithArgs(image, "/bin/true"),
		e2e.ExpectExit(0),
	)
}

// Check that we can create a directory in a root owned directory
// with others write permissions in conjunction with --writable-tmpfs.
func (c actionTests) issue5455(t *testing.T) {
	require.Filesystem(t, "overlay")

	e2e.EnsureImage(t, c.env)

	dir, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "issue5455-", "")
	defer e2e.Privileged(cleanup)(t)

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("build"),
		e2e.WithArgs("--force", "--sandbox", dir, c.env.ImagePath),
		e2e.PostRun(func(t *testing.T) {
			if t.Failed() {
				return
			}
			permDir := filepath.Join(dir, "perm")
			if err := os.Mkdir(permDir, 0777); err != nil {
				t.Errorf("while creating %s: %s", permDir, err)
			}
			if err := os.Chmod(permDir, 0777); err != nil {
				t.Errorf("while setting permission on %s: %s", permDir, err)
			}
		}),
		e2e.ExpectExit(0),
	)

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("exec"),
		e2e.WithArgs("--writable-tmpfs", dir, "mkdir", "/perm/issue5455"),
		e2e.ExpectExit(0),
	)
}

// Check that we can run a container with no fuse mounts when they are disabled
// by config enable fusemount=no
func (c actionTests) issue5631(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	// Set enable fusemount = no in a custom config file
	tmpDir, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "issue-5631-", "")
	defer e2e.Privileged(cleanup)(t)
	tmpConfig := path.Join(tmpDir, "singularity.conf")
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.PreRun(
			// Custom config file must exist and be root owned with tight permissions
			func(t *testing.T) {
				err := fs.EnsureFileWithPermission(tmpConfig, 0600)
				if err != nil {
					t.Fatalf("while creating temporary config file: %s", err)
				}
			}),
		e2e.WithCommand("config global"),
		e2e.WithGlobalOptions("--config", tmpConfig),
		e2e.WithArgs("--set", "enable fusemount", "no"),
		e2e.ExpectExit(0),
	)

	// Check we can run a bare container still against the custom config
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("exec"),
		e2e.WithGlobalOptions("--config", tmpConfig),
		e2e.WithArgs(c.env.ImagePath, "/bin/true"),
		e2e.ExpectExit(0),
	)
}

// Check that mount failure for /etc/hosts and /etc/localtime are not fatal
// Separate code paths for contain and non-contained, so check both
func (c actionTests) issue5465(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	sandbox, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "sandbox-", "")
	defer cleanup(t)

	// convert test image to sandbox
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("build"),
		e2e.WithArgs("--force", "--sandbox", sandbox, c.env.ImagePath),
		e2e.ExpectExit(0),
	)

	// Link /etc/localtime and /etc/hosts to a directory
	// (bind file onto dir will fail)
	if err := os.Mkdir(filepath.Join(sandbox, "dir"), 0o755); err != nil {
		t.Fatalf("couldn't create dir in sandbox: %s", err)
	}
	if err := os.Remove(filepath.Join(sandbox, "etc", "localtime")); err != nil && !os.IsNotExist(err) {
		t.Fatalf("couldn't remove sandbox localtime: %s", err)
	}
	if err := os.Remove(filepath.Join(sandbox, "etc", "hosts")); err != nil && !os.IsNotExist(err) {
		t.Fatalf("couldn't remove sandbox hosts: %s", err)
	}
	if err := os.Symlink("/dir", filepath.Join(sandbox, "etc", "localtime")); err != nil {
		t.Fatalf("couldn't symlink sandbox localtime: %s", err)
	}
	if err := os.Symlink("/dir", filepath.Join(sandbox, "etc", "hosts")); err != nil {
		t.Fatalf("couldn't symlink sandbox hosts: %s", err)
	}

	// The standard flow where the binds come from singularity.conf
	c.env.RunSingularity(
		t,
		e2e.AsSubtest("Standard"),
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithDir(c.env.TestDir),
		e2e.WithCommand("exec"),
		e2e.WithArgs(sandbox, "true"),
		e2e.ExpectExit(0),
	)

	// With `--contain` where the binds are hard coded
	c.env.RunSingularity(
		t,
		e2e.AsSubtest("Contain"),
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithDir(c.env.TestDir),
		e2e.WithCommand("exec"),
		e2e.WithArgs("--contain", sandbox, "true"),
		e2e.ExpectExit(0),
	)
}

// Check that flag / env var binds are passed in $SINGULARITY_BIND in the
// container. Sometimes used by containers that require data to be bound in to a
// location etc., and was present in older versions of Singularity.
func (c actionTests) issue5599(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	tmpDir, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "issue-5599-", "")
	defer e2e.Privileged(cleanup)(t)
	// Binds from env var and flag are additive
	envBind := tmpDir + ":/srv"
	bindEnv := "SINGULARITY_BIND=" + envBind
	flagBind := tmpDir + ":/mnt"
	expectedEnv := fmt.Sprintf("SINGULARITY_BIND=%s,%s", flagBind, envBind)
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("exec"),
		e2e.WithEnv(append(os.Environ(), bindEnv)),
		e2e.WithArgs("--bind", flagBind, c.env.ImagePath, "/usr/bin/env"),
		e2e.ExpectExit(0,
			e2e.ExpectOutput(e2e.ContainMatch, expectedEnv),
		),
	)
}

// Check that unsquashfs (for version >= 4.4) works for non root users when image contains
// pseudo devices in /dev.
func (c actionTests) issue5690(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("exec"),
		e2e.WithArgs(c.env.ImagePath, "/bin/true"),
		e2e.ExpectExit(0),
	)

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.FakerootProfile),
		e2e.WithCommand("exec"),
		e2e.WithArgs(c.env.ImagePath, "/bin/true"),
		e2e.ExpectExit(0),
	)
}
