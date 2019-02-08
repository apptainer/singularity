// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package tools

import (
	"fmt"
	"os"

	"github.com/sylabs/singularity/pkg/util/loop"
)

// CreateLoop associates a file to loop device and returns
// path of loop device used
func CreateLoop(file *os.File, offset, size uint64) (string, error) {
	loopDev := &loop.Device{
		MaxLoopDevices: 256,
		Shared:         true,
		Info: &loop.Info64{
			SizeLimit: size,
			Offset:    offset,
			Flags:     loop.FlagsAutoClear,
		},
	}
	idx := 0
	if err := loopDev.AttachFromFile(file, os.O_RDONLY, &idx); err != nil {
		return "", fmt.Errorf("failed to attach image %s: %s", file.Name(), err)
	}
	return fmt.Sprintf("/dev/loop%d", idx), nil
}
