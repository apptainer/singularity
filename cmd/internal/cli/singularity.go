// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	ocitypes "github.com/containers/image/v5/types"
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/plugin"
	scs "github.com/sylabs/singularity/internal/pkg/remote"
	"github.com/sylabs/singularity/internal/pkg/util/auth"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/pkg/cmdline"
	clicallback "github.com/sylabs/singularity/pkg/plugin/callback/cli"
	"github.com/sylabs/singularity/pkg/syfs"
	"github.com/sylabs/singularity/pkg/sylog"
	"github.com/sylabs/singularity/pkg/util/singularityconf"
	"golang.org/x/crypto/ssh/terminal"
)

// cmdInits holds all the init function to be called
// for commands/flags registration.
var cmdInits = make([]func(*cmdline.CommandManager), 0)

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

	dockerAuthConfig ocitypes.DockerAuthConfig
	dockerLogin      bool

	encryptionPEMPath   string
	promptForPassphrase bool
	forceOverwrite      bool
	noHTTPS             bool
	tmpDir              string
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

	configurationFile string
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

// --docker-username
var dockerUsernameFlag = cmdline.Flag{
	ID:           "dockerUsernameFlag",
	Value:        &dockerAuthConfig.Username,
	DefaultValue: "",
	Name:         "docker-username",
	Usage:        "specify a username for docker authentication",
	Hidden:       true,
	EnvKeys:      []string{"DOCKER_USERNAME"},
}

// --docker-password
var dockerPasswordFlag = cmdline.Flag{
	ID:           "dockerPasswordFlag",
	Value:        &dockerAuthConfig.Password,
	DefaultValue: "",
	Name:         "docker-password",
	Usage:        "specify a password for docker authentication",
	Hidden:       true,
	EnvKeys:      []string{"DOCKER_PASSWORD"},
}

// --docker-login
var dockerLoginFlag = cmdline.Flag{
	ID:           "dockerLoginFlag",
	Value:        &dockerLogin,
	DefaultValue: false,
	Name:         "docker-login",
	Usage:        "login to a Docker Repository interactively",
	EnvKeys:      []string{"DOCKER_LOGIN"},
}

// --passphrase
var commonPromptForPassphraseFlag = cmdline.Flag{
	ID:           "commonPromptForPassphraseFlag",
	Value:        &promptForPassphrase,
	DefaultValue: false,
	Name:         "passphrase",
	Usage:        "prompt for an encryption passphrase",
}

// --pem-path
var commonPEMFlag = cmdline.Flag{
	ID:           "actionEncryptionPEMPath",
	Value:        &encryptionPEMPath,
	DefaultValue: "",
	Name:         "pem-path",
	Usage:        "enter an path to a PEM formated RSA key for an encrypted container",
}

// -F|--force
var commonForceFlag = cmdline.Flag{
	ID:           "commonForceFlag",
	Value:        &forceOverwrite,
	DefaultValue: false,
	Name:         "force",
	ShortHand:    "F",
	Usage:        "overwrite an image file if it exists",
	EnvKeys:      []string{"FORCE"},
}

// --nohttps
var commonNoHTTPSFlag = cmdline.Flag{
	ID:           "commonNoHTTPSFlag",
	Value:        &noHTTPS,
	DefaultValue: false,
	Name:         "nohttps",
	Usage:        "do NOT use HTTPS with the docker:// transport (useful for local docker registries without a certificate)",
	EnvKeys:      []string{"NOHTTPS"},
}

// --tmpdir
var commonTmpDirFlag = cmdline.Flag{
	ID:           "commonTmpDirFlag",
	Value:        &tmpDir,
	DefaultValue: os.TempDir(),
	Hidden:       true,
	Name:         "tmpdir",
	Usage:        "specify a temporary directory to use for build",
	EnvKeys:      []string{"TMPDIR"},
}

// -c|--config
var singConfigFileFlag = cmdline.Flag{
	ID:           "singConfigFileFlag",
	Value:        &configurationFile,
	DefaultValue: buildcfg.SINGULARITY_CONF_FILE,
	Name:         "config",
	ShortHand:    "c",
	Usage:        "specify a configuration file (for root or unprivileged installation only)",
	EnvKeys:      []string{"CONFIG_FILE"},
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

func addCmdInit(cmdInit func(*cmdline.CommandManager)) {
	cmdInits = append(cmdInits, cmdInit)
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

	color := true
	if nocolor || !terminal.IsTerminal(2) {
		color = false
	}

	sylog.SetLevel(level, color)
}

// handleRemoteConf will make sure your 'remote.yaml' config file
// is the correct permission.
func handleRemoteConf(remoteConfFile string) {
	// Only check the permission if it exists.
	if fs.IsFile(remoteConfFile) {
		sylog.Debugf("Ensuring file permission of 0600 on %s", remoteConfFile)
		if err := fs.EnsureFileWithPermission(remoteConfFile, 0600); err != nil {
			sylog.Fatalf("Unable to correct the permission on %s: %s", remoteConfFile, err)
		}
	}
}

// handleConfDir tries to create the user's configuration directory and handles
// messages and/or errors.
func handleConfDir(confDir string) {
	if err := fs.Mkdir(confDir, 0700); err != nil {
		if os.IsExist(err) {
			sylog.Debugf("%s already exists. Not creating.", confDir)
			fi, err := os.Stat(confDir)
			if err != nil {
				sylog.Fatalf("Failed to retrieve information for %s: %s", confDir, err)
			}
			if fi.Mode().Perm() != 0700 {
				sylog.Debugf("Enforce permission 0700 on %s", confDir)
				// enforce permission on user configuration directory
				if err := os.Chmod(confDir, 0700); err != nil {
					// best effort as chmod could fail for various reasons (eg: readonly FS)
					sylog.Warningf("Couldn't enforce permission 0700 on %s: %s", confDir, err)
				}
			}
		} else {
			sylog.Debugf("Could not create %s: %s", confDir, err)
		}
	} else {
		sylog.Debugf("Created %s", confDir)
	}
}

func setupConfig() {
	setSylogMessageLevel()
	sylog.Debugf("Singularity version: %s", buildcfg.PACKAGE_VERSION)

	if os.Geteuid() != 0 && buildcfg.SINGULARITY_SUID_INSTALL == 1 {
		if configurationFile != singConfigFileFlag.DefaultValue {
			sylog.Fatalf("--config requires to be root or an unprivileged installation")
		}
	}

	sylog.Debugf("Parsing configuration file %s", configurationFile)
	// when the current configuration is already set it means
	// that we are generating the bash completion scripts and
	// we are not parsing the configuration file at the expected
	// location because it may not already exists at this stage
	if singularityconf.GetCurrentConfig() == nil {
		config, err := singularityconf.Parse(configurationFile)
		if err != nil {
			sylog.Fatalf("Couldn't not parse configuration file %s: %s", configurationFile, err)
		}
		singularityconf.SetCurrentConfig(config)
	}

	// Handle the config dir (~/.singularity),
	// then check the remove conf file permission.
	handleConfDir(syfs.ConfigDir())
	handleRemoteConf(syfs.RemoteConf())
}

// Init initializes and registers all singularity commands.
func Init(loadPlugins bool) {
	cmdManager := cmdline.NewCommandManager(singularityCmd)

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

	// set persistent pre run function here to avoid initialization loop error
	singularityCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		return cmdManager.UpdateCmdFlagFromEnv(cmd, envPrefix)
	}

	cmdManager.RegisterFlagForCmd(&singDebugFlag, singularityCmd)
	cmdManager.RegisterFlagForCmd(&singNoColorFlag, singularityCmd)
	cmdManager.RegisterFlagForCmd(&singSilentFlag, singularityCmd)
	cmdManager.RegisterFlagForCmd(&singQuietFlag, singularityCmd)
	cmdManager.RegisterFlagForCmd(&singVerboseFlag, singularityCmd)
	cmdManager.RegisterFlagForCmd(&singTokenFileFlag, singularityCmd)
	cmdManager.RegisterFlagForCmd(&singConfigFileFlag, singularityCmd)

	cmdManager.RegisterCmd(VersionCmd)

	// force command line parsing for --config flag used by
	// setupConfig call, errors are ignored to fail later
	// with the full set of commands. If the current configuration
	// is already set it means that we are not running in a
	// singularity CLI context and we can ignore arguments
	args := os.Args[1:]
	if singularityconf.GetCurrentConfig() != nil {
		args = nil
	}
	if err := singularityCmd.ParseFlags(args); err == nil {
		setupConfig()
	}

	// register all others commands/flags
	for _, cmdInit := range cmdInits {
		cmdInit(cmdManager)
	}

	// load plugins and register commands/flags if any
	if loadPlugins {
		callbackType := (clicallback.Command)(nil)
		callbacks, err := plugin.LoadCallbacks(callbackType)
		if err != nil {
			sylog.Fatalf("Failed to load plugins callbacks '%T': %s", callbackType, err)
		}
		for _, c := range callbacks {
			c.(clicallback.Command)(cmdManager)
		}
	}

	// any error reported by command manager is considered as fatal
	cliErrors := len(cmdManager.GetError())
	if cliErrors > 0 {
		for _, e := range cmdManager.GetError() {
			sylog.Errorf("%s", e)
		}
		sylog.Fatalf("CLI command manager reported %d error(s)", cliErrors)
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

// RootCmd returns the root singularity cobra command.
func RootCmd() *cobra.Command {
	return singularityCmd
}

// ExecuteSingularity adds all child commands to the root command and sets
// flags appropriately. This is called by main.main(). It only needs to happen
// once to the root command (singularity).
func ExecuteSingularity() {
	loadPlugins := true

	// we avoid to load installed plugins to not double load
	// them during execution of plugin compile and plugin install
	args := os.Args
	if len(args) > 1 {
		loadPlugins = !strings.HasPrefix(args[1], "plugin")
	}

	Init(loadPlugins)

	// Setup a cancellable context that will trap Ctrl-C / SIGINT
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	defer func() {
		signal.Stop(c)
		cancel()
	}()
	go func() {
		select {
		case <-c:
			sylog.Debugf("User requested cancellation with interrupt")
			cancel()
		case <-ctx.Done():
		}
	}()

	if err := singularityCmd.ExecuteContext(ctx); err != nil {
		// Find the subcommand to display more useful help, and the correct
		// subcommand name in messages - i.e. 'run' not 'singularity'
		// This is required because we previously used ExecuteC that returns the
		// subcommand... but there is no ExecuteC that variant accepts a context.
		subCmd, _, subCmdErr := singularityCmd.Find(args[1:])
		if subCmdErr != nil {
			singularityCmd.Printf("Error: %v\n\n", subCmdErr)
		}

		name := subCmd.Name()
		switch err.(type) {
		case cmdline.FlagError:
			usage := subCmd.Flags().FlagUsagesWrapped(getColumns())
			singularityCmd.Printf("Error for command %q: %s\n\n", name, err)
			singularityCmd.Printf("Options for %s command:\n\n%s\n", name, usage)
		case cmdline.CommandError:
			singularityCmd.Println(subCmd.UsageString())
		default:
			singularityCmd.Printf("Error for command %q: %s\n\n", name, err)
			singularityCmd.Println(subCmd.UsageString())
		}
		singularityCmd.Printf("Run '%s --help' for more detailed usage information.\n",
			singularityCmd.CommandPath())
		os.Exit(1)
	}
}

// UseDefaultConfig set the current configuration to
// the default configuration. Useful and called before
// the creation of documentation, bash completion scripts
// or e2e command usage report.
func UseDefaultConfig() {
	config, err := singularityconf.Parse("")
	if err != nil {
		sylog.Fatalf("Could not generate default configuration: %s", err)
	}
	singularityconf.SetCurrentConfig(config)
}

// GenBashCompletionFile
func GenBashCompletion(w io.Writer) error {
	UseDefaultConfig()
	Init(false)
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

func singularityExec(image string, args []string) (string, error) {
	// Record from stdout and store as a string to return as the contents of the file.
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	abspath, err := filepath.Abs(image)
	if err != nil {
		return "", fmt.Errorf("while determining absolute path for %s: %v", image, err)
	}

	// re-use singularity exec to grab image file content,
	// we reduce binds to the bare minimum with options below
	cmdArgs := []string{"exec", "--contain", "--no-home", "--no-nv", "--no-rocm", abspath}
	cmdArgs = append(cmdArgs, args...)

	singularityCmd := filepath.Join(buildcfg.BINDIR, "singularity")

	cmd := exec.Command(singularityCmd, cmdArgs...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	// move to the root to not bind the current working directory
	// while inspecting the image
	cmd.Dir = "/"

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("unable to process command: %s: error output:\n%s", err, stderr.String())
	}

	return stdout.String(), nil
}

// CheckRoot ensures that a command is executed with root privileges.
func CheckRoot(cmd *cobra.Command, args []string) {
	if os.Geteuid() != 0 {
		sylog.Fatalf("%q command requires root privileges", cmd.CommandPath())
	}
}

// CheckRootOrUnpriv ensures that a command is executed with root
// privileges or that Singularity is installed unprivileged.
func CheckRootOrUnpriv(cmd *cobra.Command, args []string) {
	if os.Geteuid() != 0 && buildcfg.SINGULARITY_SUID_INSTALL == 1 {
		sylog.Fatalf("%q command requires root privileges or an unprivileged installation", cmd.CommandPath())
	}
}
