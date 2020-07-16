// Copyright (c) 2017-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	scs "github.com/sylabs/singularity/internal/pkg/remote"
	"github.com/sylabs/singularity/pkg/cmdline"
	"github.com/sylabs/singularity/pkg/sylog"
	"github.com/sylabs/singularity/pkg/util/singularityconf"
)

const (
	defaultKeyServer = "https://keys.sylabs.io"
)

var (
	keyServerURI        string // -u command line option
	keySearchLongList   bool   // -l option for long-list
	keyNewpairBitLength int    // -b option for bit length
)

// -u|--url
var keyServerURIFlag = cmdline.Flag{
	ID:           "keyServerURIFlag",
	Value:        &keyServerURI,
	DefaultValue: defaultKeyServer,
	Name:         "url",
	ShortHand:    "u",
	Usage:        "specify the key server URL",
	EnvKeys:      []string{"URL"},
}

// -l|--long-list
var keySearchLongListFlag = cmdline.Flag{
	ID:           "keySearchLongListFlag",
	Value:        &keySearchLongList,
	DefaultValue: false,
	Name:         "long-list",
	ShortHand:    "l",
	Usage:        "output long list when searching for keys",
}

// -b|--bit-length
var keyNewpairBitLengthFlag = cmdline.Flag{
	ID:           "keyNewpairBitLengthFlag",
	Value:        &keyNewpairBitLength,
	DefaultValue: 4096,
	Name:         "bit-length",
	ShortHand:    "b",
	Usage:        "specify key bit length",
}

func init() {
	addCmdInit(func(cmdManager *cmdline.CommandManager) {
		// set the default keyserver URL
		config := singularityconf.GetCurrentConfig()

		if config != nil && config.DefaultKeyserver != "" {
			keyServerURIFlag.DefaultValue = strings.TrimSuffix(config.DefaultKeyserver, "/")
		}

		cmdManager.RegisterCmd(KeyCmd)

		cmdManager.RegisterSubCmd(KeyCmd, KeyNewPairCmd)
		cmdManager.RegisterFlagForCmd(KeyNewPairNameFlag, KeyNewPairCmd)
		cmdManager.RegisterFlagForCmd(KeyNewPairEmailFlag, KeyNewPairCmd)
		cmdManager.RegisterFlagForCmd(KeyNewPairCommentFlag, KeyNewPairCmd)
		cmdManager.RegisterFlagForCmd(KeyNewPairPasswordFlag, KeyNewPairCmd)
		cmdManager.RegisterFlagForCmd(KeyNewPairPushFlag, KeyNewPairCmd)

		cmdManager.RegisterSubCmd(KeyCmd, KeyListCmd)
		cmdManager.RegisterSubCmd(KeyCmd, KeySearchCmd)
		cmdManager.RegisterSubCmd(KeyCmd, KeyPullCmd)
		cmdManager.RegisterSubCmd(KeyCmd, KeyPushCmd)
		cmdManager.RegisterSubCmd(KeyCmd, KeyImportCmd)
		cmdManager.RegisterSubCmd(KeyCmd, KeyRemoveCmd)
		cmdManager.RegisterSubCmd(KeyCmd, KeyExportCmd)

		cmdManager.RegisterFlagForCmd(&keyServerURIFlag, KeySearchCmd, KeyPushCmd, KeyPullCmd)
		cmdManager.RegisterFlagForCmd(&keySearchLongListFlag, KeySearchCmd)
		cmdManager.RegisterFlagForCmd(&keyNewpairBitLengthFlag, KeyNewPairCmd)
		cmdManager.RegisterFlagForCmd(&keyImportWithNewPasswordFlag, KeyImportCmd)
	})
}

// KeyCmd is the 'key' command that allows management of key stores
var KeyCmd = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("Invalid command")
	},
	DisableFlagsInUseLine: true,
	Aliases:               []string{"keys"},

	Use:           docs.KeyUse,
	Short:         docs.KeyShort,
	Long:          docs.KeyLong,
	Example:       docs.KeyExample,
	SilenceErrors: true,
}

func getRemoteKeyServer(warn bool) (string, string, error) {
	// if we can load config and if default endpoint is set, use that
	// otherwise fall back on regular authtoken and URI behavior
	endpoint, err := sylabsRemote(remoteConfig)
	if err == scs.ErrNoDefault {
		if warn {
			sylog.Warningf("No default remote in use, falling back to: %v", defaultKeyServer)
		}
		return defaultKeyServer, "", nil
	} else if err != nil {
		return "", "", fmt.Errorf("unable to load remote configuration: %v", err)
	}

	// default remote endpoint, don't query the remote
	// endpoint for keystore URI, it's the defaultKeyServer
	if endpoint.URI == defaultRemote.URI {
		return defaultKeyServer, endpoint.Token, nil
	}

	authToken = endpoint.Token
	uri, err := endpoint.GetServiceURI("keystore")
	if err != nil {
		return "", "", fmt.Errorf("unable to get library service URI: %v", err)
	}

	return strings.TrimSuffix(uri, "/"), authToken, nil
}

func handleKeyFlags(cmd *cobra.Command) {
	var err error

	keyServerURI = strings.TrimSuffix(keyServerURI, "/")

	if keyServerURI != defaultKeyServer {
		uri, token, _ := getRemoteKeyServer(false)
		// if the active remote endpoint match the keyserver set
		// by user or defined by 'default keyserver' within the
		// configuration file, set the associated token if any
		if uri == keyServerURI {
			authToken = token
		}
		sylog.Debugf("Using keyserver URL: %s", keyServerURI)
		return
	}

	keyServerURI, authToken, err = getRemoteKeyServer(true)
	if err != nil {
		sylog.Fatalf("%s", err)
	}
}
