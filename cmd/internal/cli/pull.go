// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/sylabs/scs-library-client/client"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/app/singularity"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	ociclient "github.com/sylabs/singularity/internal/pkg/client/oci"
	scs "github.com/sylabs/singularity/internal/pkg/remote"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/uri"
	net "github.com/sylabs/singularity/pkg/client/net"
	"github.com/sylabs/singularity/pkg/cmdline"
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
	// keyServerURL server URL.
	keyServerURL = "https://keys.sylabs.io"
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
	DefaultValue: "https://library.sylabs.io",
	Name:         "library",
	Usage:        "download images from the provided library",
	EnvKeys:      []string{"LIBRARY"},
}

// -F|--force
var pullForceFlag = cmdline.Flag{
	ID:           "pullForceFlag",
	Value:        &force,
	DefaultValue: false,
	Name:         "force",
	ShortHand:    "F",
	Usage:        "overwrite an image file if it exists",
	EnvKeys:      []string{"FORCE"},
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

// --tmpdir
var pullTmpdirFlag = cmdline.Flag{
	ID:           "pullTmpdirFlag",
	Value:        &tmpDir,
	DefaultValue: "",
	Hidden:       true,
	Name:         "tmpdir",
	Usage:        "specify a temporary directory to use for build",
	EnvKeys:      []string{"TMPDIR"},
}

// --nohttps
var pullNoHTTPSFlag = cmdline.Flag{
	ID:           "pullNoHTTPSFlag",
	Value:        &noHTTPS,
	DefaultValue: false,
	Name:         "nohttps",
	Usage:        "do NOT use HTTPS with the docker:// transport (useful for local docker registries without a certificate)",
	EnvKeys:      []string{"NOHTTPS"},
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
	cmdManager.RegisterCmd(PullCmd)

	cmdManager.RegisterFlagForCmd(&pullForceFlag, PullCmd)
	cmdManager.RegisterFlagForCmd(&pullLibraryURIFlag, PullCmd)
	cmdManager.RegisterFlagForCmd(&pullNameFlag, PullCmd)
	cmdManager.RegisterFlagForCmd(&pullNoHTTPSFlag, PullCmd)
	cmdManager.RegisterFlagForCmd(&pullTmpdirFlag, PullCmd)
	cmdManager.RegisterFlagForCmd(&pullDisableCacheFlag, PullCmd)
	cmdManager.RegisterFlagForCmd(&pullDirFlag, PullCmd)

	cmdManager.RegisterFlagForCmd(&actionDockerUsernameFlag, PullCmd)
	cmdManager.RegisterFlagForCmd(&actionDockerPasswordFlag, PullCmd)
	cmdManager.RegisterFlagForCmd(&actionDockerLoginFlag, PullCmd)

	cmdManager.RegisterFlagForCmd(&buildNoCleanupFlag, PullCmd)
	cmdManager.RegisterFlagForCmd(&pullAllowUnsignedFlag, PullCmd)
	cmdManager.RegisterFlagForCmd(&pullAllowUnauthenticatedFlag, PullCmd)
	cmdManager.RegisterFlagForCmd(&pullArchFlag, PullCmd)
}

// PullCmd singularity pull
var PullCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args:                  cobra.RangeArgs(1, 2),
	PreRun:                sylabsToken,
	Run:                   pullRun,
	Use:                   docs.PullUse,
	Short:                 docs.PullShort,
	Long:                  docs.PullLong,
	Example:               docs.PullExample,
}

func pullRun(cmd *cobra.Command, args []string) {
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
		if !force {
			sylog.Fatalf("Image file already exists: %q - will not overwrite", pullTo)
		}
		sylog.Debugf("Removing overridden file: %s", pullTo)
		if err := os.Remove(pullTo); err != nil {
			sylog.Fatalf("Unable to remove %q: %s", pullTo, err)
		}
	}

	// monitor for OS signals and remove invalid file
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func(fileName string) {
		<-c
		sylog.Debugf("Removing incomplete file because of receiving Termination signal")
		os.Remove(fileName)
		os.Exit(1)
	}(pullTo)

	switch transport {
	case LibraryProtocol, "":
		handlePullFlags(cmd)

		libraryConfig := &client.Config{
			BaseURL:   pullLibraryURI,
			AuthToken: authToken,
		}
		lib, err := singularity.NewLibrary(libraryConfig, imgCache, keyServerURL)
		if err != nil {
			sylog.Fatalf("Could not initialize library: %v", err)
		}

		err = lib.Pull(context.TODO(), pullFrom, pullTo, pullArch)
		if err != nil && err != singularity.ErrLibraryPullUnsigned {
			sylog.Fatalf("While pulling library image: %v", err)
		}
		if err == singularity.ErrLibraryPullUnsigned {
			sylog.Warningf("Skipping container verification")
		}
	case ShubProtocol:
		err := singularity.PullShub(imgCache, pullTo, pullFrom, noHTTPS)
		if err != nil {
			sylog.Fatalf("While pulling shub image: %v\n", err)
		}
	case OrasProtocol:
		ociAuth, err := makeDockerCredentials(cmd)
		if err != nil {
			sylog.Fatalf("Unable to make docker oci credentials: %s", err)
		}

		err = singularity.OrasPull(imgCache, pullTo, ref, force, ociAuth)
		if err != nil {
			sylog.Fatalf("While pulling image from oci registry: %v", err)
		}
	case HTTPProtocol, HTTPSProtocol:
		err := net.DownloadImage(pullTo, pullFrom)
		if err != nil {
			sylog.Fatalf("While pulling from image from http(s): %v\n", err)
		}
	case ociclient.IsSupported(transport):
		ociAuth, err := makeDockerCredentials(cmd)
		if err != nil {
			sylog.Fatalf("While creating Docker credentials: %v", err)
		}

		err = singularity.OciPull(imgCache, pullTo, pullFrom, tmpDir, ociAuth, noHTTPS)
		if err != nil {
			sylog.Fatalf("While making image from oci registry: %v", err)
		}
	default:
		sylog.Fatalf("Unsupported transport type: %s", transport)
	}
}

func handlePullFlags(cmd *cobra.Command) {
	// if we can load config and if default endpoint is set, use that
	// otherwise fall back on regular authtoken and URI behavior
	endpoint, err := sylabsRemote(remoteConfig)
	if err == scs.ErrNoDefault {
		sylog.Warningf("No default remote in use, falling back to: %v", pullLibraryURI)
		sylog.Debugf("Using default key server url: %v", keyServerURL)
		return
	}
	if err != nil {
		sylog.Fatalf("Unable to load remote configuration: %v", err)
	}

	authToken = endpoint.Token
	if !cmd.Flags().Lookup("library").Changed {
		libraryURI, err := endpoint.GetServiceURI("library")
		if err != nil {
			sylog.Fatalf("Unable to get library service URI: %v", err)
		}
		pullLibraryURI = libraryURI
	}

	keystoreURI, err := endpoint.GetServiceURI("keystore")
	if err != nil {
		sylog.Warningf("Unable to get library service URI: %v, defaulting to %s", err, keyServerURL)
		return
	}
	keyServerURL = keystoreURI
}
