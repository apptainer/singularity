// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
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
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/plugin"
	scs "github.com/sylabs/singularity/internal/pkg/remote"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/auth"
	"github.com/sylabs/singularity/pkg/cmdline"
)

var cmdManager = cmdline.NewCommandManager(SingularityCmd)
var flagManager = cmdline.NewFlagManager()

// CurrentUser holds the current user account information
var CurrentUser = getCurrentUser()

var defaultTokenFile = getDefaultTokenFile()

var (
	// TokenFile holds the path to the sylabs auth token file
	tokenFile string
	// authToken holds the sylabs auth token
	authToken, authWarning string
)

const (
	envPrefix = "SINGULARITY_"
)

// singularity command flags
var (
	debug   bool
	nocolor bool
	silent  bool
	verbose bool
	quiet   bool
)

// -d|--debug
var singDebugFlag = cmdline.Flag{
	ID:           "singDebugFlag",
	Value:        &debug,
	DefaultValue: false,
	Name:         "debug",
	ShortHand:    "d",
	Usage:        "print debugging information (highest verbosity)",
}

// --nocolor
var singNoColorFlag = cmdline.Flag{
	ID:           "singNoColorFlag",
	Value:        &nocolor,
	DefaultValue: false,
	Name:         "nocolor",
	Usage:        "print without color output (default False)",
}

// -s|--silent
var singSilentFlag = cmdline.Flag{
	ID:           "singSilentFlag",
	Value:        &silent,
	DefaultValue: false,
	Name:         "silent",
	ShortHand:    "s",
	Usage:        "only print errors",
}

// -q|--quiet
var singQuietFlag = cmdline.Flag{
	ID:           "singQuietFlag",
	Value:        &quiet,
	DefaultValue: false,
	Name:         "quiet",
	ShortHand:    "q",
	Usage:        "suppress normal output",
}

// --verbose
var singVerboseFlag = cmdline.Flag{
	ID:           "singVerboseFlag",
	Value:        &verbose,
	DefaultValue: false,
	Name:         "verbose",
	Usage:        "print additional information",
}

var singTokenFileFlag = cmdline.Flag{
	ID:           "singTokenFileFlag",
	Value:        &tokenFile,
	DefaultValue: defaultTokenFile,
	Name:         "tokenfile",
	ShortHand:    "t",
	Usage:        "path to the file holding your sylabs authentication token",
	Deprecated:   "Use 'singularity remote' to manage remote endpoints and tokens.",
}

func getCurrentUser() *user.User {
	usr, err := user.Current()
	if err != nil {
		sylog.Fatalf("Couldn't determine user account information: %v", err)
	}
	return usr
}

func getDefaultTokenFile() string {
	return path.Join(CurrentUser.HomeDir, ".singularity", "sylabs-token")
}

// initializePlugins should be called in any init() function which needs to interact with the plugin
// systems internal API. This will guarantee that any internal API calls happen AFTER all plugins
// have been properly loaded and initialized
func initializePlugins() {
	if err := plugin.InitializeAll(buildcfg.LIBEXECDIR); err != nil {
		sylog.Fatalf("Unable to initialize plugins: %s\n", err)
	}
}

func init() {
	SingularityCmd.Flags().SetInterspersed(false)
	SingularityCmd.PersistentFlags().SetInterspersed(false)

	templateFuncs := template.FuncMap{
		"TraverseParentsUses": TraverseParentsUses,
	}
	cobra.AddTemplateFuncs(templateFuncs)

	SingularityCmd.SetHelpTemplate(docs.HelpTemplate)
	SingularityCmd.SetUsageTemplate(docs.UseTemplate)

	vt := fmt.Sprintf("%s version {{printf \"%%s\" .Version}}\n", buildcfg.PACKAGE_NAME)
	SingularityCmd.SetVersionTemplate(vt)

	flagManager.RegisterCmdFlag(&singDebugFlag, SingularityCmd)
	flagManager.RegisterCmdFlag(&singNoColorFlag, SingularityCmd)
	flagManager.RegisterCmdFlag(&singSilentFlag, SingularityCmd)
	flagManager.RegisterCmdFlag(&singQuietFlag, SingularityCmd)
	flagManager.RegisterCmdFlag(&singVerboseFlag, SingularityCmd)
	flagManager.RegisterCmdFlag(&singTokenFileFlag, SingularityCmd)

	cmdManager.RegisterCmd(VersionCmd, false)

	initializePlugins()
	plugin.AddCommands(SingularityCmd)
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

func setSylogColor(cmd *cobra.Command, args []string) {
	if nocolor {
		sylog.DisableColor()
	}
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
	SilenceUsage:  true,
}

// ExecuteSingularity adds all child commands to the root command and sets
// flags appropriately. This is called by main.main(). It only needs to happen
// once to the root command (singularity).
func ExecuteSingularity() {
	if cmd, err := SingularityCmd.ExecuteC(); err != nil {
		if str := err.Error(); strings.Contains(str, "unknown flag: ") {
			flag := strings.TrimPrefix(str, "unknown flag: ")
			SingularityCmd.Printf("Invalid flag %q for command %q.\n\nOptions:\n\n%s\n",
				flag,
				cmd.Name(),
				cmd.Flags().FlagUsagesWrapped(getColumns()))
		} else {
			SingularityCmd.Println(cmd.UsageString())
		}
		SingularityCmd.Printf("Run '%s --help' for more detailed usage information.\n",
			cmd.CommandPath())
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
	Short: "Show the version for Singularity",
}

func persistentPreRun(cmd *cobra.Command, args []string) {
	setSylogMessageLevel(cmd, args)
	setSylogColor(cmd, args)
	flagManager.UpdateCmdFlagFromEnv(cmd, envPrefix)
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
	if authToken != "" {
		sylog.Warningf("sylabs-token files are deprecated. Use 'singularity remote' to manage remote endpoints and tokens.")
	}
}

// sylabsRemote returns the remote in use or an error
func sylabsRemote(filepath string) (*scs.EndPoint, error) {
	file, err := os.OpenFile(filepath, os.O_RDONLY, 0600)
	if err != nil {
		// catch non existing remotes.yaml file or missing .singularity/
		if os.IsNotExist(err) {
			return nil, scs.ErrNoDefault
		}
		return nil, fmt.Errorf("while opening remote config file: %s", err)
	}
	defer file.Close()

	c, err := scs.ReadFrom(file)
	if err != nil {
		return nil, fmt.Errorf("while parsing remote config data: %s", err)
	}

	return c.GetDefault()
}

// map of functions to use to bind flags to environment variables
var flagEnvFuncs = map[string]cmdline.EnvHandler{
	// action flags
	"bind":          cmdline.EnvAppend,
	"home":          cmdline.EnvStringNSlice,
	"overlay":       cmdline.EnvStringNSlice,
	"scratch":       cmdline.EnvStringNSlice,
	"workdir":       cmdline.EnvStringNSlice,
	"shell":         cmdline.EnvStringNSlice,
	"pwd":           cmdline.EnvStringNSlice,
	"hostname":      cmdline.EnvStringNSlice,
	"network":       cmdline.EnvStringNSlice,
	"network-args":  cmdline.EnvStringNSlice,
	"dns":           cmdline.EnvStringNSlice,
	"containlibs":   cmdline.EnvStringNSlice,
	"security":      cmdline.EnvStringNSlice,
	"apply-cgroups": cmdline.EnvStringNSlice,
	"app":           cmdline.EnvStringNSlice,

	"boot":           cmdline.EnvBool,
	"fakeroot":       cmdline.EnvBool,
	"cleanenv":       cmdline.EnvBool,
	"contain":        cmdline.EnvBool,
	"containall":     cmdline.EnvBool,
	"nv":             cmdline.EnvBool,
	"no-nv":          cmdline.EnvBool,
	"vm":             cmdline.EnvBool,
	"writable":       cmdline.EnvBool,
	"writable-tmpfs": cmdline.EnvBool,
	"no-home":        cmdline.EnvBool,
	"no-init":        cmdline.EnvBool,

	"pid":    cmdline.EnvBool,
	"ipc":    cmdline.EnvBool,
	"net":    cmdline.EnvBool,
	"uts":    cmdline.EnvBool,
	"userns": cmdline.EnvBool,

	"keep-privs":   cmdline.EnvBool,
	"no-privs":     cmdline.EnvBool,
	"add-caps":     cmdline.EnvStringNSlice,
	"drop-caps":    cmdline.EnvStringNSlice,
	"allow-setuid": cmdline.EnvBool,

	// build flags
	"sandbox": cmdline.EnvBool,
	"section": cmdline.EnvStringNSlice,
	"json":    cmdline.EnvBool,
	"name":    cmdline.EnvStringNSlice,
	// "writable": envBool, // set above for now
	"force":           cmdline.EnvBool,
	"update":          cmdline.EnvBool,
	"notest":          cmdline.EnvBool,
	"remote":          cmdline.EnvBool,
	"detached":        cmdline.EnvBool,
	"builder":         cmdline.EnvStringNSlice,
	"library":         cmdline.EnvStringNSlice,
	"nohttps":         cmdline.EnvBool,
	"no-cleanup":      cmdline.EnvBool,
	"tmpdir":          cmdline.EnvStringNSlice,
	"docker-username": cmdline.EnvStringNSlice,
	"docker-password": cmdline.EnvStringNSlice,
	"docker-login":    cmdline.EnvBool,

	// capability flags (and others)
	"user":  cmdline.EnvStringNSlice,
	"group": cmdline.EnvStringNSlice,
	"desc":  cmdline.EnvBool,
	"all":   cmdline.EnvBool,

	// instance flags
	"signal": cmdline.EnvStringNSlice,

	// keys flags
	"secret": cmdline.EnvBool,
	"url":    cmdline.EnvStringNSlice,

	// inspect flags
	"labels":      cmdline.EnvBool,
	"deffile":     cmdline.EnvBool,
	"runscript":   cmdline.EnvBool,
	"test":        cmdline.EnvBool,
	"environment": cmdline.EnvBool,
	"helpfile":    cmdline.EnvBool,
}
