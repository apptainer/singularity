// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/libexec"
	scs "github.com/sylabs/singularity/internal/pkg/remote"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/uri"
	"github.com/sylabs/singularity/pkg/build/types"
	client "github.com/sylabs/singularity/pkg/client/library"
)

const (
	// LibraryProtocol holds the sylabs cloud library base URI
	// for more info refer to https://cloud.sylabs.io/library
	LibraryProtocol = "library"
	// ShubProtocol holds singularity hub base URI
	// for more info refer to https://singularity-hub.org/
	ShubProtocol = "shub"
	// HTTPProtocol holds the remote http base URI
	HTTPProtocol = "http"
	// HTTPSProtocol holds the remote https base URI
	HTTPSProtocol = "https"
)

var (
	// PullLibraryURI holds the base URI to a Sylabs library API instance
	PullLibraryURI string
	// PullImageName holds the name to be given to the pulled image
	PullImageName string
)

func init() {
	PullCmd.Flags().SetInterspersed(false)

	PullCmd.Flags().StringVar(&PullLibraryURI, "library", "https://library.sylabs.io", "the library to pull from")
	PullCmd.Flags().SetAnnotation("library", "envkey", []string{"LIBRARY"})

	PullCmd.Flags().BoolVarP(&force, "force", "F", false, "overwrite an image file if it exists")
	PullCmd.Flags().SetAnnotation("force", "envkey", []string{"FORCE"})

	PullCmd.Flags().StringVar(&PullImageName, "name", "", "specify a custom image name")
	PullCmd.Flags().Lookup("name").Hidden = true
	PullCmd.Flags().SetAnnotation("name", "envkey", []string{"NAME"})

	PullCmd.Flags().StringVar(&tmpDir, "tmpdir", "", "specify a temporary directory to use for build")
	PullCmd.Flags().Lookup("tmpdir").Hidden = true
	PullCmd.Flags().SetAnnotation("tmpdir", "envkey", []string{"TMPDIR"})

	PullCmd.Flags().BoolVar(&noHTTPS, "nohttps", false, "do NOT use HTTPS, for communicating with local docker registry")
	PullCmd.Flags().SetAnnotation("nohttps", "envkey", []string{"NOHTTPS"})

	PullCmd.Flags().AddFlag(actionFlags.Lookup("docker-username"))
	PullCmd.Flags().AddFlag(actionFlags.Lookup("docker-password"))
	PullCmd.Flags().AddFlag(actionFlags.Lookup("docker-login"))

	PullCmd.Flags().AddFlag(BuildCmd.Flags().Lookup("no-cleanup"))

	SingularityCmd.AddCommand(PullCmd)
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
	i := len(args) - 1 // uri is stored in args[len(args)-1]
	transport, ref := uri.Split(args[i])
	if ref == "" {
		sylog.Fatalf("bad uri %s", args[i])
	}

	var name string
	if PullImageName == "" {
		name = args[0]
		if len(args) == 1 {
			if transport == "" {
				name = uri.GetName("library://" + args[i])
			} else {
				name = uri.GetName(args[i]) // TODO: If not library/shub & no name specified, simply put to cache
			}
		}
	} else {
		name = PullImageName
	}

	// monitor for OS signals and remove invalid file
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func(fileName string) {
		<-c
		sylog.Debugf("Removing incomplete file because of receiving Termination signal")
		os.Remove(fileName)
		os.Exit(1)
	}(name)

	switch transport {
	case LibraryProtocol, "":
		if !force {
			if _, err := os.Stat(name); err == nil {
				sylog.Fatalf("image file already exists - will not overwrite")
			}
		}

		handlePullFlags(cmd)

		libraryImage, err := client.GetImage(PullLibraryURI, authToken, args[i])
		if err != nil {
			sylog.Fatalf("While getting image info: %v", err)
		}

		var imageName string
		if transport == "" {
			imageName = uri.GetName("library://" + args[i])
		} else {
			imageName = uri.GetName(args[i])
		}
		imagePath := cache.LibraryImage(libraryImage.Hash, imageName)
		if exists, err := cache.LibraryImageExists(libraryImage.Hash, imageName); err != nil {
			sylog.Fatalf("unable to check if %v exists: %v", imagePath, err)
		} else if !exists {
			sylog.Infof("Downloading library image")
			if err = client.DownloadImage(imagePath, args[i], PullLibraryURI, true, authToken); err != nil {
				sylog.Fatalf("unable to Download Image: %v", err)
			}

			if cacheFileHash, err := client.ImageHash(imagePath); err != nil {
				sylog.Fatalf("Error getting ImageHash: %v", err)
			} else if cacheFileHash != libraryImage.Hash {
				sylog.Fatalf("Cached File Hash(%s) and Expected Hash(%s) does not match", cacheFileHash, libraryImage.Hash)
			}
		}

		// Perms are 777 *prior* to umask
		dstFile, err := os.OpenFile(name, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0777)
		if err != nil {
			sylog.Fatalf("%v\n", err)
		}
		defer dstFile.Close()

		srcFile, err := os.OpenFile(imagePath, os.O_RDONLY, 0444)
		if err != nil {
			sylog.Fatalf("%v\n", err)
		}
		defer srcFile.Close()

		// Copy SIF from cache
		_, err = io.Copy(dstFile, srcFile)
		if err != nil {
			sylog.Fatalf("%v\n", err)
		}
	case ShubProtocol:
		libexec.PullShubImage(name, args[i], force, noHTTPS)
	case HTTPProtocol, HTTPSProtocol:
		libexec.PullNetImage(name, args[i], force)
	default:
		if !force {
			if _, err := os.Stat(name); err == nil {
				sylog.Fatalf("image file already exists - will not overwrite")
			}
		}

		authConf, err := makeDockerCredentials(cmd)
		if err != nil {
			sylog.Fatalf("While creating Docker credentials: %v", err)
		}

		libexec.PullOciImage(name, args[i], types.Options{
			TmpDir:           tmpDir,
			Force:            force,
			NoHTTPS:          noHTTPS,
			DockerAuthConfig: authConf,
			NoCleanUp:        noCleanUp,
		})
	}
}

func handlePullFlags(cmd *cobra.Command) {
	// if we can load config and if default endpoint is set, use that
	// otherwise fall back on regular authtoken and URI behavior
	endpoint, err := sylabsRemote(remoteConfig)
	if err == scs.ErrNoDefault {
		sylog.Warningf("No default remote in use, falling back to: %v", PullLibraryURI)
		return
	} else if err != nil {
		sylog.Fatalf("Unable to load remote configuration: %v", err)
	}

	authToken = endpoint.Token
	if !cmd.Flags().Lookup("library").Changed {
		uri, err := endpoint.GetServiceURI("library")
		if err != nil {
			sylog.Fatalf("Unable to get library service URI: %v", err)
		}
		PullLibraryURI = uri
	}
}
