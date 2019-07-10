// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sifbundle

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/sylabs/singularity/pkg/ocibundle/tools"

	"github.com/opencontainers/runtime-tools/generate"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/test"
	testCache "github.com/sylabs/singularity/internal/pkg/test/tool/cache"
)

func TestFromSif(t *testing.T) {
	test.EnsurePrivilege(t)

	// prepare bundle directory and download a SIF image
	bundlePath, err := ioutil.TempDir("", "bundle")
	if err != nil {
		t.Fatal(err)
	}
	f, err := ioutil.TempFile("", "busybox")
	if err != nil {
		t.Fatal(err)
	}
	sifFile := f.Name()
	f.Close()
	defer os.Remove(sifFile)

	sing, err := exec.LookPath("singularity")
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"build", "-F", sifFile, "docker://busybox"}

	// create a clean image cache
	imgCacheDir := testCache.MakeDir(t, "")
	defer testCache.DeleteDir(t, imgCacheDir)

	// build SIF image
	cmd := exec.Command(sing, args...)
	cmd.Env = append(os.Environ(), cache.DirEnv+"="+imgCacheDir)
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	// test with a wrong image path
	bundle, err := FromSif("/blah", bundlePath, false)
	if err != nil {
		t.Errorf("unexpected success while opening non existent image")
	}
	// create OCI bundle from SIF
	if err := bundle.Create(nil); err == nil {
		// check if cleanup occurred
		t.Errorf("unexpected success while creating OCI bundle")
	}

	// create OCI bundle from SIF
	bundle, err = FromSif(sifFile, bundlePath, true)
	if err != nil {
		t.Fatal(err)
	}
	// generate a default configuration
	g, err := generate.New(runtime.GOOS)
	if err != nil {
		t.Fatal(err)
	}
	// remove seccomp filter for CI
	g.Config.Linux.Seccomp = nil
	g.SetProcessArgs([]string{tools.RunScript, "id"})

	if err := bundle.Create(g.Config); err != nil {
		// check if cleanup occurred
		t.Fatal(err)
	}

	// execute oci run command
	args = []string{"oci", "run", "-b", bundlePath, filepath.Base(sifFile)}
	cmd = exec.Command(sing, args...)
	if err := cmd.Run(); err != nil {
		t.Error(err)
	}

	if err := bundle.Delete(); err != nil {
		t.Error(err)
	}
}
