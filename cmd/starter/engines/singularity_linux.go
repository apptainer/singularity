// Copyright (c) 2019-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

//go:build singularity_engine
// +build singularity_engine

package engines

import (
	// register the singularity runtime engine
	_ "github.com/hpcng/singularity/internal/pkg/runtime/engine/singularity"
)
