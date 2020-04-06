// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"github.com/sylabs/singularity/internal/pkg/plugin"
	"github.com/sylabs/singularity/pkg/sylog"
)

// CreatePlugin create the plugin directory skeleton.
func CreatePlugin(dir, name string) error {
	sylog.Debugf("Create %q plugin directory %s", name, dir)
	return plugin.Create(dir, name)
}
