// Copyright (c) 2020, Control Command Inc. All rights reserved.
// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/pkg/cmdline"
	"github.com/sylabs/singularity/pkg/sylog"
)

// --app
var runHelpAppNameFlag = cmdline.Flag{
	ID:           "runHelpAppNameFlag",
	Value:        &AppName,
	DefaultValue: "",
	Name:         "app",
	Usage:        "show the help for an app",
}

func init() {
	addCmdInit(func(cmdManager *cmdline.CommandManager) {
		cmdManager.RegisterCmd(RunHelpCmd)

		cmdManager.RegisterFlagForCmd(&runHelpAppNameFlag, RunHelpCmd)
	})
}

// RunHelpCmd singularity run-help <image>
var RunHelpCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Sanity check
		if _, err := os.Stat(args[0]); err != nil {
			sylog.Fatalf("container not found: %s", err)
		}

		cmdArgs := []string{"inspect", "--helpfile"}
		if AppName != "" {
			sylog.Debugf("App specified. Looking for help section of %s", AppName)
			cmdArgs = append(cmdArgs, "--app", AppName)
		}
		cmdArgs = append(cmdArgs, args[0])

		execCmd := exec.Command(filepath.Join(buildcfg.BINDIR, "singularity"), cmdArgs...)
		execCmd.Stderr = os.Stderr
		execCmd.Env = []string{}

		out, err := execCmd.Output()
		if err != nil {
			sylog.Fatalf("While getting run-help: %s", err)
		}
		if len(out) == 0 {
			fmt.Println("No help sections were defined for this image")
		} else {
			fmt.Printf("%s", string(out))
		}
	},

	Use:     docs.RunHelpUse,
	Short:   docs.RunHelpShort,
	Long:    docs.RunHelpLong,
	Example: docs.RunHelpExample,
}
