// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"golang.org/x/sys/unix"
)

func SetupPluginDir(t *testing.T, testDir string) {
	Privileged(func(t *testing.T) {
		path := buildcfg.PLUGIN_ROOTDIR

		if err := os.Mkdir(path, 0755); err != nil && !os.IsExist(err) {
			t.Fatalf("while creating plugin directory %s: %s", path, err)
		}
		dir := filepath.Join(testDir, "plugin-install")
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatalf("while creating plugin temporary directory %s: %s", dir, err)
		}
		if err := unix.Mount(dir, path, "", unix.MS_BIND, ""); err != nil {
			t.Fatalf("while mounting %s to %s: %s", dir, path, err)
		}
	})(t)
}
