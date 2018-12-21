// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/auth"
	"github.com/sylabs/singularity/docs"
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

const (
	envPrefix = "SINGULARITY_"
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

	usr, err := user.Current()
	if err != nil {
		sylog.Fatalf("Couldn't determine user home directory: %v", err)
	}
	defaultTokenFile = path.Join(usr.HomeDir, ".singularity", "sylabs-token")

	SingularityCmd.Flags().BoolVarP(&debug, "debug", "d", false, "print debugging information (highest verbosity)")
	SingularityCmd.Flags().BoolVarP(&silent, "silent", "s", false, "only print errors")
	SingularityCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "suppress normal output")
	SingularityCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "print additional information")
	SingularityCmd.Flags().StringVarP(&tokenFile, "tokenfile", "t", defaultTokenFile, "path to the file holding your sylabs authentication token")

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
	PersistentPreRun:      persistentPreRun,
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("Invalid command")
	},

	Use:           docs.SingularityUse,
	Version:       buildcfg.PACKAGE_VERSION,
	Short:         docs.SingularityShort,
	Long:          docs.SingularityLong,
	Example:       docs.SingularityExample,
	SilenceErrors: true,
}

// ExecuteSingularity adds all child commands to the root command and sets
// flags appropriately. This is called by main.main(). It only needs to happen
// once to the root command (singularity).
func ExecuteSingularity() {
	defaultEnv := "/bin:/usr/bin:/sbin:/usr/sbin:/usr/local/bin:/usr/local/sbin"

	// backup user PATH
	userEnv := strings.Join([]string{os.Getenv("PATH"), defaultEnv}, ":")
	os.Setenv("USER_PATH", userEnv)

	os.Setenv("PATH", defaultEnv)
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

func updateFlagsFromEnv(cmd *cobra.Command) {
	cmd.Flags().VisitAll(handleEnv)
}

func handleEnv(flag *pflag.Flag) {
	envKeys, ok := flag.Annotations["envkey"]
	if !ok {
		return
	}

	for _, key := range envKeys {
		val, set := os.LookupEnv(envPrefix + key)
		if !set {
			continue
		}

		updateFn := flagEnvFuncs[flag.Name]
		updateFn(flag, val)
	}

}

func persistentPreRun(cmd *cobra.Command, args []string) {
	setSylogMessageLevel(cmd, args)
	updateFlagsFromEnv(cmd)
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

// envAppend combines command line and environment var into a single argument
func envAppend(flag *pflag.Flag, envvar string) {
	if err := flag.Value.Set(envvar); err != nil {
		sylog.Warningf("Unable to set %s to environment variable value %s", flag.Name, envvar)
	} else {
		flag.Changed = true
		sylog.Debugf("Update flag Value to: %s", flag.Value)
	}
}

// envBool sets a bool flag if the CLI option is unset and env var is set
func envBool(flag *pflag.Flag, envvar string) {
	if flag.Changed == true || envvar == "" {
		return
	}

	if err := flag.Value.Set(envvar); err != nil {
		sylog.Debugf("Unable to set flag %s to value %s: %s", flag.Name, envvar, err)
		if err := flag.Value.Set("true"); err != nil {
			sylog.Warningf("Unable to set flag %s to value %s: %s", flag.Name, envvar, err)
			return
		}
	}

	flag.Changed = true
	sylog.Debugf("Set %s Value to: %s", flag.Name, flag.Value)
}

// envStringNSlice writes to a string or slice flag if CLI option/argument
// string is unset and env var is set
func envStringNSlice(flag *pflag.Flag, envvar string) {
	if flag.Changed == true {
		return
	}

	if err := flag.Value.Set(envvar); err != nil {
		sylog.Warningf("Unable to set flag %s to value %s: %s", flag.Name, envvar, err)
		return
	}

	flag.Changed = true
	sylog.Debugf("Set %s Value to: %s", flag.Name, flag.Value)
}

type envHandle func(*pflag.Flag, string)

// map of functions to use to bind flags to environment variables
var flagEnvFuncs = map[string]envHandle{
	// action flags
	"bind":          envAppend,
	"home":          envStringNSlice,
	"overlay":       envStringNSlice,
	"scratch":       envStringNSlice,
	"workdir":       envStringNSlice,
	"shell":         envStringNSlice,
	"pwd":           envStringNSlice,
	"hostname":      envStringNSlice,
	"network":       envStringNSlice,
	"network-args":  envStringNSlice,
	"dns":           envStringNSlice,
	"containlibs":   envStringNSlice,
	"security":      envStringNSlice,
	"apply-cgroups": envStringNSlice,
	"app":           envStringNSlice,

	"boot":           envBool,
	"fakeroot":       envBool,
	"cleanenv":       envBool,
	"contain":        envBool,
	"containall":     envBool,
	"nv":             envBool,
	"no-nv":          envBool,
	"writable":       envBool,
	"writable-tmpfs": envBool,
	"no-home":        envBool,
	"no-init":        envBool,

	"pid":    envBool,
	"ipc":    envBool,
	"net":    envBool,
	"uts":    envBool,
	"userns": envBool,

	"keep-privs":   envBool,
	"no-privs":     envBool,
	"add-caps":     envStringNSlice,
	"drop-caps":    envStringNSlice,
	"allow-setuid": envBool,

	// build flags
	"sandbox": envBool,
	"section": envStringNSlice,
	"json":    envBool,
	"name":    envStringNSlice,
	// "writable": envBool, // set above for now
	"force":           envBool,
	"update":          envBool,
	"notest":          envBool,
	"remote":          envBool,
	"detached":        envBool,
	"builder":         envStringNSlice,
	"library":         envStringNSlice,
	"nohttps":         envBool,
	"tmpdir":          envStringNSlice,
	"docker-username": envStringNSlice,
	"docker-password": envStringNSlice,
	"docker-login":    envBool,

	// capability flags (and others)
	"user":  envStringNSlice,
	"group": envStringNSlice,
	"desc":  envBool,
	"all":   envBool,

	// instance flags
	"signal": envStringNSlice,

	// keys flags
	"secret": envBool,
	"url":    envStringNSlice,

	// inspect flags
	"labels":      envBool,
	"deffile":     envBool,
	"runscript":   envBool,
	"test":        envBool,
	"environment": envBool,
	"helpfile":    envBool,
}
