// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"

	"github.com/sylabs/singularity/internal/pkg/buildcfg"

	"github.com/spf13/cobra"
)

func init() {
	cmdManager.RegisterCmd(BuildConfigCmd)
}

// BuildConfigCmd outputs a list of the compile-time parameters with which
// singularity was compiled
var BuildConfigCmd = &cobra.Command{
	Run: func(cmd *cobra.Command, args []string) {
		printParam("PACKAGE_NAME", buildcfg.PACKAGE_NAME)
		printParam("PACKAGE_VERSION", buildcfg.PACKAGE_VERSION)
		printParam("BUILDDIR", buildcfg.BUILDDIR)
		printParam("PREFIX", buildcfg.PREFIX)
		printParam("EXECPREFIX", buildcfg.EXECPREFIX)
		printParam("BINDIR", buildcfg.BINDIR)
		printParam("SBINDIR", buildcfg.SBINDIR)
		printParam("LIBEXECDIR", buildcfg.LIBEXECDIR)
		printParam("DATAROOTDIR", buildcfg.DATAROOTDIR)
		printParam("DATADIR", buildcfg.DATADIR)
		printParam("SYSCONFDIR", buildcfg.SYSCONFDIR)
		printParam("SHAREDSTATEDIR", buildcfg.SHAREDSTATEDIR)
		printParam("LOCALSTATEDIR", buildcfg.LOCALSTATEDIR)
		printParam("RUNSTATEDIR", buildcfg.RUNSTATEDIR)
		printParam("INCLUDEDIR", buildcfg.INCLUDEDIR)
		printParam("DOCDIR", buildcfg.DOCDIR)
		printParam("INFODIR", buildcfg.INFODIR)
		printParam("LIBDIR", buildcfg.LIBDIR)
		printParam("LOCALEDIR", buildcfg.LOCALEDIR)
		printParam("MANDIR", buildcfg.MANDIR)
		printParam("SINGULARITY_CONFDIR", buildcfg.SINGULARITY_CONFDIR)
		printParam("SESSIONDIR", buildcfg.SESSIONDIR)
	},
	DisableFlagsInUseLine: true,

	Hidden:  true,
	Args:    cobra.ExactArgs(0),
	Use:     "buildcfg",
	Short:   "Output the currently set compile-time parameters",
	Example: "$ singularity buildcfg",
}

func printParam(n, v string) {
	fmt.Printf("%s=%s\n", n, v)
}
