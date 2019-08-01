// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/app/singularity"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/cmdline"
	"github.com/sylabs/singularity/pkg/syfs"
)

const (
	fileName      = "remote.yaml"
	sysDir        = "singularity"
	remoteWarning = "no authentication token, log in with `singularity remote login`"
)

var (
	loginTokenFile string
	remoteConfig   string
	remoteNoLogin  bool
	global         bool
)

// assemble values of remoteConfig for user/sys locations
var remoteConfigUser = filepath.Join(syfs.ConfigDir(), fileName)
var remoteConfigSys = filepath.Join(buildcfg.SYSCONFDIR, sysDir, fileName)

// -g|--global
var remoteGlobalFlag = cmdline.Flag{
	ID:           "remoteGlobalFlag",
	Value:        &global,
	DefaultValue: false,
	Name:         "global",
	ShortHand:    "g",
	Usage:        "edit the list of globally configured remote endpoints",
}

// -c|--config
var remoteConfigFlag = cmdline.Flag{
	ID:           "remoteConfigFlag",
	Value:        &remoteConfig,
	DefaultValue: remoteConfigUser,
	Name:         "config",
	ShortHand:    "c",
	Usage:        "path to the file holding remote endpoint configurations",
}

// --tokenfile
var remoteTokenFileFlag = cmdline.Flag{
	ID:           "remoteTokenFileFlag",
	Value:        &loginTokenFile,
	DefaultValue: "",
	Name:         "tokenfile",
	Usage:        "path to the file holding token",
}

// --no-login
var remoteNoLoginFlag = cmdline.Flag{
	ID:           "remoteNoLoginFlag",
	Value:        &remoteNoLogin,
	DefaultValue: false,
	Name:         "no-login",
	Usage:        "skip automatic login step",
}

func init() {
	cmdManager.RegisterCmd(RemoteCmd)
	cmdManager.RegisterSubCmd(RemoteCmd, RemoteAddCmd)
	cmdManager.RegisterSubCmd(RemoteCmd, RemoteRemoveCmd)
	cmdManager.RegisterSubCmd(RemoteCmd, RemoteUseCmd)
	cmdManager.RegisterSubCmd(RemoteCmd, RemoteListCmd)
	cmdManager.RegisterSubCmd(RemoteCmd, RemoteLoginCmd)
	cmdManager.RegisterSubCmd(RemoteCmd, RemoteStatusCmd)

	// default location of the remote.yaml file is the user directory
	cmdManager.RegisterFlagForCmd(&remoteConfigFlag, RemoteCmd)
	// use tokenfile to log in to a remote
	cmdManager.RegisterFlagForCmd(&remoteTokenFileFlag, RemoteLoginCmd, RemoteAddCmd)
	// add --global flag to remote add/remove/use commands
	cmdManager.RegisterFlagForCmd(&remoteGlobalFlag, RemoteAddCmd, RemoteRemoveCmd, RemoteUseCmd)
	// add --no-login flag to add command
	cmdManager.RegisterFlagForCmd(&remoteNoLoginFlag, RemoteAddCmd)
}

// RemoteCmd singularity remote [...]
var RemoteCmd = &cobra.Command{
	Run: nil,

	Use:     docs.RemoteUse,
	Short:   docs.RemoteShort,
	Long:    docs.RemoteLong,
	Example: docs.RemoteExample,

	DisableFlagsInUseLine: true,
}

// setGlobalRemoteConfig will assign the appropriate value to remoteConfig if the global flag is set
func setGlobalRemoteConfig(_ *cobra.Command, _ []string) {
	if !global {
		return
	}

	uid := uint32(os.Getuid())
	if uid != 0 {
		sylog.Fatalf("Unable to modify global endpoint configuration file: not root user")
	}

	// set remoteConfig value to the location of the global remote.yaml file
	remoteConfig = remoteConfigSys
}

// RemoteAddCmd singularity remote add [remoteName] [remoteURI]
var RemoteAddCmd = &cobra.Command{
	Args:   cobra.ExactArgs(2),
	PreRun: setGlobalRemoteConfig,
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		uri := args[1]
		if err := singularity.RemoteAdd(remoteConfig, name, uri, global); err != nil {
			sylog.Fatalf("%s", err)
		}
		sylog.Infof("Remote %q added.", name)

		// ensure that this was not called with global flag, otherwise this will store the token in the
		// world readable config
		if global && !remoteNoLogin {
			sylog.Infof("Global option detected. Will not automatically log into remote.")
		} else if !remoteNoLogin {
			sylog.Infof("Authenticating with remote: %s", name)
			if err := singularity.RemoteLogin(remoteConfig, remoteConfigSys, name, loginTokenFile); err != nil {
				sylog.Fatalf("%s", err)
			}
		}
	},

	Use:     docs.RemoteAddUse,
	Short:   docs.RemoteAddShort,
	Long:    docs.RemoteAddLong,
	Example: docs.RemoteAddExample,

	DisableFlagsInUseLine: true,
}

// RemoteRemoveCmd singularity remote remove [remoteName]
var RemoteRemoveCmd = &cobra.Command{
	Args:   cobra.ExactArgs(1),
	PreRun: setGlobalRemoteConfig,
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		if err := singularity.RemoteRemove(remoteConfig, name); err != nil {
			sylog.Fatalf("%s", err)
		}
		sylog.Infof("Remote %q removed.", name)
	},

	Use:     docs.RemoteRemoveUse,
	Short:   docs.RemoteRemoveShort,
	Long:    docs.RemoteRemoveLong,
	Example: docs.RemoteRemoveExample,

	DisableFlagsInUseLine: true,
}

// RemoteUseCmd singularity remote use [remoteName]
var RemoteUseCmd = &cobra.Command{
	Args:   cobra.ExactArgs(1),
	PreRun: setGlobalRemoteConfig,
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		if err := singularity.RemoteUse(remoteConfig, remoteConfigSys, name, global); err != nil {
			sylog.Fatalf("%s", err)
		}
		sylog.Infof("Remote %q now in use.", name)
	},

	Use:     docs.RemoteUseUse,
	Short:   docs.RemoteUseShort,
	Long:    docs.RemoteUseLong,
	Example: docs.RemoteUseExample,

	DisableFlagsInUseLine: true,
}

// RemoteListCmd singularity remote list
var RemoteListCmd = &cobra.Command{
	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if err := singularity.RemoteList(remoteConfig, remoteConfigSys); err != nil {
			sylog.Fatalf("%s", err)
		}
	},

	Use:     docs.RemoteListUse,
	Short:   docs.RemoteListShort,
	Long:    docs.RemoteListLong,
	Example: docs.RemoteListExample,

	DisableFlagsInUseLine: true,
}

// RemoteLoginCmd singularity remote login [remoteName]
var RemoteLoginCmd = &cobra.Command{
	Args: cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		// default to empty string to signal to RemoteLogin to use default remote
		name := ""
		if len(args) > 0 {
			name = args[0]
			sylog.Infof("Authenticating with remote: %s", name)
		} else {
			sylog.Infof("Authenticating with default remote.")
		}

		if err := singularity.RemoteLogin(remoteConfig, remoteConfigSys, name, loginTokenFile); err != nil {
			sylog.Fatalf("%s", err)
		}
	},

	Use:     docs.RemoteLoginUse,
	Short:   docs.RemoteLoginShort,
	Long:    docs.RemoteLoginLong,
	Example: docs.RemoteLoginExample,

	DisableFlagsInUseLine: true,
}

// RemoteStatusCmd singularity remote status [remoteName]
var RemoteStatusCmd = &cobra.Command{
	Args: cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		// default to empty string to signal to RemoteStatus to use default remote
		name := ""
		if len(args) > 0 {
			name = args[0]
			sylog.Infof("Checking status of remote: %s", name)
		} else {
			sylog.Infof("Checking status of default remote.")
		}

		if err := singularity.RemoteStatus(remoteConfig, remoteConfigSys, name); err != nil {
			sylog.Fatalf("%s", err)
		}
	},

	Use:     docs.RemoteStatusUse,
	Short:   docs.RemoteStatusShort,
	Long:    docs.RemoteStatusLong,
	Example: docs.RemoteStatusExample,

	DisableFlagsInUseLine: true,
}
