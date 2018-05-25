// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"os"
	"os/exec"

	"github.com/singularityware/singularity/src/pkg/sylog"

	"github.com/singularityware/singularity/src/pkg/buildcfg"
	"github.com/spf13/cobra"
)

func init() {
	sifCmd.Flags().SetInterspersed(false)
	singularityCmd.AddCommand(sifCmd)

}

var sif = buildcfg.SBINDIR + "/sif"

var sifCmd = &cobra.Command{
	Use:    "sif COMMAND OPTION FILE",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {

		sifCmd := exec.Command(sif, args...)
		sifCmd.Stdout = os.Stdout
		sifCmd.Stderr = os.Stderr

		if err := sifCmd.Start(); err != nil {
			sylog.Errorf("%v", err)
		}
		if err := sifCmd.Wait(); err != nil {
			sylog.Errorf("%v", err)
		}
	},
}
