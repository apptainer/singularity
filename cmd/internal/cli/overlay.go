package cli

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/pkg/cmdline"
)

func init() {
	addCmdInit(func(cmdManager *cmdline.CommandManager) {
		cmdManager.RegisterCmd(OverlayCmd)
		cmdManager.RegisterSubCmd(OverlayCmd, OverlayCreateCmd)

		cmdManager.RegisterFlagForCmd(&overlaySizeFlag, OverlayCreateCmd)
		cmdManager.RegisterFlagForCmd(&overlayCreateDirFlag, OverlayCreateCmd)
	})
}

// OverlayCmd is the 'overlay' command that allows to manage writable overlay.
var OverlayCmd = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("Invalid command")
	},
	DisableFlagsInUseLine: true,

	Use:     docs.OverlayUse,
	Short:   docs.OverlayShort,
	Long:    docs.OverlayLong,
	Example: docs.OverlayExample,
}
