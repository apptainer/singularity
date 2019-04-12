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
	"syscall"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	ociclient "github.com/sylabs/singularity/internal/pkg/client/oci"
	ocitypes "github.com/containers/image/types"
	"github.com/sylabs/singularity/internal/pkg/libexec"
	scs "github.com/sylabs/singularity/internal/pkg/remote"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/uri"
	"github.com/sylabs/singularity/pkg/build/types"
	client "github.com/sylabs/singularity/pkg/client/library"
	"github.com/sylabs/singularity/pkg/signing"
	"github.com/sylabs/singularity/pkg/sypgp"
	"github.com/sylabs/singularity/internal/pkg/build"
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
)

func init() {
	PullCmd.Flags().SetInterspersed(false)

	PullCmd.Flags().StringVar(&PullLibraryURI, "library", "https://library.sylabs.io", "download images from the provided library")
	PullCmd.Flags().SetAnnotation("library", "envkey", []string{"LIBRARY"})

	PullCmd.Flags().BoolVarP(&force, "force", "F", false, "overwrite an image file if it exists")
	PullCmd.Flags().SetAnnotation("force", "envkey", []string{"FORCE"})

	PullCmd.Flags().BoolVarP(&unauthenticatedPull, "allow-unauthenticated", "U", false, "do not require a signed container")
	PullCmd.Flags().SetAnnotation("allow-unauthenticated", "envkey", []string{"ALLOW_UNAUTHENTICATED"})

	PullCmd.Flags().StringVar(&PullImageName, "name", "", "specify a custom image name")
	PullCmd.Flags().Lookup("name").Hidden = true
	PullCmd.Flags().SetAnnotation("name", "envkey", []string{"NAME"})

	PullCmd.Flags().StringVar(&tmpDir, "tmpdir", "", "specify a temporary directory to use for build")
	PullCmd.Flags().Lookup("tmpdir").Hidden = true
	PullCmd.Flags().SetAnnotation("tmpdir", "envkey", []string{"TMPDIR"})

	PullCmd.Flags().BoolVar(&noHTTPS, "nohttps", false, "do NOT use HTTPS with the docker:// transport (useful for local docker registries without a certificate)")
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
			imageSigned, err := signing.IsSigned(name, KeyServerURL, 0, false, authToken, true)
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
			}
		} else {
			sylog.Warningf("Skipping container verification")
		}

	case ShubProtocol:
		libexec.PullShubImage(name, args[i], force, noHTTPS)
	case HTTPProtocol, HTTPSProtocol:
		libexec.PullNetImage(name, args[i], force)
	case ociclient.IsSupported(transport):
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

	fmt.Println("u      : ", args[i])
	fmt.Println("sysCtx : ", sysCtx)

//		imagePath := cache.OciTempImage(libraryImage.Hash, imageName)


		fmt.Println("name     : ", name)
		fmt.Println("args     : ", args[i])

		fmt.Println("tmpDir    : ", tmpDir)
		fmt.Println("force     : ", force)
		fmt.Println("nohttps   : ", noHTTPS)
		fmt.Println("authConf  : ", authConf)
		fmt.Println("noCleanup : ", noCleanUp)


	sum, err := ociclient.ImageSHA(args[i], sysCtx)
	if err != nil {
		sylog.Fatalf("Failed to get SHA of %v: %v", args[i], err)
	}

	name := uri.GetName(args[i])
	imgabs := cache.OciTempImage(sum, name)

		fmt.Println("imgabs: ", imgabs)

		fmt.Println("sum: ", sum)

//	exists, err := cache.OciTempExists(sum, name)
	exists, err := cache.OciTempExists(sum, name)
	if err != nil {
		sylog.Fatalf("Unable to check if %v exists: %v", imgabs, err)
	}
	fmt.Println("exist: ", exists)
	if !exists {
		sylog.Infof("Converting OCI blobs to SIF format")
		b, err := build.NewBuild(
			args[i],
			build.Config{
				Dest:   imgabs,
//				Dest:   name,
				Format: "sif",
				Opts: types.Options{
					TmpDir:           tmpDir,
					NoTest:           true,
					NoHTTPS:          noHTTPS,
					DockerAuthConfig: authConf,
				},
			},
		)
		if err != nil {
			sylog.Fatalf("Unable to create new build: %v", err)
		}

		if err := b.Full(); err != nil {
			sylog.Fatalf("Unable to build: %v", err)
		}
	}


		// Perms are 777 *prior* to umask
		dstFile, err := os.OpenFile(name, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0777)
		if err != nil {
			sylog.Fatalf("Unable to open file: %s: %v\n", name, err)
		}
		defer dstFile.Close()

		srcFile, err := os.OpenFile(imgabs, os.O_RDONLY, 0444)
		if err != nil {
			sylog.Fatalf("Unable to open file: %s: %v\n", imgabs, err)
		}
		defer srcFile.Close()

		// Copy SIF from cache
		_, err = io.Copy(dstFile, srcFile)
		if err != nil {
			sylog.Fatalf("Failed while copying files: %v\n", err)
		}




//		dockerImage, err := client.GetImage(PullLibraryURI, authToken, args[i])
//		if err != nil {
//			sylog.Fatalf("While getting image info: %v", err)
//		}

/*		imagePath := cache.OciTempImage(dockerImage.Hash, imageName)
		exists, err := cache.OciTempExists(dockerImage.Hash, imageName)
		if err != nil {
			sylog.Fatalf("unable to check if %v exists: %v", imagePath, err)
		}
		if !exists {
			sylog.Infof("Downloading docker image...")
//			if err = client.DownloadImage(imagePath, args[i], PullLibraryURI, true, authToken); err != nil {
//				sylog.Fatalf("unable to Download Image: %v", err)
//			}
			libexec.PullOciImage(name, args[i], types.Options{
				TmpDir:           tmpDir,
				Force:            force,
				NoHTTPS:          noHTTPS,
				DockerAuthConfig: authConf,
				NoCleanUp:        noCleanUp,
			})

			if cacheFileHash, err := client.ImageHash(imagePath); err != nil {
				sylog.Fatalf("Error getting ImageHash: %v", err)
			} else if cacheFileHash != dockerImage.Hash {
				sylog.Fatalf("Cached File Hash(%s) and Expected Hash(%s) does not match", cacheFileHash, dockerImage.Hash)
			}
		}*/



/*		libexec.PullOciImage(name, args[i], types.Options{
			TmpDir:           tmpDir,
			Force:            force,
			NoHTTPS:          noHTTPS,
			DockerAuthConfig: authConf,
			NoCleanUp:        noCleanUp,
		})*/

	default:
		sylog.Fatalf("Unsupported transport type: %s", transport)
	}
	// This will exit 1 if the pulled container is signed by
	// a unknown signer, i.e, if you dont have the key in your
	// local keyring. theres proboly a better way to do this...
	os.Exit(exitStat)
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
