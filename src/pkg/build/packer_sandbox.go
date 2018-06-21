// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// Copyright (c) 2018, Vanessa Sochat. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"os/exec"

	"github.com/singularityware/singularity/src/pkg/sylog"
)

// SandboxPacker holds the source to copy from and the
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
	cmd := exec.Command("cp", "-r", rootfs+`/.`, b.Rootfs())
	err = cmd.Run()
	if err != nil {
		sylog.Errorf("cp Failed", err.Error())
		return nil, err
	}

	return
}
