package cli

import (
	"errors"

	"github.com/hpcng/singularity/docs"
	"github.com/hpcng/singularity/pkg/cmdline"
	"github.com/spf13/cobra"
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
