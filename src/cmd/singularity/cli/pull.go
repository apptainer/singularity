// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/src/docs"
	"github.com/sylabs/singularity/src/pkg/libexec"
	"github.com/sylabs/singularity/src/pkg/sylog"
	"github.com/sylabs/singularity/src/pkg/util/uri"
)

const (
	// LibraryProtocol holds the sylabs cloud library base URI
	// for more info refer to https://cloud.sylabs.io/library
	LibraryProtocol = "library"
	// ShubProtocol holds singularity hub base URI
	// for more info refer to https://singularity-hub.org/
	ShubProtocol = "shub"
)

var (
	// PullLibraryURI holds the base URI to a Sylabs library API instance
	PullLibraryURI string
	// PullImageName holds the name to be given to the pulled image
	PullImageName string
)

func init() {
	PullCmd.Flags().SetInterspersed(false)

	PullCmd.Flags().StringVar(&PullLibraryURI, "library", "https://library.sylabs.io", "the library to pull from")
	PullCmd.Flags().SetAnnotation("library", "envkey", []string{"LIBRARY"})

	PullCmd.Flags().BoolVarP(&force, "force", "F", false, "overwrite an image file if it exists")
	PullCmd.Flags().SetAnnotation("force", "envkey", []string{"FORCE"})

	PullCmd.Flags().StringVar(&PullImageName, "name", "", "specify a custom image name")
	PullCmd.Flags().Lookup("name").Hidden = true
	PullCmd.Flags().SetAnnotation("name", "envkey", []string{"NAME"})

	SingularityCmd.AddCommand(PullCmd)
}

// PullCmd singularity pull
var PullCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args:                  cobra.RangeArgs(1, 2),
	PreRun:                sylabsToken,
	Run: func(cmd *cobra.Command, args []string) {
		i := len(args) - 1 // uri is stored in args[len(args)-1]
		transport, ref := uri.SplitURI(args[i])
		if ref == "" {
			sylog.Fatalf("bad uri %s", args[i])
		}

		var name string
		if PullImageName == "" {
			name = args[0]
			if len(args) == 1 {
				name = uri.NameFromURI(args[i]) // TODO: If not library/shub & no name specified, simply put to cache
			}
		} else {
			name = PullImageName
		}

		switch transport {
		case LibraryProtocol, "":
			libexec.PullLibraryImage(name, args[i], PullLibraryURI, force, authToken)
		case ShubProtocol:
			libexec.PullShubImage(name, args[i], force)
		default:
			libexec.PullOciImage(name, args[i], force)
		}
	},

	Use:     docs.PullUse,
	Short:   docs.PullShort,
	Long:    docs.PullLong,
	Example: docs.PullExample,
}
