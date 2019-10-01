// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"
	"io"
	"os"
	"os/user"
	"path"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/plugin"
	scs "github.com/sylabs/singularity/internal/pkg/remote"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/auth"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/pkg/cmdline"
	"github.com/sylabs/singularity/pkg/syfs"
)

var cmdManager = cmdline.NewCommandManager(singularityCmd)

// CurrentUser holds the current user account information
var CurrentUser = getCurrentUser()

var defaultTokenFile = getDefaultTokenFile()

var (
	// TokenFile holds the path to the sylabs auth token file
	tokenFile string
	// authToken holds the sylabs auth token
	authToken, authWarning string
	// default remote configuration for comparison
	defaultRemote = scs.EndPoint{
		URI:    "cloud.sylabs.io",
		Token:  "",
		System: true,
	}
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

// -v|--verbose
var singVerboseFlag = cmdline.Flag{
	ID:           "singVerboseFlag",
	Value:        &verbose,
	DefaultValue: false,
	Name:         "verbose",
	ShortHand:    "v",
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
	return path.Join(syfs.ConfigDir(), "sylabs-token")
}

// initializePlugins should be called in any init() function which needs to interact with the plugin
// systems internal API. This will guarantee that any internal API calls happen AFTER all plugins
// have been properly loaded and initialized
func initializePlugins() {
	if err := plugin.InitializeAll(buildcfg.LIBEXECDIR); err != nil {
		sylog.Warningf("Unable to initialize plugins: %s", err)
	}
}

func init() {
	singularityCmd.Flags().SetInterspersed(false)
	singularityCmd.PersistentFlags().SetInterspersed(false)

	templateFuncs := template.FuncMap{
		"TraverseParentsUses": TraverseParentsUses,
	}
	cobra.AddTemplateFuncs(templateFuncs)

	singularityCmd.SetHelpTemplate(docs.HelpTemplate)
	singularityCmd.SetUsageTemplate(docs.UseTemplate)

	vt := fmt.Sprintf("%s version {{printf \"%%s\" .Version}}\n", buildcfg.PACKAGE_NAME)
	singularityCmd.SetVersionTemplate(vt)

	cmdManager.RegisterFlagForCmd(&singDebugFlag, singularityCmd)
	cmdManager.RegisterFlagForCmd(&singNoColorFlag, singularityCmd)
	cmdManager.RegisterFlagForCmd(&singSilentFlag, singularityCmd)
	cmdManager.RegisterFlagForCmd(&singQuietFlag, singularityCmd)
	cmdManager.RegisterFlagForCmd(&singVerboseFlag, singularityCmd)
	cmdManager.RegisterFlagForCmd(&singTokenFileFlag, singularityCmd)

	cmdManager.RegisterCmd(VersionCmd)
}

func setSylogMessageLevel() {
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

func setSylogColor() {
	if nocolor {
		sylog.DisableColor()
	}
}

// createConfDir tries to create the user's configuration directory and handles
// messages and/or errors
func createConfDir(d string) {
	if err := fs.Mkdir(d, os.ModePerm); err != nil {
		if os.IsExist(err) {
			sylog.Debugf("%s already exists. Not creating.", d)
		} else {
			sylog.Debugf("Could not create %s: %s", d, err)
		}
	} else {
		sylog.Debugf("Created %s", d)
	}
}

// singularityCmd is the base command when called without any subcommands
var singularityCmd = &cobra.Command{
	TraverseChildren:      true,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmdline.CommandError("invalid command")
	},

	Use:           docs.SingularityUse,
	Version:       buildcfg.PACKAGE_VERSION,
	Short:         docs.SingularityShort,
	Long:          docs.SingularityLong,
	Example:       docs.SingularityExample,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func persistentPreRunE(cmd *cobra.Command, _ []string) error {
	setSylogMessageLevel()
	setSylogColor()
	createConfDir(syfs.ConfigDir())
	return cmdManager.UpdateCmdFlagFromEnv(cmd, envPrefix)
}

// RootCmd returns the root singularity cobra command.
func RootCmd() *cobra.Command {
	return singularityCmd
}

// ExecuteSingularity adds all child commands to the root command and sets
// flags appropriately. This is called by main.main(). It only needs to happen
// once to the root command (singularity).
func ExecuteSingularity() {
	// set persistent pre run function here to avoid initialization loop error
	singularityCmd.PersistentPreRunE = persistentPreRunE

	for _, e := range cmdManager.GetError() {
		sylog.Errorf("%s", e)
	}
	// any error reported by command manager is considered as fatal
	cliErrors := len(cmdManager.GetError())
	if cliErrors > 0 {
		sylog.Fatalf("CLI command manager reported %d error(s)", cliErrors)
	}

	for _, m := range plugin.CLIMutators() {
		m.Mutate(cmdManager)
	}

	if cmd, err := singularityCmd.ExecuteC(); err != nil {
		name := cmd.Name()
		switch err.(type) {
		case cmdline.FlagError:
			usage := cmd.Flags().FlagUsagesWrapped(getColumns())
			singularityCmd.Printf("Error for command %q: %s\n\n", name, err)
			singularityCmd.Printf("Options for %s command:\n\n%s\n", name, usage)
		case cmdline.CommandError:
			singularityCmd.Println(cmd.UsageString())
		default:
			singularityCmd.Printf("Error for command %q: %s\n\n", name, err)
			singularityCmd.Println(cmd.UsageString())
		}
		singularityCmd.Printf("Run '%s --help' for more detailed usage information.\n",
			cmd.CommandPath())
		os.Exit(1)
	}
}

// GenBashCompletionFile
func GenBashCompletion(w io.Writer) error {
	return singularityCmd.GenBashCompletion(w)
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
}

func loadRemoteConf(filepath string) (*scs.Config, error) {
	f, err := os.OpenFile(filepath, os.O_RDONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("while opening remote config file: %s", err)
	}
	defer f.Close()

	c, err := scs.ReadFrom(f)
	if err != nil {
		return nil, fmt.Errorf("while parsing remote config data: %s", err)
	}

	return c, nil
}

// defaultRemoteLogin attempts to log in the default remote with the specified tokenfile
// this will update the user remote config if it succeeds, otherwise it will return an error
func defaultRemoteLogin(filepath string, c *scs.Config) error {
	endpoint, err := c.GetDefault()
	if err != nil {
		return err
	}

	token, warning := auth.ReadToken(defaultTokenFile)
	if warning != "" {
		// token not found, return non logged in endpoint
		return fmt.Errorf("token not found, cannot log in")
	}

	endpoint.Token = token
	if err := endpoint.VerifyToken(); err != nil {
		return err
	}

	// opening config file
	file, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("while opening remote config file: %s", err)
	}
	defer file.Close()

	// truncating file before writing new contents and syncing to commit file
	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("while truncating remote config file: %s", err)
	}

	if n, err := file.Seek(0, os.SEEK_SET); err != nil || n != 0 {
		return fmt.Errorf("failed to reset %s cursor: %s", file.Name(), err)
	}

	if _, err := c.WriteTo(file); err != nil {
		return fmt.Errorf("while writing remote config to file: %s", err)
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to flush remote config file %s: %s", file.Name(), err)
	}
	return nil
}

// sylabsRemote returns the remote in use or an error
func sylabsRemote(filepath string) (*scs.EndPoint, error) {
	var c *scs.Config

	// try to load both remotes, check for errors, sync if both exist,
	// if neither exist return errNoDefault to return to old auth behavior
	cSys, sysErr := loadRemoteConf(remoteConfigSys)
	cUsr, usrErr := loadRemoteConf(filepath)
	if sysErr != nil && usrErr != nil {
		return nil, scs.ErrNoDefault
	} else if sysErr != nil {
		c = cUsr
	} else if usrErr != nil {
		c = cSys
	} else {
		// sync cUsr with system config cSys
		if err := cUsr.SyncFrom(cSys); err != nil {
			return nil, err
		}
		c = cUsr
	}

	endpoint, err := c.GetDefault()
	if err != nil {
		return endpoint, err
	}

	// default remote without token, look for tokenfile to login with
	if *endpoint == defaultRemote {
		origEndpoint := *endpoint
		err := defaultRemoteLogin(filepath, c)
		if err != nil {
			// failed to log in, return unmodified endpoint
			return &origEndpoint, nil
		}
		sylog.Infof("Default remote in use, you are now logged in from existing tokenfile. Use 'singularity remote' commands to further manage remotes")
		return endpoint, nil
	}

	return endpoint, nil
}
