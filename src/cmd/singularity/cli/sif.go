// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"os"
	"os/exec"

	"github.com/singularityware/singularity/src/docs"
	"github.com/singularityware/singularity/src/pkg/buildcfg"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/spf13/cobra"
)

var (
	deffile  string
	partfile string
	content  string
	fstype   string
	parttype string
	uuid     string
)

func init() {
	SifCmd.Flags().SetInterspersed(false)
	SingularityCmd.AddCommand(SifCmd)
}

var sif = buildcfg.SBINDIR + "/sif"

// SifCmd represent the sif CLI cmd
var SifCmd = &cobra.Command{
	Use:   docs.SifUse,
	Short: docs.SifShort,
	Long:  docs.SifLong,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sif := exec.Command(sif, args...)

		sif.Stdout = os.Stdout
		sif.Stderr = os.Stderr
		sif.Stdin = os.Stdin

		if err := sif.Start(); err != nil {
			sylog.Fatalf("failed to start sif: %v\n", err)
		}
		if err := sif.Wait(); err != nil {
			sylog.Fatalf("sif failed: %v\n", err)
		}

	},
	DisableFlagParsing: true,
}
