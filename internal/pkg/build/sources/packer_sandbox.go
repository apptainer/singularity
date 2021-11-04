// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"context"
	"fmt"

	"github.com/hpcng/singularity/pkg/build/types"
	"github.com/hpcng/singularity/pkg/sylog"
	"github.com/hpcng/singularity/pkg/util/archive"
)

// SandboxPacker holds the locations of where to pack from and to
// Ext3Packer holds the locations of where to back from and to, as well as image offset info
type SandboxPacker struct {
	srcdir string
	b      *types.Bundle
}

// Pack puts relevant objects in a Bundle!
func (p *SandboxPacker) Pack(context.Context) (*types.Bundle, error) {
	rootfs := p.srcdir

	// copy filesystem into bundle rootfs
	sylog.Debugf("Copying file system from %s to %s in Bundle\n", rootfs, p.b.RootfsPath)

	err := archive.CopyWithTar(rootfs+`/.`, p.b.RootfsPath)
	if err != nil {
		return nil, fmt.Errorf("copy Failed: %v", err)
	}

	return p.b, nil
}
