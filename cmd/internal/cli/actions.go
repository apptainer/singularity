// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"
	"strings"

	ocitypes "github.com/containers/image/types"
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/build"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	ociclient "github.com/sylabs/singularity/internal/pkg/client/oci"
	"github.com/sylabs/singularity/internal/pkg/libexec"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/uri"
	"github.com/sylabs/singularity/pkg/build/types"
	library "github.com/sylabs/singularity/pkg/client/library"
)

func init() {
	actionCmds := []*cobra.Command{
		ExecCmd,
		ShellCmd,
		RunCmd,
		TestCmd,
	}

	// TODO : the next n lines of code are repeating too much but I don't
	// know how to shorten them tonight
	for _, cmd := range actionCmds {
		cmd.Flags().AddFlag(actionFlags.Lookup("bind"))
		cmd.Flags().AddFlag(actionFlags.Lookup("contain"))
		cmd.Flags().AddFlag(actionFlags.Lookup("containall"))
		cmd.Flags().AddFlag(actionFlags.Lookup("cleanenv"))
		cmd.Flags().AddFlag(actionFlags.Lookup("home"))
		cmd.Flags().AddFlag(actionFlags.Lookup("ipc"))
		cmd.Flags().AddFlag(actionFlags.Lookup("net"))
		cmd.Flags().AddFlag(actionFlags.Lookup("network"))
		cmd.Flags().AddFlag(actionFlags.Lookup("network-args"))
		cmd.Flags().AddFlag(actionFlags.Lookup("dns"))
		cmd.Flags().AddFlag(actionFlags.Lookup("nv"))
		cmd.Flags().AddFlag(actionFlags.Lookup("overlay"))
		cmd.Flags().AddFlag(actionFlags.Lookup("pid"))
		cmd.Flags().AddFlag(actionFlags.Lookup("uts"))
		cmd.Flags().AddFlag(actionFlags.Lookup("pwd"))
		cmd.Flags().AddFlag(actionFlags.Lookup("scratch"))
		cmd.Flags().AddFlag(actionFlags.Lookup("userns"))
		cmd.Flags().AddFlag(actionFlags.Lookup("workdir"))
		cmd.Flags().AddFlag(actionFlags.Lookup("hostname"))
		cmd.Flags().AddFlag(actionFlags.Lookup("fakeroot"))
		cmd.Flags().AddFlag(actionFlags.Lookup("keep-privs"))
		cmd.Flags().AddFlag(actionFlags.Lookup("no-privs"))
		cmd.Flags().AddFlag(actionFlags.Lookup("add-caps"))
		cmd.Flags().AddFlag(actionFlags.Lookup("drop-caps"))
		cmd.Flags().AddFlag(actionFlags.Lookup("allow-setuid"))
		cmd.Flags().AddFlag(actionFlags.Lookup("writable"))
		cmd.Flags().AddFlag(actionFlags.Lookup("writable-tmpfs"))
		cmd.Flags().AddFlag(actionFlags.Lookup("no-home"))
		cmd.Flags().AddFlag(actionFlags.Lookup("no-init"))
		cmd.Flags().AddFlag(actionFlags.Lookup("security"))
		cmd.Flags().AddFlag(actionFlags.Lookup("apply-cgroups"))
		cmd.Flags().AddFlag(actionFlags.Lookup("app"))
		cmd.Flags().AddFlag(actionFlags.Lookup("containlibs"))
		cmd.Flags().AddFlag(actionFlags.Lookup("no-nv"))
		cmd.Flags().AddFlag(actionFlags.Lookup("tmpdir"))
		cmd.Flags().AddFlag(actionFlags.Lookup("nohttps"))
		cmd.Flags().AddFlag(actionFlags.Lookup("docker-username"))
		cmd.Flags().AddFlag(actionFlags.Lookup("docker-password"))
		cmd.Flags().AddFlag(actionFlags.Lookup("docker-login"))
		if cmd == ShellCmd {
			cmd.Flags().AddFlag(actionFlags.Lookup("shell"))
		}
		cmd.Flags().SetInterspersed(false)
	}

	SingularityCmd.AddCommand(ExecCmd)
	SingularityCmd.AddCommand(ShellCmd)
	SingularityCmd.AddCommand(RunCmd)
	SingularityCmd.AddCommand(TestCmd)
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
		b, err := build.NewBuild(u, imgabs, "sif", "", "", types.Options{TmpDir: tmpDir, NoTest: true, NoHTTPS: noHTTPS, DockerAuthConfig: authConf})
		if err != nil {
			return "", fmt.Errorf("unable to create new build: %v", err)
		}

		if err := b.Full(); err != nil {
			return "", fmt.Errorf("unable to build: %v", err)
		}

		sylog.Infof("Image cached as SIF at %s", imgabs)
	}

	return imgabs, nil
}

func handleLibrary(u string) (string, error) {
	libraryImage, err := library.GetImage("https://library.sylabs.io", authToken, u)
	if err != nil {
		return "", err
	}

	imageName := uri.GetName(u)
	imagePath := cache.LibraryImage(libraryImage.Hash, imageName)

	if exists, err := cache.LibraryImageExists(libraryImage.Hash, imageName); err != nil {
		return "", fmt.Errorf("unable to check if %v exists: %v", imagePath, err)
	} else if !exists {
		sylog.Infof("Downloading library image")
		if err = library.DownloadImage(imagePath, u, "https://library.sylabs.io", true, authToken); err != nil {
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

	libexec.PullShubImage(imagePath, u, true, noHTTPS)

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
		sylog.Infof("Use image from cache")
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

		image, err = handleLibrary(args[0])
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
	return
}

// ExecCmd represents the exec command
var ExecCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	TraverseChildren:      true,
	Args:                  cobra.MinimumNArgs(2),
	PreRun:                replaceURIWithImage,
	Run: func(cmd *cobra.Command, args []string) {
		a := append([]string{"/.singularity.d/actions/exec"}, args[1:]...)
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
	PreRun:                replaceURIWithImage,
	Run: func(cmd *cobra.Command, args []string) {
		a := []string{"/.singularity.d/actions/shell"}
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
	PreRun:                replaceURIWithImage,
	Run: func(cmd *cobra.Command, args []string) {
		a := append([]string{"/.singularity.d/actions/run"}, args[1:]...)
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
	PreRun:                replaceURIWithImage,
	Run: func(cmd *cobra.Command, args []string) {
		a := append([]string{"/.singularity.d/test"}, args[1:]...)
		execStarter(cmd, args[0], a, "")
	},

	Use:     docs.RunTestUse,
	Short:   docs.RunTestShort,
	Long:    docs.RunTestLong,
	Example: docs.RunTestExample,
}
