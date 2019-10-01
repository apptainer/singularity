// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/sylabs/singularity/cmd/internal/cli"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"golang.org/x/sys/unix"
)

func assertAccess(dir string) {
	if err := unix.Access(dir, unix.W_OK); err != nil {
		sylog.Fatalf("Given directory (%s) does not exist or is not writable by calling user", dir)
	}
}

func markdownDocs(rootCmd *cobra.Command, outDir string) {
	assertAccess(outDir)
	sylog.Infof("Creating Singularity markdown docs at %s\n", outDir)
	if err := doc.GenMarkdownTree(rootCmd, outDir); err != nil {
		sylog.Fatalf("Failed to create markdown docs for singularity\n")
	}
}

func manDocs(rootCmd *cobra.Command, outDir string) {
	assertAccess(outDir)
	sylog.Infof("Creating Singularity man pages at %s\n", outDir)
	header := &doc.GenManHeader{
		Title:   "singularity",
		Section: "1",
	}

	// works recursively on all sub-commands (thanks bauerm97)
	if err := doc.GenManTree(rootCmd, header, outDir); err != nil {
		sylog.Fatalf("Failed to create man pages for singularity\n")
	}
}

func rstDocs(rootCmd *cobra.Command, outDir string) {
	assertAccess(outDir)
	sylog.Infof("Creating Singularity RST docs at %s\n", outDir)
	if err := doc.GenReSTTreeCustom(rootCmd, outDir, func(a string) string {
		return ""
	}, func(name, ref string) string {
		return fmt.Sprintf(":ref:`%s <%s>`", name, ref)
	}); err != nil {
		sylog.Fatalf("Failed to create RST docs for singularity\n")
	}
}

func main() {
	var dir string
	var rootCmd = &cobra.Command{
		ValidArgs: []string{"markdown", "man", "rst"},
		Args:      cobra.ExactArgs(1),
		Use:       "makeDocs {markdown | man | rst}",
		Short:     "Generates Singularity documentation",
		Run: func(cmd *cobra.Command, args []string) {
			rootCmd := cli.RootCmd()
			switch args[0] {
			case "markdown":
				markdownDocs(rootCmd, dir)
			case "man":
				manDocs(rootCmd, dir)
			case "rst":
				rstDocs(rootCmd, dir)
			default:
				sylog.Fatalf("Invalid output type %s\n", args[0])
			}
		},
	}
	rootCmd.Flags().StringVarP(&dir, "dir", "d", ".", "Directory in which to put the generated documentation")
	rootCmd.Execute()
}
