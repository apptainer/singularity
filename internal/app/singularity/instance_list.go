// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build linux

package cli

import (
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
)

type jsonList struct {
	Instance string `json:"instance"`
	Pid      int    `json:"pid"`
	Image    string `json:"img"`
}

func init() {
	InstanceListCmd.Flags().SetInterspersed(false)

	// -u|--user
	InstanceListCmd.Flags().StringVarP(&username, "user", "u", "", `If running as root, list instances from "<username>"`)
	InstanceListCmd.Flags().SetAnnotation("user", "argtag", []string{"<username>"})
	InstanceListCmd.Flags().SetAnnotation("user", "envkey", []string{"USER"})

	// -j|--json
	InstanceListCmd.Flags().BoolVarP(&jsonFormat, "json", "j", false, "Print structured json instead of list")
	InstanceListCmd.Flags().SetAnnotation("json", "envkey", []string{"JSON"})
}

// InstanceListCmd singularity instance list
var InstanceListCmd = &cobra.Command{
	Args: cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		listInstance()
	},
	DisableFlagsInUseLine: true,

	Use:     docs.InstanceListUse,
	Short:   docs.InstanceListShort,
	Long:    docs.InstanceListLong,
	Example: docs.InstanceListExample,
}
