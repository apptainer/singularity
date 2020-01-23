// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/pkg/errors"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/util/user"
)

// rpmMacrosContent contains required content to
// place in $HOME/.rpmmacros file for yum bootstrap
// build
var rpmMacrosContent = `
%_var /var
%_dbpath %{_var}/lib/rpm
`

// SetupHomeDirectories creates temporary home directories for
// privileged and unprivileged users and bind mount those directories
// on top of real ones. It's possible because e2e tests are executed
// in a dedicated mount namespace.
func SetupHomeDirectories(t *testing.T) {
	var unprivUser, privUser *user.User

	sessionDir := buildcfg.SESSIONDIR
	unprivUser = CurrentUser(t)

	Privileged(func(t *testing.T) {
		// there is no cleanup here because everything done (tmpfs, mounts)
		// in our dedicated mount namespace will be automatically discarded
		// by the kernel once all test processes exit

		privUser = CurrentUser(t)

		// create the temporary filesystem
		if err := syscall.Mount("tmpfs", sessionDir, "tmpfs", 0, "mode=0777"); err != nil {
			t.Fatalf("failed to mount temporary filesystem")
		}

		// want the already resolved current working directory
		cwd, err := os.Readlink("/proc/self/cwd")
		err = errors.Wrap(err, "getting current working directory from /proc/self/cwd")
		if err != nil {
			t.Fatalf("could not readlink /proc/self/cwd: %+v", err)
		}
		unprivResolvedHome, err := filepath.EvalSymlinks(unprivUser.Dir)
		err = errors.Wrapf(err, "resolving home from %q", unprivUser.Dir)
		if err != nil {
			t.Fatalf("could not resolve home directory: %+v", err)
		}
		privResolvedHome, err := filepath.EvalSymlinks(privUser.Dir)
		err = errors.Wrapf(err, "resolving home from %q", privUser.Dir)
		if err != nil {
			t.Fatalf("could not resolve home directory: %+v", err)
		}

		// prepare user temporary homes
		unprivSessionHome := filepath.Join(sessionDir, unprivUser.Name)
		privSessionHome := filepath.Join(sessionDir, privUser.Name)

		oldUmask := syscall.Umask(0)
		defer syscall.Umask(oldUmask)

		if err := os.Mkdir(unprivSessionHome, 0700); err != nil {
			err = errors.Wrapf(err, "creating temporary home directory at %s", unprivSessionHome)
			t.Fatalf("failed to create temporary home: %+v", err)
		}
		if err := os.Chown(unprivSessionHome, int(unprivUser.UID), int(unprivUser.GID)); err != nil {
			err = errors.Wrapf(err, "changing temporary home directory ownership at %s", unprivSessionHome)
			t.Fatalf("failed to set temporary home owner: %+v", err)
		}
		if err := os.Mkdir(privSessionHome, 0700); err != nil {
			err = errors.Wrapf(err, "changing temporary home directory %s", privSessionHome)
			t.Fatalf("failed to create temporary home: %+v", err)
		}

		sourceDir := buildcfg.SOURCEDIR

		// re-create the current source directory if it's located in the user
		// home directory and bind it. Root home directory is not checked because
		// the whole test suite can not run from there as we are dropping privileges
		if strings.HasPrefix(sourceDir, unprivResolvedHome) {
			trimmedSourceDir := strings.TrimPrefix(sourceDir, unprivResolvedHome)
			sessionSourceDir := filepath.Join(unprivSessionHome, trimmedSourceDir)
			if err := os.MkdirAll(sessionSourceDir, 0755); err != nil {
				err = errors.Wrapf(err, "creating temporary source directory at %q", sessionSourceDir)
				t.Fatalf("failed to create temporary home source directory: %+v", err)
			}
			if err := syscall.Mount(sourceDir, sessionSourceDir, "", syscall.MS_BIND, ""); err != nil {
				err = errors.Wrapf(err, "bind mounting source directory from %q to %q", sourceDir, sessionSourceDir)
				t.Fatalf("failed to bind mount source directory: %+v", err)
			}
		}

		// finally bind temporary homes on top of real ones
		// in order to not screw them by accident during e2e
		// tests execution
		if err := syscall.Mount(unprivSessionHome, unprivResolvedHome, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
			err = errors.Wrapf(err, "bind mounting source directory from %q to %q", unprivSessionHome, unprivResolvedHome)
			t.Fatalf("failed to bind mount home directory: %+v", err)
		}
		if err := syscall.Mount(privSessionHome, privResolvedHome, "", syscall.MS_BIND, ""); err != nil {
			err = errors.Wrapf(err, "bind mounting source directory from %q to %q", privSessionHome, privResolvedHome)
			t.Fatalf("failed to bind mount home directory: %+v", err)
		}
		// change to the "new" working directory if above mount override
		// the current working directory
		if err := os.Chdir(cwd); err != nil {
			err = errors.Wrapf(err, "change working directory to %s", cwd)
			t.Fatalf("failed to change working directory: %+v", err)
		}

		// create .rpmmacros files for yum bootstrap builds
		macrosFile := filepath.Join(unprivSessionHome, ".rpmmacros")
		if err := ioutil.WriteFile(macrosFile, []byte(rpmMacrosContent), 0444); err != nil {
			err = errors.Wrapf(err, "writing macros file at %s", macrosFile)
			t.Fatalf("could not write macros file: %+v", err)
		}
		macrosFile = filepath.Join(privSessionHome, ".rpmmacros")
		if err := ioutil.WriteFile(macrosFile, []byte(rpmMacrosContent), 0444); err != nil {
			err = errors.Wrapf(err, "writing macros file at %s", macrosFile)
			t.Fatalf("could not write macros file: %+v", err)
		}
	})(t)
}

// shadowInstanceDirectory creates a temporary instances directory which
// will be bound on top of current user home directory in order to execute
// a "shadow" instance (eg: docker registry).
func shadowInstanceDirectory(t *testing.T, env TestEnv) func(t *testing.T) {
	u := CurrentUser(t)

	// $TESTDIR/.singularity directory
	fakeSingularityDir := filepath.Join(env.TestDir, ".singularity")
	// $TESTDIR/.singularity/instances symlink
	fakeInstanceSymlink := filepath.Join(fakeSingularityDir, "instances")

	// create directory $TESTDIR/.singularity
	if err := os.Mkdir(fakeSingularityDir, 0755); err != nil && !os.IsExist(err) {
		err = errors.Wrapf(err, "create temporary singularity data directory at %q", fakeSingularityDir)
		t.Fatalf("failed to create fake singularity directory: %+v", err)
	}
	// mount $TESTDIR on top of $HOME
	if err := syscall.Mount(env.TestDir, u.Dir, "", syscall.MS_BIND, ""); err != nil {
		err = errors.Wrapf(err, "mounting temporary singularity data directory from %q to %q", env.TestDir, u.Dir)
		t.Fatalf("failed to mount directory: %+v", err)
	}
	// create symlink $HOME/.singularity/instances -> $TESTDIR/.singularity
	if err := os.Symlink(fakeSingularityDir, fakeInstanceSymlink); err != nil && !os.IsExist(err) {
		err = errors.Wrapf(err, "symlink temporary singularity data directory from %q to %q", fakeSingularityDir, fakeInstanceSymlink)
		t.Fatalf("failed to create symlink: %+v", err)
	}

	return func(t *testing.T) {
		if err := syscall.Unmount(u.Dir, syscall.MNT_DETACH); err != nil {
			err = errors.Wrapf(err, "unmount directory %q", u.Dir)
			t.Fatalf("failed to unmount directory: %+v", err)
		}
	}
}
