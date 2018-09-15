// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"
	"os"
	"os/user"
	"path"
	"text/template"

	"github.com/singularityware/singularity/src/docs"
	"github.com/singularityware/singularity/src/pkg/buildcfg"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/auth"
	"github.com/spf13/cobra"
)

// Global variables for singularity CLI
var (
	debug   bool
	silent  bool
	verbose bool
	quiet   bool
)

var (
	// TokenFile holds the path to the sylabs auth token file
	defaultTokenFile, tokenFile string
	// authToken holds the sylabs auth token
	authToken, authWarning string
)

func init() {
	SingularityCmd.Flags().SetInterspersed(false)
	SingularityCmd.PersistentFlags().SetInterspersed(false)

	templateFuncs := template.FuncMap{
		"TraverseParentsUses": TraverseParentsUses,
	}
	cobra.AddTemplateFuncs(templateFuncs)

	SingularityCmd.SetHelpTemplate(docs.HelpTemplate)
	SingularityCmd.SetUsageTemplate(docs.UseTemplate)

	SingularityCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Print debugging information")
	SingularityCmd.Flags().BoolVarP(&silent, "silent", "s", false, "Only print errors")
	SingularityCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Suppress all normal output")
	SingularityCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Increase verbosity +1")
	usr, err := user.Current()
	if err != nil {
		sylog.Fatalf("Couldn't determine user home directory: %v", err)
	}
	defaultTokenFile = path.Join(usr.HomeDir, ".singularity", "sylabs-token")

	SingularityCmd.Flags().StringVar(&tokenFile, "tokenfile", defaultTokenFile, "path to the file holding your sylabs authentication token")
	VersionCmd.Flags().SetInterspersed(false)
	SingularityCmd.AddCommand(VersionCmd)
}

func setSylogMessageLevel(cmd *cobra.Command, args []string) {
	var level int

	if debug {
		level = 5
	} else if verbose {
		level = 4
	} else if quiet {
		level = -1
	} else if silent {
		level = -3
	} else {
		level = 1
	}

	sylog.SetLevel(level)
}

// SingularityCmd is the base command when called without any subcommands
var SingularityCmd = &cobra.Command{
	TraverseChildren:      true,
	DisableFlagsInUseLine: true,
	PersistentPreRun:      setSylogMessageLevel,
	Run:                   nil,

	Use:     docs.SingularityUse,
	Version: buildcfg.PACKAGE_VERSION,
	Short:   docs.SingularityShort,
	Long:    docs.SingularityLong,
	Example: docs.SingularityExample,
}

// ExecuteSingularity adds all child commands to the root command and sets
// flags appropriately. This is called by main.main(). It only needs to happen
// once to the root command (singularity).
func ExecuteSingularity() {
	if err := SingularityCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// TraverseParentsUses walks the parent commands and outputs a properly formatted use string
func TraverseParentsUses(cmd *cobra.Command) string {
	if cmd.HasParent() {
		return TraverseParentsUses(cmd.Parent()) + cmd.Use + " "
	}

	return cmd.Use + " "
}

// VersionCmd displays installed singularity version
var VersionCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(buildcfg.PACKAGE_VERSION)
	},

	Use:   "version",
	Short: "Show application version",
}

// sylabsToken process the authentication Token
// priority default_file < env < file_flag
func sylabsToken(cmd *cobra.Command, args []string) {
	if val := os.Getenv("SYLABS_TOKEN"); val != "" {
		authToken = val
	}
	if tokenFile != defaultTokenFile {
		authToken, authWarning = auth.ReadToken(tokenFile)
	}
	if authToken == "" {
		authToken, authWarning = auth.ReadToken(defaultTokenFile)
	}
	if authToken == "" && authWarning == auth.WarningTokenFileNotFound {
		sylog.Warningf("%v : Only pulls of public images will succeed", authWarning)
	}
}
