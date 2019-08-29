package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sylabs/scs-library-client/client"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/app/singularity"
	scs "github.com/sylabs/singularity/internal/pkg/remote"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/interactive"
	"github.com/sylabs/singularity/pkg/cmdline"
)

func init() {
	cmdManager.RegisterCmd(deleteImageCmd)
	cmdManager.RegisterFlagForCmd(&deleteImageArchFlag, deleteImageCmd)
	cmdManager.RegisterFlagForCmd(&deleteLibraryURIFlag, deleteImageCmd)
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

var deleteImageCmd = &cobra.Command{
	Use:     docs.DeleteUse,
	Short:   docs.DeleteShort,
	Long:    docs.DeleteLong,
	Example: docs.DeleteExample,
	Args:    cobra.ExactArgs(1),
	PreRun:  sylabsToken,
	Run: func(cmd *cobra.Command, args []string) {
		handleDeleteFlags(cmd)

		imageRef := args[0]
		imageRef = strings.TrimPrefix(imageRef, "library://")

		libraryConfig := &client.Config{
			BaseURL:   deleteLibraryURI,
			AuthToken: authToken,
		}

		y, err := interactive.AskYNQuestion("n", fmt.Sprintf("Are you sure you want to delete %s arch[%s] [Y,n]\n", imageRef, deleteImageArch))
		if err != nil {
			sylog.Fatalf(err.Error())
		}
		if y == "n" {
			return
		}

		err = singularity.DeleteImage(libraryConfig, imageRef, deleteImageArch)
		if err != nil {
			sylog.Fatalf(err.Error())
		}

		sylog.Infof("Image %s arch[%s] deleted.", imageRef, deleteImageArch)
	},
}

func handleDeleteFlags(cmd *cobra.Command) {
	endpoint, err := sylabsRemote(remoteConfig)
	if err == scs.ErrNoDefault {
		sylog.Warningf("No default remote in use, falling back to: %v", keyServerURI)
		return
	} else if err != nil {
		sylog.Fatalf("Unable to load remote configuration: %v", err)
	}

	authToken = endpoint.Token
}
