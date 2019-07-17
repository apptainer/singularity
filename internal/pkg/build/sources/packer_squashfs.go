// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strconv"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/build/types"
	"github.com/sylabs/singularity/pkg/util/loop"
)

// SquashfsPacker holds the locations of where to pack from and to, aswell as image offset info
type SquashfsPacker struct {
	srcfile string
	b       *types.Bundle
	info    *loop.Info64
}

// Pack puts relevant objects in a Bundle!
func (p *SquashfsPacker) Pack() (*types.Bundle, error) {
	rootfs := p.srcfile

	err := p.unpackSquashfs(p.b, p.info, rootfs)
	if err != nil {
		sylog.Errorf("unpackSquashfs Failed: %s", err)
		return nil, err
	}

	return p.b, nil
}

// unpackSquashfs removes the image header with dd and then unpackes image into bundle directories with unsquashfs
func (p *SquashfsPacker) unpackSquashfs(b *types.Bundle, info *loop.Info64, rootfs string) (err error) {
	var stderr bytes.Buffer

	trimfile, err := ioutil.TempFile(p.b.Path, "trim.squashfs")
	if err != nil {
		return fmt.Errorf("while making tmp file: %v", err)
	}

	// trim header
	sylog.Debugf("Creating copy of %s without header at %s\n", rootfs, trimfile.Name())
	cmd := exec.Command("dd", "bs="+strconv.Itoa(int(info.Offset)), "skip=1", "if="+rootfs, "of="+trimfile.Name())
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("trimming header failed: %v: %v", err, stderr.String())
	}

	// copy filesystem into bundle rootfs
	sylog.Debugf("Unsquashing %s to %s in Bundle\n", trimfile.Name(), b.Rootfs())
	stderr.Reset()
	cmd = exec.Command("unsquashfs", "-f", "-d", b.Rootfs(), trimfile.Name())
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("unsquashfs Failed: %v: %v", err, stderr.String())
	}

	return err
}
