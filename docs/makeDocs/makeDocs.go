// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/sylabs/singularity/internal/app/singularity"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

func main() {
	var dir string
	var rootCmd = &cobra.Command{
		ValidArgs: []string{"markdown", "man", "rest"},
		Args:      cobra.ExactArgs(1),
		Use:       "makeDocs {markdown | man | rest}",
		Short:     "Generates Singularity documentation",
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "markdown":
				sylog.Infof("Creating Singularity markdown docs at %s\n", dir)
				if err := doc.GenMarkdownTree(cli.SingularityCmd, dir); err != nil {
					sylog.Fatalf("Failed to create markdown docs for singularity\n")
				}
			case "man":
				sylog.Infof("Creating Singularity man pages at %s\n", dir)
				header := &doc.GenManHeader{
					Title:   "singularity",
					Section: "1",
				}

				// works recursively on all sub-commands (thanks bauerm97)
				if err := doc.GenManTree(cli.SingularityCmd, header, dir); err != nil {
					sylog.Fatalf("Failed to create man page for singularity\n")
				}
			case "rest":
				sylog.Infof("Creating Singularity ReST docs at %s\n", dir)
				if err := doc.GenReSTTree(cli.SingularityCmd, dir); err != nil {
					sylog.Fatalf("Failed to create markdown docs for singularity\n")
				}

			default:
				sylog.Fatalf("Invalid output type %s\n", args[0])
			}
		},
	}
	rootCmd.Flags().StringVarP(&dir, "dir", "d", ".", "Directory in which to put the generated documentation")
	rootCmd.Execute()
}
