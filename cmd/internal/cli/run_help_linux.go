// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/cmdline"
)

const (
	standardHelpPath = "/.singularity.d/runscript.help"
	appHelpPath      = "/scif/apps/%s/scif/runscript.help"
	runHelpCommand   = "if [ ! -f \"%s\" ]\nthen\n    echo \"No help sections were defined for this image\"\nelse\n    /bin/cat %s\nfi"
)

// --app
var runHelpAppNameFlag = cmdline.Flag{
	ID:           "runHelpAppNameFlag",
	Value:        &AppName,
	DefaultValue: "",
	Name:         "app",
	Usage:        "Show the help for an app",
	EnvKeys:      []string{"APP"},
}

func init() {
	cmdManager.RegisterCmd(RunHelpCmd)

	cmdManager.RegisterFlagForCmd(&runHelpAppNameFlag, RunHelpCmd)
}

// RunHelpCmd singularity run-help <image>
var RunHelpCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	PreRun:                sylabsToken,
	Args:                  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Sanity check
		if _, err := os.Stat(args[0]); err != nil {
			sylog.Fatalf("container not found: %s", err)
		}

		// Help prints (if set) the sourced %help section on the definition file
		abspath, err := filepath.Abs(args[0])
		if err != nil {
			sylog.Fatalf("While getting absolute path: %s", err)
		}
		name := filepath.Base(abspath)

		a := []string{"/bin/sh", "-c", getCommand(getHelpPath(cmd))}

		b, err := genericStarterCommand("Singularity help", name, abspath, a)
		if err != nil {
			sylog.Fatalf("%s", err)
		}
		fmt.Printf(string(b))
	},

	Use:     docs.RunHelpUse,
	Short:   docs.RunHelpShort,
	Long:    docs.RunHelpLong,
	Example: docs.RunHelpExample,
}

func getCommand(helpFile string) string {
	return fmt.Sprintf(runHelpCommand, helpFile, helpFile)
}

func getHelpPath(cmd *cobra.Command) string {
	if cmd.Flags().Changed("app") {
		sylog.Debugf("App specified. Looking for help section of %s", AppName)
		return fmt.Sprintf(appHelpPath, AppName)
	}

	return standardHelpPath
}
