// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/app/singularity"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

var ociArgs singularity.OciArgs

func init() {
	SingularityCmd.AddCommand(OciCmd)

	OciCreateCmd.Flags().SetInterspersed(false)
	OciCreateCmd.Flags().StringVarP(&ociArgs.BundlePath, "bundle", "b", "", "set the OCI bundle path")
	OciCreateCmd.Flags().SetAnnotation("bundle", "argtag", []string{"<path>"})
	OciCreateCmd.Flags().StringVarP(&ociArgs.SyncSocketPath, "sync-socket", "s", "", "set the state synchronization socket path. this should be a unix socket")
	OciCreateCmd.Flags().SetAnnotation("sync-socket", "argtag", []string{"<path>"})
	OciCreateCmd.Flags().BoolVar(&ociArgs.EmptyProcess, "empty-process", false, "run a container without executing a container process (eg: POD containers)")
	OciCreateCmd.Flags().StringVarP(&ociArgs.LogPath, "log-path", "l", "", "set the log file path")
	OciCreateCmd.Flags().SetAnnotation("log-path", "argtag", []string{"<path>"})
	OciCreateCmd.Flags().StringVar(&ociArgs.LogFormat, "log-format", "kubernetes", "set the log file format. this should be either basic, kubernetes or json")
	OciCreateCmd.Flags().SetAnnotation("log-format", "argtag", []string{"<format>"})
	OciCreateCmd.Flags().StringVar(&ociArgs.PidFile, "pid-file", "", "set the pid file path")
	OciCreateCmd.Flags().SetAnnotation("pid-file", "argtag", []string{"<path>"})

	OciStartCmd.Flags().SetInterspersed(false)
	OciDeleteCmd.Flags().SetInterspersed(false)
	OciAttachCmd.Flags().SetInterspersed(false)
	OciExecCmd.Flags().SetInterspersed(false)
	OciPauseCmd.Flags().SetInterspersed(false)
	OciResumeCmd.Flags().SetInterspersed(false)

	OciStateCmd.Flags().SetInterspersed(false)
	OciStateCmd.Flags().StringVarP(&ociArgs.SyncSocketPath, "sync-socket", "s", "", "specify the path to unix socket for state synchronization (internal)")
	OciStateCmd.Flags().SetAnnotation("sync-socket", "argtag", []string{"<path>"})

	OciKillCmd.Flags().SetInterspersed(false)
	OciKillCmd.Flags().StringVarP(&ociArgs.KillSignal, "signal", "s", "SIGTERM", "set the signal sent to a container")
	OciKillCmd.Flags().SetInterspersed(false)
	OciKillCmd.Flags().BoolVarP(&ociArgs.ForceKill, "force", "f", false, "set the signal sent to a container to SIGKILL")
	OciKillCmd.Flags().SetInterspersed(false)
	OciKillCmd.Flags().Uint32VarP(&ociArgs.KillTimeout, "timeout", "t", 0, "set the timeout for killing a container (in seconds)")

	OciRunCmd.Flags().SetInterspersed(false)
	OciRunCmd.Flags().StringVarP(&ociArgs.BundlePath, "bundle", "b", "", "specify the OCI bundle path")
	OciRunCmd.Flags().SetAnnotation("bundle", "argtag", []string{"<path>"})
	OciRunCmd.Flags().StringVarP(&ociArgs.LogPath, "log-path", "l", "", "specify the log file path")
	OciRunCmd.Flags().SetAnnotation("log-path", "argtag", []string{"<path>"})
	OciRunCmd.Flags().StringVar(&ociArgs.LogFormat, "log-format", "kubernetes", "specify the log file format. Available formats are basic, kubernetes and json")
	OciRunCmd.Flags().SetAnnotation("log-format", "argtag", []string{"<format>"})
	OciRunCmd.Flags().StringVar(&ociArgs.PidFile, "pid-file", "", "specify the pid file")
	OciRunCmd.Flags().SetAnnotation("pid-file", "argtag", []string{"<path>"})

	OciUpdateCmd.Flags().SetInterspersed(false)
	OciUpdateCmd.Flags().StringVarP(&ociArgs.FromFile, "from-file", "f", "", "specify path to OCI JSON cgroups resource file ('-' to read from STDIN)")

	OciCmd.AddCommand(OciStartCmd)
	OciCmd.AddCommand(OciCreateCmd)
	OciCmd.AddCommand(OciRunCmd)
	OciCmd.AddCommand(OciDeleteCmd)
	OciCmd.AddCommand(OciKillCmd)
	OciCmd.AddCommand(OciStateCmd)
	OciCmd.AddCommand(OciAttachCmd)
	OciCmd.AddCommand(OciExecCmd)
	OciCmd.AddCommand(OciUpdateCmd)
	OciCmd.AddCommand(OciPauseCmd)
	OciCmd.AddCommand(OciResumeCmd)
	OciCmd.AddCommand(OciMountCmd)
	OciCmd.AddCommand(OciUmountCmd)
}

func ensureRootPriv(cmd *cobra.Command, args []string) {
	if os.Geteuid() != 0 {
		sylog.Fatalf("command 'oci %s' requires root privileges", cmd.Name())
	}
}

// OciCreateCmd represents oci create command.
var OciCreateCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	PreRun:                ensureRootPriv,
	Run: func(cmd *cobra.Command, args []string) {
		if err := singularity.OciCreate(args[0], &ociArgs); err != nil {
			sylog.Fatalf("%s", err)
		}
	},
	Use:     docs.OciCreateUse,
	Short:   docs.OciCreateShort,
	Long:    docs.OciCreateLong,
	Example: docs.OciCreateExample,
}

// OciRunCmd allow to create/start in row.
var OciRunCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	PreRun:                ensureRootPriv,
	Run: func(cmd *cobra.Command, args []string) {
		if err := singularity.OciRun(args[0], &ociArgs); err != nil {
			sylog.Fatalf("%s", err)
		}
	},
	Use:     docs.OciRunUse,
	Short:   docs.OciRunShort,
	Long:    docs.OciRunLong,
	Example: docs.OciRunExample,
}

// OciStartCmd represents oci start command.
var OciStartCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	PreRun:                ensureRootPriv,
	Run: func(cmd *cobra.Command, args []string) {
		if err := singularity.OciStart(args[0]); err != nil {
			sylog.Fatalf("%s", err)
		}
	},
	Use:     docs.OciStartUse,
	Short:   docs.OciStartShort,
	Long:    docs.OciStartLong,
	Example: docs.OciStartExample,
}

// OciDeleteCmd represents oci delete command.
var OciDeleteCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	PreRun:                ensureRootPriv,
	Run: func(cmd *cobra.Command, args []string) {
		if err := singularity.OciDelete(args[0]); err != nil {
			sylog.Fatalf("%s", err)
		}
	},
	Use:     docs.OciDeleteUse,
	Short:   docs.OciDeleteShort,
	Long:    docs.OciDeleteLong,
	Example: docs.OciDeleteExample,
}

// OciKillCmd represents oci kill command.
var OciKillCmd = &cobra.Command{
	Args:                  cobra.MinimumNArgs(1),
	DisableFlagsInUseLine: true,
	PreRun:                ensureRootPriv,
	Run: func(cmd *cobra.Command, args []string) {
		timeout := int(ociArgs.KillTimeout)
		killSignal := ""
		if len(args) > 1 && args[1] != "" {
			killSignal = args[1]
		} else {
			killSignal = ociArgs.KillSignal
		}
		if ociArgs.ForceKill {
			killSignal = "SIGKILL"
		}
		if err := singularity.OciKill(args[0], killSignal, timeout); err != nil {
			sylog.Fatalf("%s", err)
		}
	},
	Use:     docs.OciKillUse,
	Short:   docs.OciKillShort,
	Long:    docs.OciKillLong,
	Example: docs.OciKillExample,
}

// OciStateCmd represents oci state command.
var OciStateCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	PreRun:                ensureRootPriv,
	Run: func(cmd *cobra.Command, args []string) {
		if err := singularity.OciState(args[0], &ociArgs); err != nil {
			sylog.Fatalf("%s", err)
		}
	},
	Use:     docs.OciStateUse,
	Short:   docs.OciStateShort,
	Long:    docs.OciStateLong,
	Example: docs.OciStateExample,
}

// OciAttachCmd represents oci attach command.
var OciAttachCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	PreRun:                ensureRootPriv,
	Run: func(cmd *cobra.Command, args []string) {
		if err := singularity.OciAttach(args[0]); err != nil {
			sylog.Fatalf("%s", err)
		}
	},
	Use:     docs.OciAttachUse,
	Short:   docs.OciAttachShort,
	Long:    docs.OciAttachLong,
	Example: docs.OciAttachExample,
}

// OciExecCmd represents oci exec command.
var OciExecCmd = &cobra.Command{
	Args:                  cobra.MinimumNArgs(1),
	DisableFlagsInUseLine: true,
	PreRun:                ensureRootPriv,
	Run: func(cmd *cobra.Command, args []string) {
		if err := singularity.OciExec(args[0], args[1:]); err != nil {
			sylog.Fatalf("%s", err)
		}
	},
	Use:     docs.OciExecUse,
	Short:   docs.OciExecShort,
	Long:    docs.OciExecLong,
	Example: docs.OciExecExample,
}

// OciUpdateCmd represents oci update command.
var OciUpdateCmd = &cobra.Command{
	Args:                  cobra.MinimumNArgs(1),
	DisableFlagsInUseLine: true,
	PreRun:                ensureRootPriv,
	Run: func(cmd *cobra.Command, args []string) {
		if err := singularity.OciUpdate(args[0], &ociArgs); err != nil {
			sylog.Fatalf("%s", err)
		}
	},
	Use:     docs.OciUpdateUse,
	Short:   docs.OciUpdateShort,
	Long:    docs.OciUpdateLong,
	Example: docs.OciUpdateExample,
}

// OciPauseCmd represents oci pause command.
var OciPauseCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	PreRun:                ensureRootPriv,
	Run: func(cmd *cobra.Command, args []string) {
		if err := singularity.OciPauseResume(args[0], true); err != nil {
			sylog.Fatalf("%s", err)
		}
	},
	Use:     docs.OciPauseUse,
	Short:   docs.OciPauseShort,
	Long:    docs.OciPauseLong,
	Example: docs.OciPauseExample,
}

// OciResumeCmd represents oci resume command.
var OciResumeCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	PreRun:                ensureRootPriv,
	Run: func(cmd *cobra.Command, args []string) {
		if err := singularity.OciPauseResume(args[0], false); err != nil {
			sylog.Fatalf("%s", err)
		}
	},
	Use:     docs.OciResumeUse,
	Short:   docs.OciResumeShort,
	Long:    docs.OciResumeLong,
	Example: docs.OciResumeExample,
}

// OciMountCmd represents oci mount command.
var OciMountCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(2),
	DisableFlagsInUseLine: true,
	PreRun:                ensureRootPriv,
	Run: func(cmd *cobra.Command, args []string) {
		if err := singularity.OciMount(args[0], args[1]); err != nil {
			sylog.Fatalf("%s", err)
		}
	},
	Use:     docs.OciMountUse,
	Short:   docs.OciMountShort,
	Long:    docs.OciMountLong,
	Example: docs.OciMountExample,
}

// OciUmountCmd represents oci mount command.
var OciUmountCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	PreRun:                ensureRootPriv,
	Run: func(cmd *cobra.Command, args []string) {
		if err := singularity.OciUmount(args[0]); err != nil {
			sylog.Fatalf("%s", err)
		}
	},
	Use:     docs.OciUmountUse,
	Short:   docs.OciUmountShort,
	Long:    docs.OciUmountLong,
	Example: docs.OciUmountExample,
}

// OciCmd singularity oci runtime.
var OciCmd = &cobra.Command{
	Run:                   nil,
	DisableFlagsInUseLine: true,

	Use:     docs.OciUse,
	Short:   docs.OciShort,
	Long:    docs.OciLong,
	Example: docs.OciExample,
}
