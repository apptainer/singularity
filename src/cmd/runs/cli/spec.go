// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/singularityware/singularity/src/docs"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/oci"

	"github.com/spf13/cobra"
)

func init() {
	SpecCmd.Flags().SetInterspersed(false)

	cwd, err := os.Getwd()
	if err != nil {
		sylog.Fatalf("%v", err)
	}

	SpecCmd.Flags().StringVarP(&bundlePath, "bundle", "b", cwd, "path to singularity image file (SIF), default to current directory")
	ExecRunCmd.AddCommand(SpecCmd)
}

// SpecCmd runs spec cmd
var SpecCmd = &cobra.Command{
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(args[0])
		spec, err := oci.LoadConfigSpec(args[0])
		if err != nil {
			sylog.Errorf("%v", err)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.Encode(spec)
	},
	DisableFlagsInUseLine: true,

	Use:   docs.RunsSpecUse,
	Short: docs.RunsSpecShort,
	Long:  docs.RunsSpecLong,
}
