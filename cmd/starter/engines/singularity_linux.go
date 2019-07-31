// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build singularity_engine

package engines

import (
	// register the singularity runtime engine
	_ "github.com/sylabs/singularity/internal/pkg/runtime/engines/singularity"
)
