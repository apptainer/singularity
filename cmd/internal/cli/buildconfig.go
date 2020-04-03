// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/pkg/cmdline"
)

func init() {
	addCmdInit(func(cmdManager *cmdline.CommandManager) {
		cmdManager.RegisterCmd(BuildConfigCmd)
	})
}

// BuildConfigCmd outputs a list of the compile-time parameters with which
// singularity was compiled
var BuildConfigCmd = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error {
		name := ""
		if len(args) > 0 {
			name = args[0]
		}
		if err := printParam(name); err != nil {
			return err
		}
		return nil
	},
	DisableFlagsInUseLine: true,

	Hidden:  true,
	Args:    cobra.MaximumNArgs(1),
	Use:     "buildcfg [parameter]",
	Short:   "Output the currently set compile-time parameters",
	Example: "$ singularity buildcfg",
}

func printParam(name string) error {
	params := []struct {
		name  string
		value string
	}{
		{"PACKAGE_NAME", buildcfg.PACKAGE_NAME},
		{"PACKAGE_VERSION", buildcfg.PACKAGE_VERSION},
		{"BUILDDIR", buildcfg.BUILDDIR},
		{"PREFIX", buildcfg.PREFIX},
		{"EXECPREFIX", buildcfg.EXECPREFIX},
		{"BINDIR", buildcfg.BINDIR},
		{"SBINDIR", buildcfg.SBINDIR},
		{"LIBEXECDIR", buildcfg.LIBEXECDIR},
		{"DATAROOTDIR", buildcfg.DATAROOTDIR},
		{"DATADIR", buildcfg.DATADIR},
		{"SYSCONFDIR", buildcfg.SYSCONFDIR},
		{"SHAREDSTATEDIR", buildcfg.SHAREDSTATEDIR},
		{"LOCALSTATEDIR", buildcfg.LOCALSTATEDIR},
		{"RUNSTATEDIR", buildcfg.RUNSTATEDIR},
		{"INCLUDEDIR", buildcfg.INCLUDEDIR},
		{"DOCDIR", buildcfg.DOCDIR},
		{"INFODIR", buildcfg.INFODIR},
		{"LIBDIR", buildcfg.LIBDIR},
		{"LOCALEDIR", buildcfg.LOCALEDIR},
		{"MANDIR", buildcfg.MANDIR},
		{"SINGULARITY_CONFDIR", buildcfg.SINGULARITY_CONFDIR},
		{"SESSIONDIR", buildcfg.SESSIONDIR},
		{"PLUGIN_ROOTDIR", buildcfg.PLUGIN_ROOTDIR},
		{"SINGULARITY_CONF_FILE", buildcfg.SINGULARITY_CONF_FILE},
		{"SINGULARITY_SUID_INSTALL", fmt.Sprintf("%d", buildcfg.SINGULARITY_SUID_INSTALL)},
	}

	if name != "" {
		for _, p := range params {
			if p.name == name {
				fmt.Println(p.value)
				return nil
			}
		}
		return fmt.Errorf("no variable named %q", name)
	}
	for _, p := range params {
		fmt.Printf("%s=%s\n", p.name, p.value)
	}
	return nil
}
