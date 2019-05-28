// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/sylabs/sif/pkg/siftool"
)

// SiftoolCmd is easily set since the sif repo allows the cobra.Command struct to be
// easily accessed with Siftool(), we do not need to do anything but call that function.
var SiftoolCmd = siftool.Siftool()

func init() {
	cmdManager.RegisterCmd(SiftoolCmd)
}
