// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"os"
	"os/user"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/app/singularity"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

const (
	fileName      = "remote.yaml"
	userDir       = ".singularity"
	sysDir        = "singularity"
	remoteWarning = "no authentication token, log in with 'singularity remote` commands"
)

var (
	loginTokenFile string
	remoteConfig   string
	global         bool
)

var (
	remoteConfigUser string
	remoteConfigSys  string
)

func addGlobalFlag(c *cobra.Command) {
	c.Flags().BoolVarP(&global, "global", "g", false, "edit the list of globally configured remote endpoints")
}

func init() {
	usr, err := user.Current()
	if err != nil {
		sylog.Fatalf("Couldn't determine user home directory: %v", err)
	}

	// assemble values of remoteConfig for user/sys locations
	remoteConfigUser = filepath.Join(usr.HomeDir, userDir, fileName)
	remoteConfigSys = filepath.Join(buildcfg.SYSCONFDIR, sysDir, fileName)

	// default location of the remote.yaml file is the user directory
	RemoteCmd.Flags().StringVarP(&remoteConfig, "config", "c", remoteConfigUser, "path to the file holding remote endpoint configurations")
	// use tokenfile to log in to a remote
	RemoteLoginCmd.Flags().StringVar(&loginTokenFile, "tokenfile", "", "path to the file holding token")

	// add --global flag to remote add/remove commands
	addGlobalFlag(RemoteAddCmd)
	addGlobalFlag(RemoteRemoveCmd)

	SingularityCmd.AddCommand(RemoteCmd)
	RemoteCmd.AddCommand(RemoteAddCmd)
	RemoteCmd.AddCommand(RemoteRemoveCmd)
	RemoteCmd.AddCommand(RemoteUseCmd)
	RemoteCmd.AddCommand(RemoteListCmd)
	RemoteCmd.AddCommand(RemoteLoginCmd)
	RemoteCmd.AddCommand(RemoteStatusCmd)
}

// RemoteCmd singularity remote [...]
var RemoteCmd = &cobra.Command{
	Run: nil,

	Use:     docs.RemoteUse,
	Short:   docs.RemoteShort,
	Long:    docs.RemoteLong,
	Example: docs.RemoteExample,
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
		if err := singularity.RemoteAdd(remoteConfig, args[0], args[1], global); err != nil {
			sylog.Fatalf("%s", err)
		}
	},

	Use:     docs.RemoteAddUse,
	Short:   docs.RemoteAddShort,
	Long:    docs.RemoteAddLong,
	Example: docs.RemoteAddExample,
}

// RemoteRemoveCmd singularity remote remove [remoteName]
var RemoteRemoveCmd = &cobra.Command{
	Args:   cobra.ExactArgs(1),
	PreRun: setGlobalRemoteConfig,
	Run: func(cmd *cobra.Command, args []string) {
		if err := singularity.RemoteRemove(remoteConfig, args[0]); err != nil {
			sylog.Fatalf("%s", err)
		}
	},

	Use:     docs.RemoteRemoveUse,
	Short:   docs.RemoteRemoveShort,
	Long:    docs.RemoteRemoveLong,
	Example: docs.RemoteRemoveExample,
}

// RemoteUseCmd singularity remote use [remoteName]
var RemoteUseCmd = &cobra.Command{
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := singularity.RemoteUse(remoteConfig, remoteConfigSys, args[0]); err != nil {
			sylog.Fatalf("%s", err)
		}
	},

	Use:     docs.RemoteUseUse,
	Short:   docs.RemoteUseShort,
	Long:    docs.RemoteUseLong,
	Example: docs.RemoteUseExample,
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
}

// RemoteLoginCmd singularity remote login [remoteName]
var RemoteLoginCmd = &cobra.Command{
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := singularity.RemoteLogin(remoteConfig, remoteConfigSys, args[0], loginTokenFile); err != nil {
			sylog.Fatalf("%s", err)
		}
	},

	Use:     docs.RemoteLoginUse,
	Short:   docs.RemoteLoginShort,
	Long:    docs.RemoteLoginLong,
	Example: docs.RemoteLoginExample,
}

// RemoteStatusCmd singularity remote status [remoteName]
var RemoteStatusCmd = &cobra.Command{
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := singularity.RemoteStatus(remoteConfig, remoteConfigSys, args[0]); err != nil {
			sylog.Fatalf("%s", err)
		}
	},

	Use:     docs.RemoteStatusUse,
	Short:   docs.RemoteStatusShort,
	Long:    docs.RemoteStatusLong,
	Example: docs.RemoteStatusExample,
}
