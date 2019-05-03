// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/app/singularity"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/cmdline"
)

var ociArgs singularity.OciArgs

// -b|--bundle
var ociBundleFlag = cmdline.Flag{
	ID:           "ociBundleFlag",
	Value:        &ociArgs.BundlePath,
	DefaultValue: "",
	Name:         "bundle",
	ShortHand:    "b",
	Required:     true,
	Usage:        "specify the OCI bundle path (required)",
	Tag:          "<path>",
	EnvKeys:      []string{"BUNDLE"},
}

// -s|--sync-socket
var ociSyncSocketFlag = cmdline.Flag{
	ID:           "ociSyncSocketFlag",
	Value:        &ociArgs.SyncSocketPath,
	DefaultValue: "",
	Name:         "sync-socket",
	ShortHand:    "s",
	Usage:        "specify the path to unix socket for state synchronization",
	Tag:          "<path>",
	EnvKeys:      []string{"SYNC_SOCKET"},
}

// --empty-process
var ociCreateEmptyProcessFlag = cmdline.Flag{
	ID:           "ociCreateEmptyProcessFlag",
	Value:        &ociArgs.EmptyProcess,
	DefaultValue: false,
	Name:         "empty-process",
	Usage:        "run container without executing container process (eg: for POD container)",
	EnvKeys:      []string{"EMPTY_PROCESS"},
}

// -l|--log-path
var ociLogPathFlag = cmdline.Flag{
	ID:           "ociLogPathFlag",
	Value:        &ociArgs.LogPath,
	DefaultValue: "",
	Name:         "log-path",
	ShortHand:    "l",
	Usage:        "specify the log file path",
	Tag:          "<path>",
	EnvKeys:      []string{"LOG_PATH"},
}

// --log-format
var ociLogFormatFlag = cmdline.Flag{
	ID:           "ociLogFormatFlag",
	Value:        &ociArgs.LogFormat,
	DefaultValue: "kubernetes",
	Name:         "log-format",
	Usage:        "specify the log file format. Available formats are basic, kubernetes and json",
	Tag:          "<format>",
	EnvKeys:      []string{"LOG_FORMAT"},
}

// --pid-file
var ociPidFileFlag = cmdline.Flag{
	ID:           "ociPidFileFlag",
	Value:        &ociArgs.PidFile,
	DefaultValue: "",
	Name:         "pid-file",
	Usage:        "specify the pid file",
	Tag:          "<path>",
	EnvKeys:      []string{"PID_FILE"},
}

// -s|--signal
var ociKillSignalFlag = cmdline.Flag{
	ID:           "ociKillSignalFlag",
	Value:        &ociArgs.KillSignal,
	DefaultValue: "SIGTERM",
	Name:         "signal",
	ShortHand:    "s",
	Usage:        "signal sent to the container",
	Tag:          "<signal>",
	EnvKeys:      []string{"SIGNAL"},
}

// -f|--force
var ociKillForceFlag = cmdline.Flag{
	ID:           "ociKillForceFlag",
	Value:        &ociArgs.ForceKill,
	DefaultValue: false,
	Name:         "force",
	ShortHand:    "f",
	Usage:        "kill container process with SIGKILL",
	EnvKeys:      []string{"FORCE"},
}

// -t|--timeout
var ociKillTimeoutFlag = cmdline.Flag{
	ID:           "ociKillTimeoutFlag",
	Value:        &ociArgs.KillTimeout,
	DefaultValue: uint32(0),
	Name:         "timeout",
	ShortHand:    "t",
	Usage:        "timeout in second before killing container",
}

// -f|--from-file
var ociUpdateFromFileFlag = cmdline.Flag{
	ID:           "ociUpdateFromFileFlag",
	Value:        &ociArgs.FromFile,
	DefaultValue: "",
	Name:         "from-file",
	ShortHand:    "f",
	Usage:        "specify path to OCI JSON cgroups resource file ('-' to read from STDIN)",
	EnvKeys:      []string{"FROM_FILE"},
}

// ociContext is a variable used to describe the context of a OCI command.
// This variable is for example passed in to the EnsureRootPriv() function to
// customize the output.
var ociContext = []string{"oci"}

func init() {
	cmdManager.RegisterCmd(OciCmd)
	cmdManager.RegisterSubCmd(OciCmd, OciStartCmd)
	cmdManager.RegisterSubCmd(OciCmd, OciCreateCmd)
	cmdManager.RegisterSubCmd(OciCmd, OciRunCmd)
	cmdManager.RegisterSubCmd(OciCmd, OciDeleteCmd)
	cmdManager.RegisterSubCmd(OciCmd, OciKillCmd)
	cmdManager.RegisterSubCmd(OciCmd, OciStateCmd)
	cmdManager.RegisterSubCmd(OciCmd, OciAttachCmd)
	cmdManager.RegisterSubCmd(OciCmd, OciExecCmd)
	cmdManager.RegisterSubCmd(OciCmd, OciUpdateCmd)
	cmdManager.RegisterSubCmd(OciCmd, OciPauseCmd)
	cmdManager.RegisterSubCmd(OciCmd, OciResumeCmd)
	cmdManager.RegisterSubCmd(OciCmd, OciMountCmd)
	cmdManager.RegisterSubCmd(OciCmd, OciUmountCmd)

	cmdManager.SetCmdGroup("create_run", OciCreateCmd, OciRunCmd)
	createRunCmd := cmdManager.GetCmdGroup("create_run")

	cmdManager.RegisterFlagForCmd(&ociBundleFlag, createRunCmd...)
	cmdManager.RegisterFlagForCmd(&ociSyncSocketFlag, createRunCmd...)
	cmdManager.RegisterFlagForCmd(&ociLogPathFlag, createRunCmd...)
	cmdManager.RegisterFlagForCmd(&ociLogFormatFlag, createRunCmd...)
	cmdManager.RegisterFlagForCmd(&ociPidFileFlag, createRunCmd...)
	cmdManager.RegisterFlagForCmd(&ociCreateEmptyProcessFlag, OciCreateCmd)
	cmdManager.RegisterFlagForCmd(&ociKillForceFlag, OciKillCmd)
	cmdManager.RegisterFlagForCmd(&ociKillSignalFlag, OciKillCmd)
	cmdManager.RegisterFlagForCmd(&ociKillTimeoutFlag, OciKillCmd)
	cmdManager.RegisterFlagForCmd(&ociUpdateFromFileFlag, OciUpdateCmd)
	cmdManager.RegisterFlagForCmd(&ociSyncSocketFlag, OciStateCmd)
}

// OciCreateCmd represents oci create command.
var OciCreateCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	PreRun:                func(cmd *cobra.Command, args []string) { EnsureRootPriv(cmd, ociContext) },
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
	PreRun:                func(cmd *cobra.Command, args []string) { EnsureRootPriv(cmd, ociContext) },
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
	PreRun:                func(cmd *cobra.Command, args []string) { EnsureRootPriv(cmd, ociContext) },
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
	PreRun:                func(cmd *cobra.Command, args []string) { EnsureRootPriv(cmd, ociContext) },
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
	PreRun:                func(cmd *cobra.Command, args []string) { EnsureRootPriv(cmd, ociContext) },
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
	PreRun:                func(cmd *cobra.Command, args []string) { EnsureRootPriv(cmd, ociContext) },
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
	PreRun:                func(cmd *cobra.Command, args []string) { EnsureRootPriv(cmd, ociContext) },
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
	PreRun:                func(cmd *cobra.Command, args []string) { EnsureRootPriv(cmd, ociContext) },
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
	PreRun:                func(cmd *cobra.Command, args []string) { EnsureRootPriv(cmd, ociContext) },
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
	PreRun:                func(cmd *cobra.Command, args []string) { EnsureRootPriv(cmd, ociContext) },
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
	PreRun:                func(cmd *cobra.Command, args []string) { EnsureRootPriv(cmd, ociContext) },
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
	PreRun:                func(cmd *cobra.Command, args []string) { EnsureRootPriv(cmd, ociContext) },
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
	PreRun:                func(cmd *cobra.Command, args []string) { EnsureRootPriv(cmd, ociContext) },
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
