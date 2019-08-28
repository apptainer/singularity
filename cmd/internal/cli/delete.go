package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/sylabs/scs-library-client/client"
	"github.com/sylabs/singularity/internal/app/singularity"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/interactive"
	"github.com/sylabs/singularity/pkg/cmdline"
)

var deleteImageCmd = &cobra.Command{
	Args:   cobra.ExactArgs(1),
	PreRun: sylabsToken,
	Run: func(cmd *cobra.Command, args []string) {
		imageRef := args[0]

		libraryConfig := &client.Config{
			BaseURL:   deleteLibraryURI,
			AuthToken: authToken,
		}

		y, err := interactive.AskYNQuestion("n", fmt.Sprintf("Are you sure you want to delete %s [Y,n]", imageRef))
		if err != nil {
			sylog.Fatalf(err)
		}
		if y == "n" {
			return
		}

		err = singularity.DeleteImage(libraryConfig, imageRef, deleteImageArch)
		if err != nil {
			sylog.Fatalf(err)
		}
	},
}

var deleteImageArch string
var deleteImageArchFlag = cmdline.Flag{
	ID:           "deleteImageArchFlag",
	Value:        &deleteImageArch,
	DefaultValue: "",
	Name:         "arch",
	ShortHand:    "A",
	Required:     true,
	Usage:        "specify requested image arch",
	EnvKeys:      []string{"ARCH"},
}

var deleteLibraryURI string
var deleteLibraryURIFlag = cmdline.Flag{
	ID:           "deleteLibraryURIFlag",
	Value:        &deleteLibraryURI,
	DefaultValue: "https://library.sylabs.io",
	Name:         "library",
	Usage:        "delete images from the provided library",
	EnvKeys:      []string{"LIBRARY"},
}
