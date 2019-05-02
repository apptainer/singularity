// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"
	"os"
	"strings"

	ocitypes "github.com/containers/image/types"
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/build"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	ociclient "github.com/sylabs/singularity/internal/pkg/client/oci"
	"github.com/sylabs/singularity/internal/pkg/libexec"
	"github.com/sylabs/singularity/internal/pkg/plugin"
	scs "github.com/sylabs/singularity/internal/pkg/remote"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/uri"
	"github.com/sylabs/singularity/pkg/build/types"
	library "github.com/sylabs/singularity/pkg/client/library"
)

const (
	defaultPath = "/bin:/usr/bin:/sbin:/usr/sbin:/usr/local/bin:/usr/local/sbin"
)

func init() {
	initializePlugins()
	actionCmds := []*cobra.Command{
		ExecCmd,
		ShellCmd,
		RunCmd,
		TestCmd,
	}

	for _, cmd := range actionCmds {
		for _, flagName := range platformActionFlags {
			cmd.Flags().AddFlag(actionFlags.Lookup(flagName))
		}

		plugin.AddFlagHooks(cmd.Flags())

		if cmd == ShellCmd {
			cmd.Flags().AddFlag(actionFlags.Lookup("shell"))
			cmd.Flags().AddFlag(actionFlags.Lookup("syos"))
		}
		cmd.Flags().SetInterspersed(false)

	}

	SingularityCmd.AddCommand(ExecCmd)
	SingularityCmd.AddCommand(ShellCmd)
	SingularityCmd.AddCommand(RunCmd)
	SingularityCmd.AddCommand(TestCmd)
}

// actionPreRun will run replaceURIWithImage and will also do the proper path unsetting
func actionPreRun(cmd *cobra.Command, args []string) {
	// backup user PATH
	userPath := strings.Join([]string{os.Getenv("PATH"), defaultPath}, ":")

	os.Setenv("USER_PATH", userPath)
	os.Setenv("PATH", defaultPath)

	replaceURIWithImage(cmd, args)
}

func handleOCI(cmd *cobra.Command, u string) (string, error) {
	authConf, err := makeDockerCredentials(cmd)
	if err != nil {
		sylog.Fatalf("While creating Docker credentials: %v", err)
	}

	sysCtx := &ocitypes.SystemContext{
		OCIInsecureSkipTLSVerify:    noHTTPS,
		DockerInsecureSkipTLSVerify: noHTTPS,
		DockerAuthConfig:            authConf,
	}

	sum, err := ociclient.ImageSHA(u, sysCtx)
	if err != nil {
		return "", fmt.Errorf("failed to get SHA of %v: %v", u, err)
	}

	name := uri.GetName(u)
	imgabs := cache.OciTempImage(sum, name)

	if exists, err := cache.OciTempExists(sum, name); err != nil {
		return "", fmt.Errorf("unable to check if %v exists: %v", imgabs, err)
	} else if !exists {
		sylog.Infof("Converting OCI blobs to SIF format")
		b, err := build.NewBuild(
			u,
			build.Config{
				Dest:          imgabs,
				Format:        "sif",
				AllowUnsigned: AllowUnauthenticatedBuild,
				LocalVerify:   LocalVerifyBuild,
				Opts: types.Options{
					TmpDir:           tmpDir,
					NoTest:           true,
					NoHTTPS:          noHTTPS,
					DockerAuthConfig: authConf,
				},
			},
		)
		if err != nil {
			return "", fmt.Errorf("unable to create new build: %v", err)
		}

		if err := b.Full(); err != nil {
			return "", fmt.Errorf("unable to build: %v", err)
		}

		sylog.Verbosef("Image cached as SIF at %s", imgabs)
	}

	return imgabs, nil
}

func handleLibrary(u, libraryURL string) (string, error) {
	libraryImage, err := library.GetImage(libraryURL, authToken, u)
	if err != nil {
		return "", err
	}

	imageName := uri.GetName(u)
	imagePath := cache.LibraryImage(libraryImage.Hash, imageName)

	if exists, err := cache.LibraryImageExists(libraryImage.Hash, imageName); err != nil {
		return "", fmt.Errorf("unable to check if %v exists: %v", imagePath, err)
	} else if !exists {
		sylog.Infof("Downloading library image")
		if err = library.DownloadImage(imagePath, u, libraryURL, true, authToken); err != nil {
			return "", fmt.Errorf("unable to Download Image: %v", err)
		}

		if cacheFileHash, err := library.ImageHash(imagePath); err != nil {
			return "", fmt.Errorf("Error getting ImageHash: %v", err)
		} else if cacheFileHash != libraryImage.Hash {
			return "", fmt.Errorf("Cached File Hash(%s) and Expected Hash(%s) does not match", cacheFileHash, libraryImage.Hash)
		}
	}

	return imagePath, nil
}

func handleShub(u string) (string, error) {
	imageName := uri.GetName(u)
	imagePath := cache.ShubImage("hash", imageName)

	exists, err := cache.ShubImageExists("hash", imageName)
	if err != nil {
		return "", fmt.Errorf("unable to check if %v exists: %v", imagePath, err)
	}
	if !exists {
		sylog.Infof("Downloading shub image")
		libexec.PullShubImage(imagePath, u, true, noHTTPS)
	} else {
		sylog.Verbosef("Use image from cache")
	}

	return imagePath, nil
}

func handleNet(u string) (string, error) {
	refParts := strings.Split(u, "/")
	imageName := refParts[len(refParts)-1]
	imagePath := cache.NetImage("hash", imageName)

	exists, err := cache.NetImageExists("hash", imageName)
	if err != nil {
		return "", fmt.Errorf("unable to check if %v exists: %v", imagePath, err)
	}
	if !exists {
		sylog.Infof("Downloading network image")
		libexec.PullNetImage(imagePath, u, true)
	} else {
		sylog.Verbosef("Use image from cache")
	}

	return imagePath, nil
}

func replaceURIWithImage(cmd *cobra.Command, args []string) {
	// If args[0] is not transport:ref (ex. instance://...) formatted return, not a URI
	t, _ := uri.Split(args[0])
	if t == "instance" || t == "" {
		return
	}

	var image string
	var err error

	switch t {
	case uri.Library:
		sylabsToken(cmd, args) // Fetch Auth Token for library access

		image, err = handleLibrary(args[0], handleActionRemote(cmd))
	case uri.Shub:
		image, err = handleShub(args[0])
	case ociclient.IsSupported(t):
		image, err = handleOCI(cmd, args[0])
	case uri.HTTP:
		image, err = handleNet(args[0])
	case uri.HTTPS:
		image, err = handleNet(args[0])
	default:
		sylog.Fatalf("Unsupported transport type: %s", t)
	}

	if err != nil {
		sylog.Fatalf("Unable to handle %s uri: %v", args[0], err)
	}

	args[0] = image
}

// setVM will set the --vm option if needed by other options
func setVM(cmd *cobra.Command) {
	// check if --vm-ram or --vm-cpu changed from default value
	for _, flagName := range []string{"vm-ram", "vm-cpu"} {
		if flag := cmd.Flag(flagName); flag != nil && flag.Changed {
			// this option requires the VM setting to be enabled
			cmd.Flags().Set("vm", "true")
			return
		}
	}

	// since --syos is a boolean, it cannot be added to the above list
	if IsSyOS && !VM {
		// let the user know that passing --syos implictly enables --vm
		sylog.Warningf("The --syos option requires a virtual machine, automatically enabling --vm option.")
		cmd.Flags().Set("vm", "true")
	}
}

// returns url for library and sets auth token based on remote config
// defaults to https://library.sylabs.io
func handleActionRemote(cmd *cobra.Command) string {
	defaultURI := "https://library.sylabs.io"

	// if we can load config and if default endpoint is set, use that
	// otherwise fall back on regular authtoken and URI behavior
	endpoint, err := sylabsRemote(remoteConfig)
	if err == scs.ErrNoDefault {
		sylog.Warningf("No default remote in use, falling back to %v", defaultURI)
		return defaultURI
	} else if err != nil {
		sylog.Fatalf("Unable to load remote configuration: %v", err)
	}

	authToken = endpoint.Token
	endpointURI, err := endpoint.GetServiceURI("library")
	if err != nil {
		sylog.Warningf("Unable to get library service URI: %v", err)
		return defaultURI
	}
	return endpointURI
}

// ExecCmd represents the exec command
var ExecCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	TraverseChildren:      true,
	Args:                  cobra.MinimumNArgs(2),
	PreRun:                actionPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		a := append([]string{"/.singularity.d/actions/exec"}, args[1:]...)
		setVM(cmd)
		if VM {
			execVM(cmd, args[0], a)
			return
		}
		execStarter(cmd, args[0], a, "")
	},

	Use:     docs.ExecUse,
	Short:   docs.ExecShort,
	Long:    docs.ExecLong,
	Example: docs.ExecExamples,
}

// ShellCmd represents the shell command
var ShellCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	TraverseChildren:      true,
	Args:                  cobra.MinimumNArgs(1),
	PreRun:                actionPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		a := []string{"/.singularity.d/actions/shell"}
		setVM(cmd)
		if VM {
			execVM(cmd, args[0], a)
			return
		}
		execStarter(cmd, args[0], a, "")
	},

	Use:     docs.ShellUse,
	Short:   docs.ShellShort,
	Long:    docs.ShellLong,
	Example: docs.ShellExamples,
}

// RunCmd represents the run command
var RunCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	TraverseChildren:      true,
	Args:                  cobra.MinimumNArgs(1),
	PreRun:                actionPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		a := append([]string{"/.singularity.d/actions/run"}, args[1:]...)
		setVM(cmd)
		if VM {
			execVM(cmd, args[0], a)
			return
		}
		execStarter(cmd, args[0], a, "")
	},

	Use:     docs.RunUse,
	Short:   docs.RunShort,
	Long:    docs.RunLong,
	Example: docs.RunExamples,
}

// TestCmd represents the test command
var TestCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	TraverseChildren:      true,
	Args:                  cobra.MinimumNArgs(1),
	PreRun:                actionPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		a := append([]string{"/.singularity.d/actions/test"}, args[1:]...)
		setVM(cmd)
		if VM {
			execVM(cmd, args[0], a)
			return
		}
		execStarter(cmd, args[0], a, "")
	},

	Use:     docs.RunTestUse,
	Short:   docs.RunTestShort,
	Long:    docs.RunTestLong,
	Example: docs.RunTestExample,
}
