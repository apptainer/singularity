// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"github.com/sylabs/singularity/pkg/build/types"
	"github.com/sylabs/singularity/pkg/util/loop"
)

// Ext3Packer holds the locations of where to back from and to, aswell as image offset info
type Ext3Packer struct {
	srcfile string
	b       *types.Bundle
	info    *loop.Info64
}
