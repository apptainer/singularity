// Copyright (c) 2020, Ctrl-Cmd Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"context"
	"io/ioutil"
	"os"
	"strings"

	auth "github.com/deislabs/oras/pkg/auth/docker"
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/util/interactive"
	"github.com/sylabs/singularity/pkg/cmdline"
	"github.com/sylabs/singularity/pkg/syfs"
)

var (
	loginUsername      string
	loginPassword      string
	loginPasswordStdin bool
	loginInsecure      bool
)

// -u|--username
var loginUsernameFlag = cmdline.Flag{
	ID:           "loginUsernameFlag",
	Value:        &loginUsername,
	DefaultValue: "",
	Name:         "username",
	ShortHand:    "u",
	Usage:        "username to authenticate with (leave it empty for token authentication)",
	EnvKeys:      []string{"LOGIN_USERNAME"},
}

// -p|--password
var loginPasswordFlag = cmdline.Flag{
	ID:           "loginPasswordFlag",
	Value:        &loginPassword,
	DefaultValue: "",
	Name:         "password",
	ShortHand:    "p",
	Usage:        "password to authenticate with",
	EnvKeys:      []string{"LOGIN_PASSWORD"},
}

// --password-stdin
var loginPasswordStdinFlag = cmdline.Flag{
	ID:           "loginPasswordStdinFlag",
	Value:        &loginPasswordStdin,
	DefaultValue: false,
	Name:         "password-stdin",
	Usage:        "take password from standard input",
}

// -i|--insecure
var loginInsecureFlag = cmdline.Flag{
	ID:           "loginInsecureFlag",
	Value:        &loginInsecure,
	DefaultValue: false,
	Name:         "insecure",
	ShortHand:    "i",
	Usage:        "allow insecure login",
	EnvKeys:      []string{"LOGIN_INSECURE"},
}

func init() {
	addCmdInit(func(cmdManager *cmdline.CommandManager) {
		cmdManager.RegisterCmd(LoginCmd)

		cmdManager.RegisterFlagForCmd(&loginUsernameFlag, LoginCmd)
		cmdManager.RegisterFlagForCmd(&loginPasswordFlag, LoginCmd)
		cmdManager.RegisterFlagForCmd(&loginPasswordStdinFlag, LoginCmd)
		cmdManager.RegisterFlagForCmd(&loginInsecureFlag, LoginCmd)
	})
}

// LoginCmd is the 'login' command that allows user to login to service.
var LoginCmd = &cobra.Command{
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		hostname := args[0]
		username := loginUsername
		password := loginPassword

		if loginPasswordStdin {
			p, err := ioutil.ReadAll(os.Stdin)
			if err != nil {
				return err
			}
			password = strings.TrimSuffix(string(p), "\n")
			password = strings.TrimSuffix(password, "\r")
		} else if password == "" {
			var err error

			password, err = interactive.AskQuestionNoEcho("Password: ")
			if err != nil {
				return err
			}
		}

		return Login(username, password, hostname, loginInsecure)
	},
	DisableFlagsInUseLine: true,

	Use:           docs.LoginUse,
	Short:         docs.LoginShort,
	Long:          docs.LoginLong,
	Example:       docs.LoginExample,
	SilenceErrors: true,
}

func Login(username, password, hostname string, insecure bool) error {
	cli, err := auth.NewClient(syfs.DockerConf())
	if err != nil {
		return err
	}
	return cli.Login(context.TODO(), hostname, username, password, insecure)
}
