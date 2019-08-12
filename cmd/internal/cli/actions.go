// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	ocitypes "github.com/containers/image/types"
	"github.com/spf13/cobra"
	library "github.com/sylabs/scs-library-client/client"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/build"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	ociclient "github.com/sylabs/singularity/internal/pkg/client/oci"
	libraryhelper "github.com/sylabs/singularity/internal/pkg/library"
	"github.com/sylabs/singularity/internal/pkg/oras"
	scs "github.com/sylabs/singularity/internal/pkg/remote"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/uri"
	"github.com/sylabs/singularity/pkg/build/types"
	net "github.com/sylabs/singularity/pkg/client/net"
	shub "github.com/sylabs/singularity/pkg/client/shub"
)

const (
	defaultPath = "/bin:/usr/bin:/sbin:/usr/sbin:/usr/local/bin:/usr/local/sbin"
)

func getCacheHandle() *cache.Handle {
	h, err := cache.NewHandle(os.Getenv(cache.DirEnv))
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
	os.Setenv("PATH", defaultPath)

	// create an handle for the current image cache
	imgCache := getCacheHandle()
	if imgCache == nil {
		sylog.Fatalf("failed to create a new image cache handle")
	}

	replaceURIWithImage(imgCache, cmd, args)
}

func handleOCI(imgCache *cache.Handle, cmd *cobra.Command, u string) (string, error) {
	authConf, err := makeDockerCredentials(cmd)
	if err != nil {
		sylog.Fatalf("While creating Docker credentials: %v", err)
	}

	sysCtx := &ocitypes.SystemContext{
		OCIInsecureSkipTLSVerify:    noHTTPS,
		DockerInsecureSkipTLSVerify: noHTTPS,
		DockerAuthConfig:            authConf,
	}

	imgabs := ""
	name := uri.GetName(u)

	if disableCache {
		sylog.Infof("Converting OCI blobs to SIF format")
		var err error
		imgabs, err = ioutil.TempDir(tmpDir, "sbuild-tmp-cache-")
		if err != nil {
			return "", fmt.Errorf("unable to create tmp file: %v", err)
		}

		b, err := build.NewBuild(
			u,
			build.Config{
				Dest:   imgabs,
				Format: "sif",
				Opts: types.Options{
					ImgCache:         imgCache,
					TmpDir:           tmpDir,
					NoCache:          true,
					NoTest:           true,
					NoHTTPS:          noHTTPS,
					DockerAuthConfig: authConf,
				},
			})

		if err != nil {
			return "", fmt.Errorf("unable to create new build: %v", err)
		}

		if err := b.Full(); err != nil {
			return "", fmt.Errorf("unable to build: %v", err)
		}

	} else {
		sum, err := ociclient.ImageSHA(u, sysCtx)
		if err != nil {
			return "", fmt.Errorf("failed to get SHA of %v: %v", u, err)
		}
		imgabs = imgCache.OciTempImage(sum, name)

		exists, err := imgCache.OciTempExists(sum, name)
		if err != nil {
			return "", fmt.Errorf("unable to check if %s exists: %s", name, err)
		}
		if !exists {
			sylog.Infof("Converting OCI blobs to SIF format")
			b, err := build.NewBuild(
				u,
				build.Config{
					Dest:   imgabs,
					Format: "sif",
					Opts: types.Options{
						TmpDir:           tmpDir,
						NoTest:           true,
						NoHTTPS:          noHTTPS,
						DockerAuthConfig: authConf,
						ImgCache:         imgCache,
					},
				})
			if err != nil {
				return "", fmt.Errorf("unable to create new build: %v", err)
			}

			if err := b.Full(); err != nil {
				return "", fmt.Errorf("unable to build: %v", err)
			}

			sylog.Verbosef("Image cached as SIF at %s", imgabs)
		}
	}

	return imgabs, nil
}

func handleOras(imgCache *cache.Handle, cmd *cobra.Command, u string) (string, error) {
	ociAuth, err := makeDockerCredentials(cmd)
	if err != nil {
		return "", fmt.Errorf("while creating docker credentials: %v", err)
	}

	_, ref := uri.Split(u)
	sum, err := oras.ImageSHA(ref, ociAuth)
	if err != nil {
		return "", fmt.Errorf("failed to get SHA of %v: %v", u, err)
	}

	imageName := uri.GetName(u)
	cacheImagePath := imgCache.OrasImage(sum, imageName)
	if exists, err := imgCache.OrasImageExists(sum, imageName); err != nil {
		return "", fmt.Errorf("unable to check if %v exists: %v", cacheImagePath, err)
	} else if !exists {
		sylog.Infof("Downloading image with ORAS")

		if err := oras.DownloadImage(cacheImagePath, ref, ociAuth); err != nil {
			return "", fmt.Errorf("unable to Download Image: %v", err)
		}

		if cacheFileHash, err := oras.ImageHash(cacheImagePath); err != nil {
			return "", fmt.Errorf("error getting ImageHash: %v", err)
		} else if cacheFileHash != sum {
			return "", fmt.Errorf("cached file hash(%s) and expected hash(%s) does not match", cacheFileHash, sum)
		}
	}

	return cacheImagePath, nil
}

func handleLibrary(imgCache *cache.Handle, u, libraryURL string) (string, error) {
	ctx := context.TODO()

	c, err := library.NewClient(&library.Config{
		AuthToken: authToken,
		BaseURL:   libraryURL,
	})
	if err != nil {
		return "", fmt.Errorf("unable to initialize client library: %v", err)
	}

	imageRef := libraryhelper.NormalizeLibraryRef(u)

	libraryImage, err := c.GetImage(ctx, imageRef)
	if err == library.ErrNotFound {
		return "", fmt.Errorf("image does not exist in the library: %s", imageRef)
	}
	if err != nil {
		return "", err
	}

	imagePath := ""
	if disableCache {
		file, err := ioutil.TempFile(tmpDir, "sbuild-tmp-cache-")
		if err != nil {
			return "", fmt.Errorf("unable to create tmp file: %v", err)
		}
		imagePath = file.Name()
		sylog.Infof("Downloading library image to tmp cache: %s", imagePath)

		if err = libraryhelper.DownloadImageNoProgress(ctx, c, imagePath, imageRef); err != nil {
			return "", fmt.Errorf("unable to download image: %v", err)
		}

	} else {
		imageName := uri.GetName("library://" + imageRef)
		imagePath = imgCache.LibraryImage(libraryImage.Hash, imageName)

		if exists, err := imgCache.LibraryImageExists(libraryImage.Hash, imageName); err != nil {
			return "", fmt.Errorf("unable to check if %v exists: %v", imagePath, err)
		} else if !exists {
			sylog.Infof("Downloading library image")

			if err := libraryhelper.DownloadImageNoProgress(ctx, c, imagePath, imageRef); err != nil {
				return "", fmt.Errorf("unable to download image: %v", err)
			}

			if cacheFileHash, err := library.ImageHash(imagePath); err != nil {
				return "", fmt.Errorf("error getting image hash: %v", err)
			} else if cacheFileHash != libraryImage.Hash {
				return "", fmt.Errorf("cached file hash(%s) and expected hash(%s) does not match", cacheFileHash, libraryImage.Hash)
			}
		}
	}

	return imagePath, nil
}

func handleShub(imgCache *cache.Handle, u string) (string, error) {
	imagePath := ""

	shubURI, err := shub.ShubParseReference(u)
	if err != nil {
		return "", fmt.Errorf("failed to parse shub uri: %s", err)
	}

	// Get the image manifest
	manifest, err := shub.GetManifest(shubURI, noHTTPS)
	if err != nil {
		return "", fmt.Errorf("failed to get manifest for: %s: %s", u, err)
	}

	if disableCache {
		file, err := ioutil.TempFile(tmpDir, "sbuild-tmp-cache-")
		if err != nil {
			return "", fmt.Errorf("unable to create tmp file: %v", err)
		}
		imagePath = file.Name()

		sylog.Infof("Downloading shub image")
		err = shub.DownloadImage(manifest, imagePath, u, true, noHTTPS)
		if err != nil {
			sylog.Fatalf("%v\n", err)
		}
	} else {
		imageName := uri.GetName(u)
		imagePath = imgCache.ShubImage(manifest.Commit, imageName)

		exists, err := imgCache.ShubImageExists(manifest.Commit, imageName)
		if err != nil {
			return "", fmt.Errorf("unable to check if %v exists: %v", imagePath, err)
		}
		if !exists {
			sylog.Infof("Downloading shub image")
			err := shub.DownloadImage(manifest, imagePath, u, true, noHTTPS)
			if err != nil {
				sylog.Fatalf("%v\n", err)
			}
		} else {
			sylog.Verbosef("Use image from cache")
		}
	}

	return imagePath, nil
}

func handleNet(imgCache *cache.Handle, u string) (string, error) {
	refParts := strings.Split(u, "/")
	imageName := refParts[len(refParts)-1]
	imagePath := imgCache.NetImage("hash", imageName)

	exists, err := imgCache.NetImageExists("hash", imageName)
	if err != nil {
		return "", fmt.Errorf("unable to check if %v exists: %v", imagePath, err)
	}
	if !exists {
		sylog.Infof("Downloading network image")
		err := net.DownloadImage(imagePath, u)
		if err != nil {
			sylog.Fatalf("%v\n", err)
		}
	} else {
		sylog.Verbosef("Use image from cache")
	}

	return imagePath, nil
}

func replaceURIWithImage(imgCache *cache.Handle, cmd *cobra.Command, args []string) {
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

		image, err = handleLibrary(imgCache, args[0], handleActionRemote(cmd))
	case uri.Oras:
		image, err = handleOras(imgCache, cmd, args[0])
	case uri.Shub:
		image, err = handleShub(imgCache, args[0])
	case ociclient.IsSupported(t):
		image, err = handleOCI(imgCache, cmd, args[0])
	case uri.HTTP:
		image, err = handleNet(imgCache, args[0])
	case uri.HTTPS:
		image, err = handleNet(imgCache, args[0])
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
