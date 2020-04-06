// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/app/singularity"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/pkg/cmdline"
	"github.com/sylabs/singularity/pkg/sylog"
)

// -o|--out
var out string
var pluginCompileOutFlag = cmdline.Flag{
	ID:           "pluginCompileOutFlag",
	Value:        &out,
	DefaultValue: "",
	Name:         "out",
	ShortHand:    "o",
	Usage:        "path of the SIF output file",
}

// --disable-minor-check
var disableMinorCheck bool
var pluginCompileDisableMinorCheckFlag = cmdline.Flag{
	ID:           "pluginCompileDisableMinorCheckFlag",
	Value:        &disableMinorCheck,
	DefaultValue: false,
	Name:         "disable-minor-check",
	Usage:        "disable minor package version check",
}

func init() {
	addCmdInit(func(cmdManager *cmdline.CommandManager) {
		cmdManager.RegisterFlagForCmd(&pluginCompileOutFlag, PluginCompileCmd)
		cmdManager.RegisterFlagForCmd(&pluginCompileDisableMinorCheckFlag, PluginCompileCmd)
	})
}

// PluginCompileCmd allows a user to compile a plugin.
//
// singularity plugin compile <path> [-o name]
var PluginCompileCmd = &cobra.Command{
	Run: func(cmd *cobra.Command, args []string) {
		sourceDir, err := filepath.Abs(args[0])
		if err != nil {
			sylog.Fatalf("While sanitizing input path: %s", err)
		}

		exists, err := fs.PathExists(sourceDir)
		if err != nil {
			sylog.Fatalf("Could not check %q exists: %v", sourceDir, err)
		}

		if !exists {
			sylog.Fatalf("Compilation failed: %q doesn't exist", sourceDir)
		}

		destSif := out
		if destSif == "" {
			destSif = sifPath(sourceDir)
		}

		buildTags := buildcfg.GO_BUILD_TAGS

		sylog.Debugf("sourceDir: %s; sifPath: %s", sourceDir, destSif)
		err = singularity.CompilePlugin(sourceDir, destSif, buildTags, disableMinorCheck)
		if err != nil {
			sylog.Fatalf("Plugin compile failed with error: %s", err)
		}
	},
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(1),

	Use:     docs.PluginCompileUse,
	Short:   docs.PluginCompileShort,
	Long:    docs.PluginCompileLong,
	Example: docs.PluginCompileExample,
}

// sifPath returns the default path where a plugin's resulting SIF file will
// be built to when no custom -o has been set.
//
// The default behavior of this will place the resulting .sif file in the
// same directory as the source code.
func sifPath(sourceDir string) string {
	b := filepath.Base(sourceDir)
	return filepath.Join(sourceDir, b+".sif")
}
