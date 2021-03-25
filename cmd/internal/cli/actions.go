// Copyright (c) 2020, Control Command Inc. All rights reserved.
// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/cache"
	"github.com/sylabs/singularity/internal/pkg/client/library"
	"github.com/sylabs/singularity/internal/pkg/client/net"
	"github.com/sylabs/singularity/internal/pkg/client/oci"
	"github.com/sylabs/singularity/internal/pkg/client/oras"
	"github.com/sylabs/singularity/internal/pkg/client/shub"
	"github.com/sylabs/singularity/internal/pkg/remote/endpoint"
	"github.com/sylabs/singularity/internal/pkg/util/uri"
	"github.com/sylabs/singularity/pkg/sylog"
)

const (
	defaultPath = "/bin:/usr/bin:/sbin:/usr/sbin:/usr/local/bin:/usr/local/sbin"
)

func getCacheHandle(cfg cache.Config) *cache.Handle {
	h, err := cache.New(cache.Config{
		ParentDir: os.Getenv(cache.DirEnv),
		Disable:   cfg.Disable,
	})
	if err != nil {
		sylog.Fatalf("Failed to create an image cache handle: %s", err)
	}

	return h
}

// actionPreRun will run replaceURIWithImage and will also do the proper path unsetting
func actionPreRun(cmd *cobra.Command, args []string) {
	// backup user PATH
	userPath := strings.Join([]string{os.Getenv("PATH"), defaultPath}, ":")

	os.Setenv("USER_PATH", userPath)

	ctx := context.TODO()

	replaceURIWithImage(ctx, cmd, args)

	// set PATH after pulling images to be able to find potential
	// docker credential helpers outside of standard paths
	os.Setenv("PATH", defaultPath)
}

func handleOCI(ctx context.Context, imgCache *cache.Handle, cmd *cobra.Command, pullFrom string) (string, error) {
	ociAuth, err := makeDockerCredentials(cmd)
	if err != nil {
		sylog.Fatalf("While creating Docker credentials: %v", err)
	}
	return oci.Pull(ctx, imgCache, pullFrom, tmpDir, ociAuth, noHTTPS, false)
}

func handleOras(ctx context.Context, imgCache *cache.Handle, cmd *cobra.Command, pullFrom string) (string, error) {
	ociAuth, err := makeDockerCredentials(cmd)
	if err != nil {
		return "", fmt.Errorf("while creating docker credentials: %v", err)
	}
	return oras.Pull(ctx, imgCache, pullFrom, tmpDir, ociAuth)
}

func handleLibrary(ctx context.Context, imgCache *cache.Handle, pullFrom string) (string, error) {
	c, err := getLibraryClientConfig(endpoint.SCSDefaultLibraryURI)
	if err != nil {
		return "", err
	}
	return library.Pull(ctx, imgCache, pullFrom, runtime.GOARCH, tmpDir, c)
}

func handleShub(ctx context.Context, imgCache *cache.Handle, pullFrom string) (string, error) {
	return shub.Pull(ctx, imgCache, pullFrom, tmpDir, noHTTPS)
}

func handleNet(ctx context.Context, imgCache *cache.Handle, pullFrom string) (string, error) {
	return net.Pull(ctx, imgCache, pullFrom, tmpDir)
}

func replaceURIWithImage(ctx context.Context, cmd *cobra.Command, args []string) {
	// If args[0] is not transport:ref (ex. instance://...) formatted return, not a URI
	t, _ := uri.Split(args[0])
	if t == "instance" || t == "" {
		return
	}

	var image string
	var err error

	// Create a cache handle only when we know we are are using a URI
	imgCache := getCacheHandle(cache.Config{Disable: disableCache})
	if imgCache == nil {
		sylog.Fatalf("failed to create a new image cache handle")
	}

	switch t {
	case uri.Library:
		image, err = handleLibrary(ctx, imgCache, args[0])
	case uri.Oras:
		image, err = handleOras(ctx, imgCache, cmd, args[0])
	case uri.Shub:
		image, err = handleShub(ctx, imgCache, args[0])
	case oci.IsSupported(t):
		image, err = handleOCI(ctx, imgCache, cmd, args[0])
	case uri.HTTP:
		image, err = handleNet(ctx, imgCache, args[0])
	case uri.HTTPS:
		image, err = handleNet(ctx, imgCache, args[0])
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
