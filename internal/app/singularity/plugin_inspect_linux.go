// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"

	"github.com/sylabs/singularity/internal/pkg/plugin"
)

// InspectPlugin inspects the named plugin.
func InspectPlugin(name string) error {
	manifest, err := plugin.Inspect(name)
	if err != nil {
		return err
	}

	fmt.Printf("Name: %s\n"+
		"Description: %s\n"+
		"Author: %s\n"+
		"Version: %s\n",
		manifest.Name,
		manifest.Description,
		manifest.Author,
		manifest.Version)

	return nil
}
