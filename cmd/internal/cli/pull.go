// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	ocitypes "github.com/containers/image/types"
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/build"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	ociclient "github.com/sylabs/singularity/internal/pkg/client/oci"
	scs "github.com/sylabs/singularity/internal/pkg/remote"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/uri"
	"github.com/sylabs/singularity/pkg/build/types"
	client "github.com/sylabs/singularity/pkg/client/library"
	net "github.com/sylabs/singularity/pkg/client/net"
	shub "github.com/sylabs/singularity/pkg/client/shub"
	"github.com/sylabs/singularity/pkg/cmdline"
	"github.com/sylabs/singularity/pkg/signing"
	"github.com/sylabs/singularity/pkg/sypgp"
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
	// KeyServerURL server URL
	KeyServerURL = "https://keys.sylabs.io"
	// unauthenticatedPull when true; wont ask to keep a unsigned container after pulling it
	unauthenticatedPull bool
	// PullDir is the path that the containers will be pulled to, if set
	PullDir string
)

// --library
var pullLibraryURIFlag = cmdline.Flag{
	ID:           "pullLibraryURIFlag",
	Value:        &PushLibraryURI,
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
	Value:        &PullImageName,
	DefaultValue: "",
	Name:         "name",
	Hidden:       true,
	Usage:        "specify a custom image name",
	EnvKeys:      []string{"NAME"},
}

// --dir
var pullDirFlag = cmdline.Flag{
	ID:           "pullDirFlag",
	Value:        &PullDir,
	DefaultValue: "",
	Name:         "dir",
	Usage:        "download images to the specific directory",
	EnvKeys:      []string{"PULLDIR", "PULLFOLDER"},
}

//<<<<<<< HEAD
//	PullCmd.Flags().BoolVarP(&unauthenticatedPull, "allow-unsigned", "U", false, "do not require a signed container")
//	PullCmd.Flags().SetAnnotation("allow-unsigned", "envkey", []string{"ALLOW_UNSIGNED"})
//
//	PullCmd.Flags().BoolVarP(&unauthenticatedPull, "allow-unauthenticated", "", false, "do not require a signed container")
//	PullCmd.Flags().Lookup("allow-unauthenticated").Hidden = true
//=======
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

//>>>>>>> upstream/master

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
	cmdManager.RegisterFlagForCmd(&pullDirFlag, PullCmd)

	cmdManager.RegisterFlagForCmd(&actionDockerUsernameFlag, PullCmd)
	cmdManager.RegisterFlagForCmd(&actionDockerPasswordFlag, PullCmd)
	cmdManager.RegisterFlagForCmd(&actionDockerLoginFlag, PullCmd)

	cmdManager.RegisterFlagForCmd(&buildNoCleanupFlag, PullCmd)
	cmdManager.RegisterFlagForCmd(&pullAllowUnsignedFlag, PullCmd)
	cmdManager.RegisterFlagForCmd(&pullAllowUnauthenticatedFlag, PullCmd)
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
	exitStat := 0
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

	if PullDir != "" {
		name = filepath.Join(PullDir, name)
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
				sylog.Fatalf("image file already exists: %q - will not overwrite", name)
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
		exists, err := cache.LibraryImageExists(libraryImage.Hash, imageName)
		if err != nil {
			sylog.Fatalf("unable to check if %v exists: %v", imagePath, err)
		}
		if !exists {
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

		// check if we pulled from the library, if so; is it signed?
		if !unauthenticatedPull {
			imageSigned, err := signing.IsSigned(name, KeyServerURL, 0, false, authToken, false, true)
			if err != nil {
				// err will be: "unable to verify container: %v", err
				sylog.Warningf("%v", err)
				// if theres a warning, exit 1
				exitStat = 1
			}
			// if container is not signed, print a warning
			if !imageSigned {
				fmt.Fprintf(os.Stderr, "This image is not signed, and thus its contents cannot be verified.\n")
				resp, err := sypgp.AskQuestion("Do you with to proceed? [N/y] ")
				if err != nil {
					sylog.Fatalf("unable to parse input: %v", err)
				}
				if resp == "" || resp != "y" && resp != "Y" {
					fmt.Fprintf(os.Stderr, "Aborting.\n")
					err := os.Remove(name)
					if err != nil {
						sylog.Fatalf("Unabel to delete the container: %v", err)
					}
					// exit status 10 after replying no
					exitStat = 10
				}
				fmt.Fprintf(os.Stderr, "You can avoid this by using the '-U' flag\n")
			}
		} else {
			sylog.Warningf("Skipping container verification")
		}
		fmt.Printf("Download complete: %s\n", name)

	case ShubProtocol:
		err := shub.DownloadImage(name, args[i], force, noHTTPS)
		if err != nil {
			sylog.Fatalf("%v\n", err)
		}
	case HTTPProtocol, HTTPSProtocol:
		err := net.DownloadImage(name, args[i], force)
		if err != nil {
			sylog.Fatalf("%v\n", err)
		}
	case ociclient.IsSupported(transport):
		downloadOciImage(name, args[i], cmd)
	default:
		sylog.Fatalf("Unsupported transport type: %s", transport)
	}
	// This will exit 1 if the pulled container is signed by
	// a unknown signer, i.e, if you dont have the key in your
	// local keyring. theres proboly a better way to do this...
	os.Exit(exitStat)
}

// TODO: This should be a external function
func downloadOciImage(name, imageURI string, cmd *cobra.Command) {
	if !force {
		if _, err := os.Stat(name); err == nil {
			sylog.Fatalf("Image file already exists - will not overwrite")
		}
	}

	authConf, err := makeDockerCredentials(cmd)
	if err != nil {
		sylog.Fatalf("While creating Docker credentials: %v", err)
	}

	sysCtx := &ocitypes.SystemContext{
		OCIInsecureSkipTLSVerify:    noHTTPS,
		DockerInsecureSkipTLSVerify: noHTTPS,
		DockerAuthConfig:            authConf,
	}

	sum, err := ociclient.ImageSHA(imageURI, sysCtx)
	if err != nil {
		sylog.Fatalf("Failed to get checksum for %s: %s", imageURI, err)
	}

	imgName := uri.GetName(imageURI)
	cachedImgPath := cache.OciTempImage(sum, imgName)

	exists, err := cache.OciTempExists(sum, imgName)
	if err != nil {
		sylog.Fatalf("Unable to check if %s exists: %s", imgName, err)
	}
	if !exists {
		sylog.Infof("Converting OCI blobs to SIF format")
		if err := convertDockerToSIF(imageURI, cachedImgPath, tmpDir, noHTTPS, authConf); err != nil {
			sylog.Fatalf("%v", err)
		}
	} else {
		sylog.Infof("Using cached image")
	}

	// Perms are 777 *prior* to umask
	dstFile, err := os.OpenFile(name, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0777)
	if err != nil {
		sylog.Fatalf("Unable to open file for writing: %s: %v\n", name, err)
	}
	defer dstFile.Close()

	srcFile, err := os.Open(cachedImgPath)
	if err != nil {
		sylog.Fatalf("Unable to open file for reading: %s: %v\n", name, err)
	}
	defer srcFile.Close()

	// Copy SIF from cache
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		sylog.Fatalf("Failed while copying files: %v\n", err)
	}
}

func convertDockerToSIF(image, cachedImgPath, tmpDir string, noHTTPS bool, authConf *ocitypes.DockerAuthConfig) error {
	b, err := build.NewBuild(
		image,
		build.Config{
			Dest:   cachedImgPath,
			Format: "sif",
			Opts: types.Options{
				TmpDir:           tmpDir,
				NoTest:           true,
				NoHTTPS:          noHTTPS,
				DockerAuthConfig: authConf,
			},
		})
	if err != nil {
		return fmt.Errorf("Unable to create new build: %v", err)
	}

	return b.Full()
}

func handlePullFlags(cmd *cobra.Command) {
	// if we can load config and if default endpoint is set, use that
	// otherwise fall back on regular authtoken and URI behavior
	endpoint, err := sylabsRemote(remoteConfig)
	if err == scs.ErrNoDefault {
		sylog.Warningf("No default remote in use, falling back to: %v", PullLibraryURI)
		sylog.Debugf("using default key server url: %v", KeyServerURL)
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

	uri, err := endpoint.GetServiceURI("keystore")
	if err != nil {
		sylog.Warningf("Unable to get library service URI: %v, defaulting to %s.", err, KeyServerURL)
		return
	}
	KeyServerURL = uri
}
