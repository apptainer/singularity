// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cleanCache

import (
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
)

func CleanSingularityCache() error {

	err := cache.Clean()
	if err != nil {
		return err
	}

	sylog.Debugf("DONE!")

	return err
}


