// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package squashfs

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config"
	singularityConfig "github.com/sylabs/singularity/pkg/runtime/engines/singularity/config"
)

// GetPath figures out where the mksquashfs binary is
// and return an error is not available or not usable.
func GetPath() (string, error) {
	// Parse singularity configuration file
	c := &singularityConfig.FileConfig{}
	if err := config.Parser(buildcfg.SYSCONFDIR+"/singularity/singularity.conf", c); err != nil {
		return "", fmt.Errorf("Unable to parse singularity.conf file: %s", err)
	}

	// p is either "" or the string value in the conf file
	p := c.MksquashfsPath

	// If the path contains the binary name use it as is, otherwise add mksquashfs via filepath.Join
	if !strings.HasSuffix(c.MksquashfsPath, "mksquashfs") {
		p = filepath.Join(c.MksquashfsPath, "mksquashfs")
	}

	// exec.LookPath functions on absolute paths (ignoring $PATH) as well
	return exec.LookPath(p)
}
