// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import (
	"os"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/pkg/runtime/engine/config"
	"golang.org/x/sys/unix"
)

func SetupDefaultConfig(t *testing.T, path string) {
	c, err := config.ParseFile("")
	if err != nil {
		t.Fatalf("while generating singularity configuration: %s", err)
	}

	Privileged(func(t *testing.T) {
		f, err := os.Create(path)
		if err != nil {
			t.Fatalf("while creating singularity configuration: %s", err)
		}

		if err := config.Generate(f, "", c); err != nil {
			t.Fatalf("while generating singularity configuration: %s", err)
		}

		f.Close()

		if err := unix.Mount(path, buildcfg.SINGULARITY_CONF_FILE, "", unix.MS_BIND, ""); err != nil {
			t.Fatalf("while mounting %s to %s: %s", path, buildcfg.SINGULARITY_CONF_FILE, err)
		}
	})(t)
}
