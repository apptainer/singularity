// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/instance"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/cmdline"
)

func init() {
	cmdManager.RegisterFlagForCmd(&instanceListUserFlag, instanceListCmd)
	cmdManager.RegisterFlagForCmd(&instanceListJSONFlag, instanceListCmd)
}

// -u|--user
var instanceListUser string
var instanceListUserFlag = cmdline.Flag{
	ID:           "instanceListUserFlag",
	Value:        &instanceListUser,
	DefaultValue: "",
	Name:         "user",
	ShortHand:    "u",
	Usage:        `If running as root, list instances from "<username>"`,
	Tag:          "<username>",
	EnvKeys:      []string{"USER"},
}

// -j|--json
var instanceListJSON bool
var instanceListJSONFlag = cmdline.Flag{
	ID:           "instanceListJSONFlag",
	Value:        &instanceListJSON,
	DefaultValue: false,
	Name:         "json",
	ShortHand:    "j",
	Usage:        "Print structured json instead of list",
	EnvKeys:      []string{"JSON"},
}

// singularity instance list
var instanceListCmd = &cobra.Command{
	Args: cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		name := "*"
		if len(args) > 0 {
			name = args[0]
		}
		listInstance(name)
	},
	DisableFlagsInUseLine: true,

	Use:     docs.InstanceListUse,
	Short:   docs.InstanceListShort,
	Long:    docs.InstanceListLong,
	Example: docs.InstanceListExample,
}

type instanceInfo struct {
	Instance string `json:"instance"`
	Pid      int    `json:"pid"`
	Image    string `json:"img"`
}

func listInstance(name string) {
	uid := os.Getuid()
	if instanceListUser != "" && uid != 0 {
		sylog.Fatalf("only root user can list user's instances")
	}

	files, err := instance.List(instanceListUser, name, instance.SingSubDir)
	if err != nil {
		sylog.Fatalf("failed to retrieve instance list: %v", err)
	}

	if !instanceListJSON {
		fmt.Printf("%-16s %-8s %s\n", "INSTANCE NAME", "PID", "IMAGE")
		for _, file := range files {
			fmt.Printf("%-16s %-8d %s\n", file.Name, file.Pid, file.Image)
		}
		return
	}

	instances := make([]instanceInfo, len(files))
	for i := range instances {
		instances[i].Image = files[i].Image
		instances[i].Pid = files[i].Pid
		instances[i].Instance = files[i].Name
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "\t")
	err = enc.Encode(
		map[string][]instanceInfo{
			"instances": instances,
		})
	if err != nil {
		sylog.Fatalf("error while printing structured JSON: %v", err)
	}
}
