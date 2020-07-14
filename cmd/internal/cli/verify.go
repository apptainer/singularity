// Copyright (c) 2017-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sylabs/scs-key-client/client"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/app/singularity"
	scs "github.com/sylabs/singularity/internal/pkg/remote"
	"github.com/sylabs/singularity/pkg/cmdline"
	"github.com/sylabs/singularity/pkg/sylog"
	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
)

var (
	sifGroupID   uint32 // -g groupid specification
	sifDescID    uint32 // -i id specification
	localVerify  bool   // -l flag
	jsonVerify   bool   // -j flag
	verifyAll    bool
	verifyLegacy bool
)

// -u|--url
var verifyServerURIFlag = cmdline.Flag{
	ID:           "verifyServerURIFlag",
	Value:        &keyServerURI,
	DefaultValue: defaultKeyServer,
	Name:         "url",
	ShortHand:    "u",
	Usage:        "specify a URL for a key server",
	EnvKeys:      []string{"URL"},
}

// -g|--group-id
var verifySifGroupIDFlag = cmdline.Flag{
	ID:           "verifySifGroupIDFlag",
	Value:        &sifGroupID,
	DefaultValue: uint32(0),
	Name:         "group-id",
	ShortHand:    "g",
	Usage:        "verify objects with the specified group ID",
}

// --groupid (deprecated)
var verifyOldSifGroupIDFlag = cmdline.Flag{
	ID:           "verifyOldSifGroupIDFlag",
	Value:        &sifGroupID,
	DefaultValue: uint32(0),
	Name:         "groupid",
	Usage:        "verify objects with the specified group ID",
	Deprecated:   "use '--group-id'",
}

// -i|--sif-id
var verifySifDescSifIDFlag = cmdline.Flag{
	ID:           "verifySifDescSifIDFlag",
	Value:        &sifDescID,
	DefaultValue: uint32(0),
	Name:         "sif-id",
	ShortHand:    "i",
	Usage:        "verify object with the specified ID",
}

// --id (deprecated)
var verifySifDescIDFlag = cmdline.Flag{
	ID:           "verifySifDescIDFlag",
	Value:        &sifDescID,
	DefaultValue: uint32(0),
	Name:         "id",
	Usage:        "verify object with the specified ID",
	Deprecated:   "use '--sif-id'",
}

// -l|--local
var verifyLocalFlag = cmdline.Flag{
	ID:           "verifyLocalFlag",
	Value:        &localVerify,
	DefaultValue: false,
	Name:         "local",
	ShortHand:    "l",
	Usage:        "only verify with local key(s) in keyring",
	EnvKeys:      []string{"LOCAL_VERIFY"},
}

// -j|--json
var verifyJSONFlag = cmdline.Flag{
	ID:           "verifyJsonFlag",
	Value:        &jsonVerify,
	DefaultValue: false,
	Name:         "json",
	ShortHand:    "j",
	Usage:        "output json",
}

// -a|--all
var verifyAllFlag = cmdline.Flag{
	ID:           "verifyAllFlag",
	Value:        &verifyAll,
	DefaultValue: false,
	Name:         "all",
	ShortHand:    "a",
	Usage:        "verify all objects",
}

// --legacy-insecure
var verifyLegacyFlag = cmdline.Flag{
	ID:           "verifyLegacyFlag",
	Value:        &verifyLegacy,
	DefaultValue: false,
	Name:         "legacy-insecure",
	Usage:        "enable verification of (insecure) legacy signatures",
}

func init() {
	addCmdInit(func(cmdManager *cmdline.CommandManager) {
		cmdManager.RegisterCmd(VerifyCmd)

		cmdManager.RegisterFlagForCmd(&verifyServerURIFlag, VerifyCmd)
		cmdManager.RegisterFlagForCmd(&verifySifGroupIDFlag, VerifyCmd)
		cmdManager.RegisterFlagForCmd(&verifyOldSifGroupIDFlag, VerifyCmd)
		cmdManager.RegisterFlagForCmd(&verifySifDescSifIDFlag, VerifyCmd)
		cmdManager.RegisterFlagForCmd(&verifySifDescIDFlag, VerifyCmd)
		cmdManager.RegisterFlagForCmd(&verifyLocalFlag, VerifyCmd)
		cmdManager.RegisterFlagForCmd(&verifyJSONFlag, VerifyCmd)
		cmdManager.RegisterFlagForCmd(&verifyAllFlag, VerifyCmd)
		cmdManager.RegisterFlagForCmd(&verifyLegacyFlag, VerifyCmd)
	})
}

// VerifyCmd singularity verify
var VerifyCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(1),
	PreRun:                sylabsToken,

	Run: func(cmd *cobra.Command, args []string) {
		// args[0] contains image path
		doVerifyCmd(cmd, args[0])
	},

	Use:     docs.VerifyUse,
	Short:   docs.VerifyShort,
	Long:    docs.VerifyLong,
	Example: docs.VerifyExample,
}

func doVerifyCmd(cmd *cobra.Command, cpath string) {
	var opts []singularity.VerifyOpt

	// Set keyserver option, if applicable.
	if !localVerify {
		handleVerifyFlags(cmd)

		c := client.Config{
			BaseURL:   keyServerURI,
			AuthToken: authToken,
			UserAgent: useragent.Value(),
		}
		opts = append(opts, singularity.OptVerifyUseKeyServer(&c))
	}

	// Set group option, if applicable.
	if cmd.Flag(verifySifGroupIDFlag.Name).Changed || cmd.Flag(verifyOldSifGroupIDFlag.Name).Changed {
		opts = append(opts, singularity.OptVerifyGroup(sifGroupID))
	}

	// Set object option, if applicable.
	if cmd.Flag(verifySifDescSifIDFlag.Name).Changed || cmd.Flag(verifySifDescIDFlag.Name).Changed {
		opts = append(opts, singularity.OptVerifyObject(sifDescID))
	}

	// Set all option, if applicable.
	if verifyAll {
		opts = append(opts, singularity.OptVerifyAll())
	}

	// Set legacy option, if applicable.
	if verifyLegacy {
		opts = append(opts, singularity.OptVerifyLegacy())
	}

	// Set callback option.
	if jsonVerify {
		var kl keyList

		opts = append(opts, singularity.OptVerifyCallback(getJSONCallback(&kl)))

		verifyErr := singularity.Verify(cmd.Context(), cpath, opts...)

		// Always output JSON.
		if err := outputJSON(os.Stdout, kl); err != nil {
			sylog.Fatalf("Failed to output JSON: %v", err)
		}

		if verifyErr != nil {
			sylog.Fatalf("Failed to verify container: %s", verifyErr)
		}
	} else {
		opts = append(opts, singularity.OptVerifyCallback(outputVerify))

		fmt.Printf("Verifying image: %s\n", cpath)

		if err := singularity.Verify(cmd.Context(), cpath, opts...); err != nil {
			sylog.Fatalf("Failed to verify container: %s", err)
		}

		fmt.Printf("Container verified: %s\n", cpath)
	}
}

func handleVerifyFlags(cmd *cobra.Command) {
	// if we can load config and if default endpoint is set, use that
	// otherwise fall back on regular authtoken and URI behavior
	endpoint, err := sylabsRemote(remoteConfig)
	if err == scs.ErrNoDefault {
		sylog.Warningf("No default remote in use, falling back to: %v", keyServerURI)
		return
	} else if err != nil {
		sylog.Fatalf("Unable to load remote configuration: %v", err)
	}

	authToken = endpoint.Token
	if !cmd.Flags().Lookup("url").Changed {
		uri, err := endpoint.GetServiceURI("keystore")
		if err != nil {
			sylog.Fatalf("Unable to get library service URI: %v", err)
		}
		keyServerURI = uri
	}
}
