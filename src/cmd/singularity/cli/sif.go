// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/singularityware/singularity/src/pkg/sylog"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/singularityware/singularity/src/pkg/buildcfg"
)

func init() {
	sifCmd.Flags().SetInterspersed(false)
	singularityCmd.AddCommand(signCmd)

	sifCmd.AddCommand(sifCreate)
	sifCreate.
	// -D deffile : include definitions file `deffile'
	// -E : include environment variables
	// -P partfile : include file system partition `partfile'
	// 		-c CONTENT : freeform partition content string
	// 		-f FSTYPE : filesystem type: EXT3, SQUASHFS
	// 		-p PARTTYPE : filesystem partition type: SYSTEM, DATA, OVERLAY
	// 		-u uuid : pass a uuid to use instead of generating a new one
}

var sif = buildcfg.SBINDIR + "/sif" 

var sifCmd = &cobra.Command{
	Use:    "sif COMMAND OPTION FILE",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	Run:    nil,
}

var sifCreate = &cobra.Command{
	Use:    "sif create OPTION FILE",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) == 0 {
			fmt.Println("Error: At least one partition (-P) is required")
			os.Exit(2)
		}

		sifCmd := exec.Command(sif, args[:])
		sifout, err := sifCmd.CombinedOutput()
	if err != nil {
		sylog.Errorf("%v",err)
	}  

	},
}
