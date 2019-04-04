// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/pkg/cmdline"
)

type jsonList struct {
	Instance string `json:"instance"`
	Pid      int    `json:"pid"`
	Image    string `json:"img"`
}

// -u|--user
var instanceListUserFlag = cmdline.Flag{
	ID:           "instanceListUserFlag",
	Value:        &username,
	DefaultValue: "",
	Name:         "user",
	ShortHand:    "u",
	Usage:        `If running as root, list instances from "<username>"`,
	Tag:          "<username>",
	EnvKeys:      []string{"USER"},
}

// -j|--json
var instanceListJSONFlag = cmdline.Flag{
	ID:           "instanceListJSONFlag",
	Value:        &jsonFormat,
	DefaultValue: false,
	Name:         "json",
	ShortHand:    "j",
	Usage:        "Print structured json instead of list",
	EnvKeys:      []string{"JSON"},
}

func init() {
	flagManager.RegisterCmdFlag(&instanceListUserFlag, InstanceListCmd)
	flagManager.RegisterCmdFlag(&instanceListJSONFlag, InstanceListCmd)
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
