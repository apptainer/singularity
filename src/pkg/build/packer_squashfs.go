// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// Copyright (c) 2018, Vanessa Sochat. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"io/ioutil"
	"os/exec"
	"strconv"

	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/loop"
)

// SquashfsPacker holds the locations of where to pack from and to, aswell as image offset info
type SquashfsPacker struct {
	srcfile string
	tmpfs   string
	info    *loop.Info64
}

// Pack puts relevant objects in a Bundle!
func (p *SquashfsPacker) Pack() (b *Bundle, err error) {
	rootfs := p.srcfile

	b, err = NewBundle(p.tmpfs)
	if err != nil {
		return
	}
	err = p.unpackSquashfs(b, p.info, rootfs)
	if err != nil {
		sylog.Errorf("unpackSquashfs Failed", err.Error())
		return nil, err
	}

	return b, nil
}

// unpackSquashfs removes the image header with dd and then unpackes image into bundle directories with unsquashfs
func (p *SquashfsPacker) unpackSquashfs(b *Bundle, info *loop.Info64, rootfs string) (err error) {
	trimfile, err := ioutil.TempFile(p.tmpfs, "trim.squashfs")

	//trim header
	sylog.Debugf("Creating copy of %s without header at %s\n", rootfs, trimfile.Name())
	cmd := exec.Command("dd", "bs="+strconv.Itoa(int(info.Offset)), "skip=1", "if="+rootfs, "of="+trimfile.Name())
	err = cmd.Run()
	if err != nil {
		sylog.Errorf("Trimming header Failed", err.Error())
		return err
	}

	//copy filesystem into bundle rootfs
	sylog.Debugf("Unsquashing %s to %s in Bundle\n", trimfile.Name(), b.Rootfs())
	cmd = exec.Command("unsquashfs", "-f", "-d", b.Rootfs(), trimfile.Name())
	err = cmd.Run()
	if err != nil {
		sylog.Errorf("unsquashfs Failed", err.Error())
		return err
	}

	return err
}
