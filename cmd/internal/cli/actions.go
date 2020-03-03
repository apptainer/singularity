// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	ocitypes "github.com/containers/image/v5/types"
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
	os.Setenv("PATH", defaultPath)

	// create an handle for the current image cache
	imgCache := getCacheHandle(cache.Config{Disable: disableCache})
	if imgCache == nil {
		sylog.Fatalf("failed to create a new image cache handle")
	}

	ctx := context.TODO()

	replaceURIWithImage(ctx, imgCache, cmd, args)
}

func handleOCI(ctx context.Context, imgCache *cache.Handle, cmd *cobra.Command, u string) (string, error) {
	authConf, err := makeDockerCredentials(cmd)
	if err != nil {
		sylog.Fatalf("While creating Docker credentials: %v", err)
	}

	sysCtx := &ocitypes.SystemContext{
		OCIInsecureSkipTLSVerify:    noHTTPS,
		DockerInsecureSkipTLSVerify: ocitypes.NewOptionalBool(noHTTPS),
		DockerAuthConfig:            authConf,
	}

	imagePath := ""

	if disableCache {
		sylog.Infof("Converting OCI blobs to SIF format")
		var err error
		imagePath, err = ioutil.TempDir(tmpDir, "sbuild-tmp-cache-")
		if err != nil {
			return "", fmt.Errorf("unable to create tmp file: %v", err)
		}

		b, err := build.NewBuild(
			u,
			build.Config{
				Dest:   imagePath,
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

		if err := b.Full(ctx); err != nil {
			return "", fmt.Errorf("unable to build: %v", err)
		}

	} else {
		hash, err := ociclient.ImageSHA(ctx, u, sysCtx)
		if err != nil {
			return "", fmt.Errorf("failed to get SHA of %v: %v", u, err)
		}

		cacheEntry, err := imgCache.GetEntry(cache.OciTempCacheType, hash)
		if err != nil {
			return "", fmt.Errorf("unable to check if %v exists in cache: %v", hash, err)
		}

		if !cacheEntry.Exists {
			sylog.Infof("Converting OCI blobs to SIF format")
			b, err := build.NewBuild(
				u,
				build.Config{
					Dest:   cacheEntry.TmpPath,
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

			if err := b.Full(ctx); err != nil {
				return "", fmt.Errorf("unable to build: %v", err)
			}

			err = cacheEntry.Finalize()
			if err != nil {
				return "", err
			}

			sylog.Verbosef("Image cached as SIF at %s", cacheEntry.Path)

		} else {
			sylog.Infof("Using cached image")
		}

		imagePath = cacheEntry.Path
	}

	return imagePath, nil
}

func handleOras(ctx context.Context, imgCache *cache.Handle, cmd *cobra.Command, u string) (string, error) {
	ociAuth, err := makeDockerCredentials(cmd)
	if err != nil {
		return "", fmt.Errorf("while creating docker credentials: %v", err)
	}

	_, ref := uri.Split(u)
	hash, err := oras.ImageSHA(ctx, ref, ociAuth)
	if err != nil {
		return "", fmt.Errorf("failed to get SHA of %v: %v", u, err)
	}

	cacheEntry, err := imgCache.GetEntry(cache.OrasCacheType, hash)
	if err != nil {
		return "", fmt.Errorf("unable to check if %v exists in cache: %v", hash, err)
	}
	if !cacheEntry.Exists {
		sylog.Infof("Downloading oras image")

		if err := oras.DownloadImage(cacheEntry.TmpPath, ref, ociAuth); err != nil {
			return "", fmt.Errorf("unable to Download Image: %v", err)
		}
		if cacheFileHash, err := oras.ImageHash(cacheEntry.TmpPath); err != nil {
			return "", fmt.Errorf("error getting ImageHash: %v", err)
		} else if cacheFileHash != hash {
			_ = cacheEntry.Abort()
			return "", fmt.Errorf("cached file hash(%s) and expected hash(%s) does not match", cacheFileHash, hash)
		}

		err = cacheEntry.Finalize()
		if err != nil {
			return "", err
		}

	} else {
		sylog.Infof("Using cached image")

	}

	return cacheEntry.Path, nil
}

func handleLibrary(ctx context.Context, imgCache *cache.Handle, u, libraryURL string) (string, error) {
	c, err := library.NewClient(&library.Config{
		AuthToken: authToken,
		BaseURL:   libraryURL,
	})
	if err != nil {
		return "", fmt.Errorf("unable to initialize client library: %v", err)
	}

	imageRef := libraryhelper.NormalizeLibraryRef(u)

	libraryImage, err := c.GetImage(ctx, runtime.GOARCH, imageRef)
	if err == library.ErrNotFound {
		return "", fmt.Errorf("image does not exist in the library: %s (%s)", imageRef, runtime.GOARCH)
	}
	if err != nil {
		return "", err
	}

	imagePath := ""
	if imgCache.IsDisabled() {
		file, err := ioutil.TempFile(tmpDir, "sbuild-tmp-cache-")
		if err != nil {
			return "", fmt.Errorf("unable to create tmp file: %v", err)
		}
		imagePath = file.Name()
		sylog.Infof("Downloading library image to tmp cache: %s", imagePath)

		if err = libraryhelper.DownloadImageNoProgress(ctx, c, imagePath, runtime.GOARCH, imageRef); err != nil {
			return "", fmt.Errorf("unable to download image: %v", err)
		}

	} else {
		cacheEntry, err :=imgCache.GetEntry(cache.LibraryCacheType, libraryImage.Hash)
		if err != nil {
			return "", fmt.Errorf("unable to check if %v exists in cache: %v", libraryImage.Hash, err)
		}
		if !cacheEntry.Exists {
			sylog.Infof("Downloading library image")

			if err := libraryhelper.DownloadImageNoProgress(ctx, c, cacheEntry.TmpPath, runtime.GOARCH, imageRef); err != nil {
				return "", fmt.Errorf("unable to download image: %v", err)
			}

			if cacheFileHash, err := library.ImageHash(cacheEntry.TmpPath); err != nil {
				return "", fmt.Errorf("error getting image hash: %v", err)
			} else if cacheFileHash != libraryImage.Hash {
				return "", fmt.Errorf("cached file hash(%s) and expected hash(%s) does not match", cacheFileHash, libraryImage.Hash)
			}

			err = cacheEntry.Finalize()
			if err != nil {
				return "", err
			}

		} else {
			sylog.Infof("Using cached image")
		}
		imagePath = cacheEntry.Path
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
		cacheEntry, err := imgCache.GetEntry(cache.ShubCacheType, manifest.Commit)
		if err != nil {
			return "", fmt.Errorf("unable to check if %v exists in cache: %v", manifest.Commit, err)
		}
		if !cacheEntry.Exists {
			sylog.Infof("Downloading shub image")

			err := shub.DownloadImage(manifest, cacheEntry.TmpPath, u, true, noHTTPS)
			if err != nil {
				return "", err
			}

			err = cacheEntry.Finalize()
			if err != nil {
				return "", err
			}

		} else {
			sylog.Infof("Use image from cache")
		}
		imagePath = cacheEntry.Path
	}

	return imagePath, nil
}


func handleNet(imgCache *cache.Handle, u string) (string, error) {
	// We will cache using a sha256 over the URL and the date of the file that
	// is to be fetched, as returned by an HTTP HEAD call and the Last-Modified
	// header. If no date is available, use the current date-time, which will
	// effectively result in no caching.
	imageDate := time.Now().String()

	req, err := http.NewRequest("HEAD", u, nil)
	if err != nil {
		sylog.Fatalf("Error constructing http request: %v\n", err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		sylog.Fatalf("Error making http request: %v\n", err)
	}

	headerDate := res.Header.Get("Last-Modified")
	sylog.Debugf("HTTP Last-Modified header is: %s", headerDate)
	if headerDate != "" {
		imageDate = headerDate
	}

	h := sha256.New()
	h.Write([]byte(u + imageDate))
	hash := hex.EncodeToString(h.Sum(nil))
	sylog.Debugf("Image hash for cache is: %s", hash)

	cacheEntry, err  := imgCache.GetEntry(cache.NetCacheType, hash)
	if err != nil {
		return "", fmt.Errorf("unable to check if %v exists in cache: %v", hash, err)
	}

	if !cacheEntry.Exists {
		sylog.Infof("Downloading network image")
		err := net.DownloadImage(cacheEntry.TmpPath, u)
		if err != nil {
			sylog.Fatalf("%v\n", err)
		}

		err = cacheEntry.Finalize()
		if err != nil {
			return "", err
		}

	} else {
		sylog.Verbosef("Using image from cache")
	}

	return cacheEntry.Path, nil
}

func replaceURIWithImage(ctx context.Context, imgCache *cache.Handle, cmd *cobra.Command, args []string) {
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

		image, err = handleLibrary(ctx, imgCache, args[0], handleActionRemote(cmd))
	case uri.Oras:
		image, err = handleOras(ctx, imgCache, cmd, args[0])
	case uri.Shub:
		image, err = handleShub(imgCache, args[0])
	case ociclient.IsSupported(t):
		image, err = handleOCI(ctx, imgCache, cmd, args[0])
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
