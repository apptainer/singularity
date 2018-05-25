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
	Long: `
	create --  Create a new sif file with input data objects
	del    id  Delete a specified set of descriptor+object
	dump   id  Display data object content
	header --  Display SIF header
	info   id  Print data object descriptor info
	list   --  List SIF data descriptors from an input SIF file


	create options:
        -D deffile : include definitions file 'deffile'
        -E : include environment variables
        -P partfile : include file system partition 'partfile'
                -c CONTENT : freeform partition content string
                -f FSTYPE : filesystem type: EXT3, SQUASHFS
                -p PARTTYPE : filesystem partition type: SYSTEM, DATA, OVERLAY
                -u uuid : pass a uuid to use instead of generating a new one`,
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
