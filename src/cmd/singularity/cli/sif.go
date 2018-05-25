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

	sifCmd.AddCommand(sifCreate)
	sifCreate.Flags().StringVarP(&deffile, "deffile", "D","", "include definitions file 'deffile'")
	sifCreate.Flags().StringVarP(&partfile, "partfile", "P","", "include file system partition `partfile'")
	sifCreate.Flags().StringVarP(&content, "CONTENT", "c","", "freeform partition content string")
	sifCreate.Flags().StringVarP(&fstype, "FSTYPE", "f","", "filesystem type: EXT3, SQUASHFS")
	sifCreate.Flags().StringVarP(&parttype, "PARTTYPE", "p","", "filesystem partition type: SYSTEM, DATA, OVERLAY")
	sifCreate.Flags().StringVarP(&uuID, "uuid", "u","", "pass a uuid to use instead of generating a new one")
}

var sifCreateExample = `
sif create -P /tmp/fs.squash -f "SQUASHFS" -p "SYSTEM" -c "Linux" /tmp/container.sif`

var sifCmd = &cobra.Command{
	Use:  "sif [command] [option] <file>",
	Args: cobra.MinimumNArgs(1),
	Run: nil,
}

var sifCreate := &cobra.Command{
	Use:  "create [option] <file>",
	Short: "Create a new sif file with input data objects",
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var sif = buildcfg.SBINDIR + "/sif"

		sifCmd := exec.Command(sif, "create",args...)
		sifCmd.Stdout = os.Stdout
		sifCmd.Stderr = os.Stderr

		if err := sifCmd.Start(); err != nil {
			sylog.Fatalf("%v", err)
		}
		if err := sifCmd.Wait(); err != nil {
			sylog.Fatalf("%v", err)
		}
	},
	Example: sifCreateExample,
}

var sifCreate := &cobra.Command{
	Use:  "create [option] <file>",
	Short: "List SIF data descriptors from an input SIF file",
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var sif = buildcfg.SBINDIR + "/sif"

		sifCmd := exec.Command(sif, "list",args...)
		sifCmd.Stdout = os.Stdout
		sifCmd.Stderr = os.Stderr

		if err := sifCmd.Start(); err != nil {
			sylog.Fatalf("%v", err)
		}
		if err := sifCmd.Wait(); err != nil {
			sylog.Fatalf("%v", err)
		}
	},
}