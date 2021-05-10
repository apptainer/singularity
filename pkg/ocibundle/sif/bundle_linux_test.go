// Copyright (c) 2019-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sifbundle

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/hpcng/singularity/internal/pkg/runtime/engine/config/oci"
	"github.com/hpcng/singularity/internal/pkg/test"
	"github.com/hpcng/singularity/internal/pkg/util/fs"
	"github.com/hpcng/singularity/pkg/ocibundle/tools"
	"github.com/hpcng/singularity/pkg/util/fs/proc"
	"github.com/opencontainers/runtime-tools/validate"
)

// We need a busybox SIF for these tests. We used to download it each time, but we have one
// around for some e2e tests already.
const busyboxSIF = "../../../e2e/testdata/busybox.sif"

func TestFromSif(t *testing.T) {
	test.EnsurePrivilege(t)

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

	if err := fs.CopyFileAtomic(busyboxSIF, sifFile, 0755); err != nil {
		t.Fatalf("Could not copy test image: %v", err)
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

	tests := []struct {
		name     string
		writable bool
	}{
		{"FromSif", false},
		{"FromSifWritable", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.writable {
				requireFilesystem(t, "overlay")
			}

			// create OCI bundle from SIF
			bundle, err = FromSif(sifFile, bundlePath, tt.writable)
			if err != nil {
				t.Fatal(err)
			}
			// generate a default configuration
			g, err := oci.DefaultConfig()
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

			// Validate the bundle using OCI runtime-tools
			// Run in non-host-specific mode. Our bundle is for the "linux" platform
			v, err := validate.NewValidatorFromPath(bundlePath, false, "linux")
			if err != nil {
				t.Errorf("Could not create bundle validator: %v", err)
			}
			if err := v.CheckAll(); err != nil {
				t.Errorf("Bundle not valid: %v", err)
			}

			// Clean up
			if err := bundle.Delete(); err != nil {
				t.Error(err)
			}

		})
	}

}

// TODO: This is a duplicate from internal/pkg/test/tool/require
// in order avoid needing buildcfg for this unit test, such that
// it can be run directly from the source tree without compilation.
// This bundle code is in `pkg/` so *should not* depend on a compiled
// Singularity (https://github.com/hpcng/singularity/issues/2316).
//
// Ideally we would refactor i/p/t/t/require so requirements that
// don't need a compiled Singularity can be used without compiled
// Singularity.
//
// Filesystem checks that the current test could use the
// corresponding filesystem, if the filesystem is not
// listed in /proc/filesystems, the current test is skipped
// with a message.
func requireFilesystem(t *testing.T, fs string) {
	has, err := proc.HasFilesystem(fs)
	if err != nil {
		t.Fatalf("error while checking filesystem presence: %s", err)
	}
	if !has {
		t.Skipf("%s filesystem seems not supported", fs)
	}
}
