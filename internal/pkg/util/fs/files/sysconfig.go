// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package files

import (
	"path/filepath"

	"github.com/sylabs/singularity/internal/pkg/buildcfg"
)

// GetSysConfigFile returns the path the Singularity system
// configuration file
func GetSysConfigFile() string {
	return filepath.Join(buildcfg.SYSCONFDIR, "singularity/singularity.conf")
}
