// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"os"
	"os/exec"

	"github.com/singularityware/singularity/src/pkg/buildcfg"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/spf13/cobra"
)

func init() {
	sifCmd.Flags().SetInterspersed(false)
	singularityCmd.AddCommand(sifCmd)
}

var sifExample = `
sif create -P /tmp/fs.squash -f "SQUASHFS" -p "SYSTEM" -c "Linux" /tmp/container.sif`

var sifCmd = &cobra.Command{
	Use: "sif COMMAND OPTION FILE",
	Run: func(cmd *cobra.Command, args []string) {
		var sif = buildcfg.SBINDIR + "/sif"

		sifCmd := exec.Command(sif, args...)
		sifCmd.Stdout = os.Stdout
		sifCmd.Stderr = os.Stderr

		if err := sifCmd.Start(); err != nil {
			sylog.Fatalf("%v", err)
		}
		if err := sifCmd.Wait(); err != nil {
			sylog.Fatalf("%v", err)
		}
	},
	Example: sifExample,
}
