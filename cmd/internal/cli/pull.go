// Copyright (c) 2020, Control Command Inc. All rights reserved.
// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"os"
	"path/filepath"
	"runtime"

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
	"github.com/sylabs/singularity/pkg/cmdline"
	"github.com/sylabs/singularity/pkg/sylog"
)

const (
	// LibraryProtocol holds the sylabs cloud library base URI,
	// for more info refer to https://cloud.sylabs.io/library.
	LibraryProtocol = "library"
	// ShubProtocol holds singularity hub base URI,
	// for more info refer to https://singularity-hub.org/
	ShubProtocol = "shub"
	// HTTPProtocol holds the remote http base URI.
	HTTPProtocol = "http"
	// HTTPSProtocol holds the remote https base URI.
	HTTPSProtocol = "https"
	// OrasProtocol holds the oras URI.
	OrasProtocol = "oras"
)

var (
	// pullLibraryURI holds the base URI to a Sylabs library API instance.
	pullLibraryURI string
	// pullImageName holds the name to be given to the pulled image.
	pullImageName string
	// unauthenticatedPull when true; wont ask to keep a unsigned container after pulling it.
	unauthenticatedPull bool
	// pullDir is the path that the containers will be pulled to, if set.
	pullDir string
	// pullArch is the architecture for which containers will be pulled from the
	// SCS library.
	pullArch string
)

// --arch
var pullArchFlag = cmdline.Flag{
	ID:           "pullArchFlag",
	Value:        &pullArch,
	DefaultValue: runtime.GOARCH,
	Name:         "arch",
	Usage:        "architecture to pull from library",
	EnvKeys:      []string{"PULL_ARCH"},
}

// --library
var pullLibraryURIFlag = cmdline.Flag{
	ID:           "pullLibraryURIFlag",
	Value:        &pullLibraryURI,
	DefaultValue: "",
	Name:         "library",
	Usage:        "download images from the provided library",
	EnvKeys:      []string{"LIBRARY"},
}

// --name
var pullNameFlag = cmdline.Flag{
	ID:           "pullNameFlag",
	Value:        &pullImageName,
	DefaultValue: "",
	Name:         "name",
	Hidden:       true,
	Usage:        "specify a custom image name",
	EnvKeys:      []string{"PULL_NAME"},
}

// --dir
var pullDirFlag = cmdline.Flag{
	ID:           "pullDirFlag",
	Value:        &pullDir,
	DefaultValue: "",
	Name:         "dir",
	Usage:        "download images to the specific directory",
	EnvKeys:      []string{"PULLDIR", "PULLFOLDER"},
}

// --disable-cache
var pullDisableCacheFlag = cmdline.Flag{
	ID:           "pullDisableCacheFlag",
	Value:        &disableCache,
	DefaultValue: false,
	Name:         "disable-cache",
	Usage:        "dont use cached images/blobs and dont create them",
	EnvKeys:      []string{"DISABLE_CACHE"},
}

// -U|--allow-unsigned
var pullAllowUnsignedFlag = cmdline.Flag{
	ID:           "pullAllowUnauthenticatedFlag",
	Value:        &unauthenticatedPull,
	DefaultValue: false,
	Name:         "allow-unsigned",
	ShortHand:    "U",
	Usage:        "do not require a signed container",
	EnvKeys:      []string{"ALLOW_UNSIGNED"},
	Deprecated:   `pull no longer exits with an error code in case of unsigned image. Now the flag only suppress warning message.`,
}

// --allow-unauthenticated
var pullAllowUnauthenticatedFlag = cmdline.Flag{
	ID:           "pullAllowUnauthenticatedFlag",
	Value:        &unauthenticatedPull,
	DefaultValue: false,
	Name:         "allow-unauthenticated",
	ShortHand:    "",
	Usage:        "do not require a signed container",
	EnvKeys:      []string{"ALLOW_UNAUTHENTICATED"},
	Hidden:       true,
}

func init() {
	addCmdInit(func(cmdManager *cmdline.CommandManager) {
		cmdManager.RegisterCmd(PullCmd)

		cmdManager.RegisterFlagForCmd(&commonForceFlag, PullCmd)
		cmdManager.RegisterFlagForCmd(&pullLibraryURIFlag, PullCmd)
		cmdManager.RegisterFlagForCmd(&pullNameFlag, PullCmd)
		cmdManager.RegisterFlagForCmd(&commonNoHTTPSFlag, PullCmd)
		cmdManager.RegisterFlagForCmd(&commonTmpDirFlag, PullCmd)
		cmdManager.RegisterFlagForCmd(&pullDisableCacheFlag, PullCmd)
		cmdManager.RegisterFlagForCmd(&pullDirFlag, PullCmd)

		cmdManager.RegisterFlagForCmd(&dockerUsernameFlag, PullCmd)
		cmdManager.RegisterFlagForCmd(&dockerPasswordFlag, PullCmd)
		cmdManager.RegisterFlagForCmd(&dockerLoginFlag, PullCmd)

		cmdManager.RegisterFlagForCmd(&buildNoCleanupFlag, PullCmd)
		cmdManager.RegisterFlagForCmd(&pullAllowUnsignedFlag, PullCmd)
		cmdManager.RegisterFlagForCmd(&pullAllowUnauthenticatedFlag, PullCmd)
		cmdManager.RegisterFlagForCmd(&pullArchFlag, PullCmd)
	})
}

// PullCmd singularity pull
var PullCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args:                  cobra.RangeArgs(1, 2),
	Run:                   pullRun,
	Use:                   docs.PullUse,
	Short:                 docs.PullShort,
	Long:                  docs.PullLong,
	Example:               docs.PullExample,
}

func pullRun(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()

	imgCache := getCacheHandle(cache.Config{Disable: disableCache})
	if imgCache == nil {
		sylog.Fatalf("Failed to create an image cache handle")
	}

	pullFrom := args[len(args)-1]
	transport, ref := uri.Split(pullFrom)
	if ref == "" {
		sylog.Fatalf("Bad URI %s", pullFrom)
	}

	pullTo := pullImageName
	if pullTo == "" {
		pullTo = args[0]
		if len(args) == 1 {
			if transport == "" {
				pullTo = uri.GetName("library://" + pullFrom)
			} else {
				pullTo = uri.GetName(pullFrom) // TODO: If not library/shub & no name specified, simply put to cache
			}
		}
	}

	if pullDir != "" {
		pullTo = filepath.Join(pullDir, pullTo)
	}

	_, err := os.Stat(pullTo)
	if !os.IsNotExist(err) {
		// image already exists
		if !forceOverwrite {
			sylog.Fatalf("Image file already exists: %q - will not overwrite", pullTo)
		}
	}

	switch transport {
	case LibraryProtocol, "":
		lc, err := getLibraryClientConfig(pullLibraryURI)
		if err != nil {
			sylog.Fatalf("Unable to get library client configuration: %v", err)
		}
		co, err := getKeyserverClientOpts("", endpoint.KeyserverVerifyOp)
		if err != nil {
			sylog.Fatalf("Unable to get keyserver client configuration: %v", err)
		}

		_, err = library.PullToFile(ctx, imgCache, pullTo, pullFrom, pullArch, tmpDir, lc, co)
		if err != nil && err != library.ErrLibraryPullUnsigned {
			sylog.Fatalf("While pulling library image: %v", err)
		}
		if err == library.ErrLibraryPullUnsigned {
			sylog.Warningf("Skipping container verification")
		}
	case ShubProtocol:
		_, err := shub.PullToFile(ctx, imgCache, pullTo, pullFrom, tmpDir, noHTTPS)
		if err != nil {
			sylog.Fatalf("While pulling shub image: %v\n", err)
		}
	case OrasProtocol:
		ociAuth, err := makeDockerCredentials(cmd)
		if err != nil {
			sylog.Fatalf("Unable to make docker oci credentials: %s", err)
		}

		_, err = oras.PullToFile(ctx, imgCache, pullTo, pullFrom, tmpDir, ociAuth)
		if err != nil {
			sylog.Fatalf("While pulling image from oci registry: %v", err)
		}
	case HTTPProtocol, HTTPSProtocol:
		_, err := net.PullToFile(ctx, imgCache, pullTo, pullFrom, tmpDir)
		if err != nil {
			sylog.Fatalf("While pulling from image from http(s): %v\n", err)
		}
	case oci.IsSupported(transport):
		ociAuth, err := makeDockerCredentials(cmd)
		if err != nil {
			sylog.Fatalf("While creating Docker credentials: %v", err)
		}

		_, err = oci.PullToFile(ctx, imgCache, pullTo, pullFrom, tmpDir, ociAuth, noHTTPS, buildArgs.noCleanUp)
		if err != nil {
			sylog.Fatalf("While making image from oci registry: %v", err)
		}
	default:
		sylog.Fatalf("Unsupported transport type: %s", transport)
	}
}
