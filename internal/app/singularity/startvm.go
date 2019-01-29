// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

func prepareVM(cmd *cobra.Command, image string, args []string) {
	// SIF image we are running
	sifImage := image

	cliExtra := ""
	singAction := cmd.Name()

	imgPath := strings.Split(sifImage, ":")
	isInternal := false
	if strings.HasPrefix("internal", filepath.Base(imgPath[0])) {
		cliExtra = "syos"
		isInternal = true
	} else {
		// Get our "action" (run, exec, shell) based on the action script being called
		singAction = filepath.Base(args[0])
		cliExtra = strings.Join(args[1:], " ")
	}

	if err := startVM(sifImage, singAction, cliExtra, isInternal); err != nil {
		sylog.Errorf("VM instance failed: %s", err)
		os.Exit(2)
	}
}
