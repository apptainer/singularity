// Copyright (c) 2020, Ctrl-Cmd Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"context"
	"errors"

	auth "github.com/deislabs/oras/pkg/auth/docker"
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/pkg/cmdline"
	"github.com/sylabs/singularity/pkg/syfs"
)

func init() {
	addCmdInit(func(cmdManager *cmdline.CommandManager) {
		cmdManager.RegisterCmd(LogoutCmd)
	})
}

// LogoutCmd is the 'logout' command that allows user to remove credential for a registry.
var LogoutCmd = &cobra.Command{
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		hostname := args[0]
		if hostname == "" {
			return errors.New("a hostname must be specified")
		}
		return Logout(hostname)
	},
	DisableFlagsInUseLine: true,

	Use:           docs.LogoutUse,
	Short:         docs.LogoutShort,
	Long:          docs.LogoutLong,
	Example:       docs.LogoutExample,
	SilenceErrors: true,
}

func Logout(hostname string) error {
	cli, err := auth.NewClient(syfs.DockerConf())
	if err != nil {
		return err
	}
	return cli.Logout(context.TODO(), hostname)
}
