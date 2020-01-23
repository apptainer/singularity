// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import "github.com/sylabs/singularity/internal/pkg/plugin"

// DisablePlugin disables the named plugin.
func DisablePlugin(name, libexecdir string) error {
	return plugin.Disable(name)
}
