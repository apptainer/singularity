// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/app/singularity"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

var (
	out string
)

func init() {
	PluginCompileCmd.Flags().StringVarP(&out, "out", "o", "", "")
}

// PluginCompileCmd allows a user to compile a plugin
//
// singularity plugin compile <path> [-o name]
var PluginCompileCmd = &cobra.Command{
	Run: func(cmd *cobra.Command, args []string) {
		s, err := filepath.Abs(args[0])
		if err != nil {
			sylog.Fatalf("While sanitizing input path: %s", err)
		}
		sourceDir := filepath.Clean(s)

		destSif := out

		if destSif == "" {
			destSif = sifPath(sourceDir)
		}

		sylog.Debugf("sourceDir: %s; sifPath: %s", sourceDir, destSif)
		if err := singularity.CompilePlugin(sourceDir, destSif); err != nil {
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
