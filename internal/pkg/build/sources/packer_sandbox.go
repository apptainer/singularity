// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"os/exec"

	"github.com/sylabs/singularity/internal/pkg/build/types"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

// SandboxPacker holds the locations of where to pack from and to
// Ext3Packer holds the locations of where to back from and to, aswell as image offset info
type SandboxPacker struct {
	srcdir string
	b      *types.Bundle
}

// Pack puts relevant objects in a Bundle!
func (p *SandboxPacker) Pack() (*types.Bundle, error) {
	rootfs := p.srcdir

	//copy filesystem into bundle rootfs
	sylog.Debugf("Copying file system from %s to %s in Bundle\n", rootfs, p.b.Rootfs())
	cmd := exec.Command("cp", "-r", rootfs+`/.`, p.b.Rootfs())
	err := cmd.Run()
	if err != nil {
		sylog.Errorf("cp Failed: %s", err)
		return nil, err
	}

	return p.b, nil
}
