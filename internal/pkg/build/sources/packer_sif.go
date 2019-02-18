// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"github.com/sylabs/singularity/pkg/build/types"
)

// SIFPacker holds the locations of where to pack from and to
type SIFPacker struct {
	srcfile string
	b       *types.Bundle
}
