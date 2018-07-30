// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"os/exec"

	"github.com/singularityware/singularity/src/pkg/sylog"
)

// SandboxPacker holds the locations of where to pack from and to
// Ext3Packer holds the locations of where to back from and to, aswell as image offset info
type SandboxPacker struct {
	srcdir string
	tmpfs  string
}

// Pack puts relevant objects in a Bundle!
func (p *SandboxPacker) Pack() (b *Bundle, err error) {
	rootfs := p.srcdir

	b, err = NewBundle(p.tmpfs)
	if err != nil {
		return
	}

	//copy filesystem into bundle rootfs
	sylog.Debugf("Copying file system from %s to %s in Bundle\n", rootfs, b.Rootfs())
	cmd := exec.Command("cp", "-r", rootfs+`/.`, b.Rootfs())
	err = cmd.Run()
	if err != nil {
		sylog.Errorf("cp Failed", err.Error())
		return nil, err
	}

	return b, nil
}
