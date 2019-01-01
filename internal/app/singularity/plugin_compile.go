// Copyright (c) 2018, Sylabs Inc. All rights reserved.
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
)

const (
	containerPath = "/home/mibauer/plugin-compile/compile_plugin.sif"
	sourcePath    = "/go/src/github.com/sylabs/singularity/plugins/"
)

// PluginCompileCmd allows a user to compile a plugin
//
// singularity plugin compile <path> [-o name]
var PluginCompileCmd = &cobra.Command{
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Compiling Plugin!")
		pluginCompile(args[0], "")
	},
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(1),

	Use:     docs.PluginCompileUse,
	Short:   docs.PluginCompileShort,
	Long:    docs.PluginCompileLong,
	Example: docs.PluginCompileExample,
}

// pluginCompile takes an input and output path. The input path
// is the directory on the host which contains the source code of
// the plugin. The output path is where the plugin .so file should
// end up.
func pluginCompile(in, out string) error {
	baseDir := filepath.Base(in)
	scmd := exec.Command("singularity", "run", "-B",
		in+":"+filepath.Join(sourcePath, baseDir),
		containerPath, baseDir)

	scmd.Stderr = os.Stderr
	scmd.Stdout = os.Stdout
	scmd.Stdin = os.Stdin
	return scmd.Run()
}
