// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	ocitypes "github.com/containers/image/types"
	"github.com/sylabs/scs-library-client/client"
	"github.com/sylabs/singularity/internal/pkg/build"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	ociclient "github.com/sylabs/singularity/internal/pkg/client/oci"
	"github.com/sylabs/singularity/internal/pkg/library"
	"github.com/sylabs/singularity/internal/pkg/oras"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/uri"
	"github.com/sylabs/singularity/pkg/build/types"
	shub "github.com/sylabs/singularity/pkg/client/shub"
	"github.com/sylabs/singularity/pkg/signing"
	pb "gopkg.in/cheggaaa/pb.v1"
)

var (
	// ErrLibraryPullUnsigned indicates that the interactive portion of the pull was aborted.
	ErrLibraryPullUnsigned = errors.New("failed to verify container")
)

// LibraryPull will download the image specified by file from the library specified by libraryURI.
// After downloading, the image will be checked for a valid signature and removed if it does not contain one,
// unless specified not to by the unauthenticated bool
func LibraryPull(imgCache *cache.Handle, name, fullURI, libraryURI, keyServerURL, authToken string, unauthenticated, noCache bool) error {
	libraryClient, err := client.NewClient(&client.Config{
		BaseURL:   libraryURI,
		AuthToken: authToken,
	})
	if err != nil {
		return fmt.Errorf("error initializing library client: %v", err)
	}

	// strip leading "library://" and append default tag, as necessary
	imageRef := library.NormalizeLibraryRef(fullURI)

	// check if image exists in library
	libraryImage, err := libraryClient.GetImage(context.TODO(), imageRef)
	if err == client.ErrNotFound {
		return fmt.Errorf("image does not exist in the library: %s", imageRef)
	}
	if err != nil {
		return fmt.Errorf("could not get image info: %v", err)
	}

	if noCache {
		// don't use cached image
		sylog.Infof("Downloading library image: %s", name)
		err := library.DownloadImage(context.TODO(), libraryClient, name, imageRef, downloadImageCallback)
		if err != nil {
			return fmt.Errorf("unable to download image: %v", err)
		}
	} else {
		// check and use cached image
		imageName := uri.GetName("library://" + imageRef)
		imagePath := imgCache.LibraryImage(libraryImage.Hash, imageName)
		exists, err := imgCache.LibraryImageExists(libraryImage.Hash, imageName)
		if err != nil {
			return fmt.Errorf("unable to check if %s exists: %v", imagePath, err)
		}
		if !exists {
			sylog.Infof("Downloading library image")
			go interruptCleanup(imagePath)

			// call library download image helper
			err := library.DownloadImage(context.TODO(), libraryClient, imagePath, imageRef, downloadImageCallback)
			if err != nil {
				return fmt.Errorf("unable to download image: %v", err)
			}

			if cacheFileHash, err := client.ImageHash(imagePath); err != nil {
				return fmt.Errorf("error getting image hash: %v", err)
			} else if cacheFileHash != libraryImage.Hash {
				return fmt.Errorf("cached file hash(%s) and expected hash(%s) does not match", cacheFileHash, libraryImage.Hash)
			}
		}

		// Perms are 777 *prior* to umask in order to allow image to be
		// executed with its leading shebang like a script
		dstFile, err := os.OpenFile(name, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0777)
		if err != nil {
			return fmt.Errorf("while opening destination file: %v", err)
		}
		defer dstFile.Close()

		srcFile, err := os.Open(imagePath)
		if err != nil {
			return fmt.Errorf("while opening cached image: %v", err)
		}
		defer srcFile.Close()

		// Copy SIF from cache
		_, err = io.Copy(dstFile, srcFile)
		if err != nil {
			return fmt.Errorf("while copying image from cache: %v", err)
		}
	}

	// check if we pulled from the library, if so; is it signed?
	if !unauthenticated {
		imageSigned, err := signing.IsSigned(name, keyServerURL, 0, false, authToken)
		if err != nil {
			sylog.Warningf("%v", err)
		}
		if !imageSigned {
			return ErrLibraryPullUnsigned
		}
	} else {
		sylog.Warningf("Skipping container verification")
	}

	sylog.Infof("Download complete: %s\n", name)

	return nil
}

// PullShub will download a image from shub, and cache it. Next time
// that container is downloaded this will just use that cached image.
func PullShub(imgCache *cache.Handle, filePath string, shubRef string, noHTTPS, noCache bool) (err error) {
	shubURI, err := shub.ShubParseReference(shubRef)
	if err != nil {
		return fmt.Errorf("failed to parse shub uri: %s", err)
	}

	// Get the image manifest
	manifest, err := shub.GetManifest(shubURI, noHTTPS)
	if err != nil {
		return fmt.Errorf("failed to get manifest for: %s: %s", shubRef, err)
	}

	imageName := uri.GetName(shubRef)
	imagePath := imgCache.ShubImage(manifest.Commit, imageName)

	if noCache {
		// Dont use cached image
		if err := shub.DownloadImage(manifest, filePath, shubRef, true, noHTTPS); err != nil {
			return err
		}
	} else {
		exists, err := imgCache.ShubImageExists(manifest.Commit, imageName)
		if err != nil {
			return fmt.Errorf("unable to check if %v exists: %v", imagePath, err)
		}
		if !exists {
			sylog.Infof("Downloading shub image")
			go interruptCleanup(imagePath)

			err := shub.DownloadImage(manifest, imagePath, shubRef, true, noHTTPS)
			if err != nil {
				return err
			}
		} else {
			sylog.Infof("Use image from cache")
		}

		// Perms are 777 *prior* to umask in order to allow image to be
		// executed with its leading shebang like a script
		dstFile, err := os.OpenFile(filePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0777)
		if err != nil {
			return fmt.Errorf("while opening destination file: %v", err)
		}
		defer dstFile.Close()

		srcFile, err := os.Open(imagePath)
		if err != nil {
			return fmt.Errorf("while opening cached image: %v", err)
		}
		defer srcFile.Close()

		// Copy image from cache
		_, err = io.Copy(dstFile, srcFile)
		if err != nil {
			return fmt.Errorf("while copying image from cache: %v", err)
		}
	}

	return nil
}

// downloadImageCallback is called to display progress bar while downloading
// image from library
func downloadImageCallback(totalSize int64, r io.Reader, w io.Writer) error {
	bar := pb.New64(totalSize).SetUnits(pb.U_BYTES)
	bar.ShowTimeLeft = true
	bar.ShowSpeed = true

	// create proxy reader
	bodyProgress := bar.NewProxyReader(r)

	bar.Start()

	// Write the body to file
	_, err := io.Copy(w, bodyProgress)
	if err != nil {
		return err
	}

	bar.Finish()

	return nil
}

// OrasPull will download the image specified by the provided oci reference and store
// it at the location specified by file, it will use credentials if supplied
func OrasPull(imgCache *cache.Handle, name, ref string, force bool, ociAuth *ocitypes.DockerAuthConfig) error {
	sum, err := oras.ImageSHA(ref, ociAuth)
	if err != nil {
		return fmt.Errorf("failed to get checksum for %s: %s", ref, err)
	}

	imageName := uri.GetName("oras:" + ref)

	cacheImagePath := imgCache.OrasImage(sum, imageName)
	exists, err := imgCache.OrasImageExists(sum, imageName)
	if err == cache.ErrBadChecksum {
		sylog.Warningf("Removing cached image: %s: cache could be corrupted", cacheImagePath)
		err := os.Remove(cacheImagePath)
		if err != nil {
			return fmt.Errorf("unable to remove corrupted cache: %v", err)
		}
	} else if err != nil {
		return fmt.Errorf("unable to check if %s exists: %v", cacheImagePath, err)
	}

	if !exists {
		sylog.Infof("Downloading image with ORAS")
		go interruptCleanup(cacheImagePath)

		if err := oras.DownloadImage(cacheImagePath, ref, ociAuth); err != nil {
			return fmt.Errorf("unable to Download Image: %v", err)
		}

		if cacheFileHash, err := oras.ImageHash(cacheImagePath); err != nil {
			return fmt.Errorf("error getting ImageHash: %v", err)
		} else if cacheFileHash != sum {
			return fmt.Errorf("cached file hash(%s) and expected hash(%s) does not match", cacheFileHash, sum)
		}
	} else {
		sylog.Infof("Using cached image")
	}

	// Perms are 777 *prior* to umask in order to allow image to be
	// executed with its leading shebang like a script
	dstFile, err := os.OpenFile(name, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0777)
	if err != nil {
		return fmt.Errorf("while opening destination file: %v", err)
	}
	defer dstFile.Close()

	srcFile, err := os.Open(cacheImagePath)
	if err != nil {
		return fmt.Errorf("while opening cached image: %v", err)
	}
	defer srcFile.Close()

	// Copy SIF from cache
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("while copying image from cache: %v", err)
	}

	sylog.Infof("Pull complete: %s\n", name)

	return nil
}

// OciPull will build a SIF image from the specified oci URI
func OciPull(imgCache *cache.Handle, name, imageURI, tmpDir string, ociAuth *ocitypes.DockerAuthConfig, noHTTPS, noCache bool) error {
	sysCtx := &ocitypes.SystemContext{
		OCIInsecureSkipTLSVerify:    noHTTPS,
		DockerInsecureSkipTLSVerify: noHTTPS,
		DockerAuthConfig:            ociAuth,
	}

	sum, err := ociclient.ImageSHA(imageURI, sysCtx)
	if err != nil {
		return fmt.Errorf("failed to get checksum for %s: %s", imageURI, err)
	}

	if noCache {
		if err := convertDockerToSIF(imgCache, imageURI, name, tmpDir, noHTTPS, true, ociAuth); err != nil {
			return fmt.Errorf("while building SIF from layers: %v", err)
		}
	} else {
		imgName := uri.GetName(imageURI)
		cachedImgPath := imgCache.OciTempImage(sum, imgName)

		exists, err := imgCache.OciTempExists(sum, imgName)
		if err != nil {
			return fmt.Errorf("unable to check if %s exists: %s", imgName, err)
		}
		if !exists {
			sylog.Infof("Converting OCI blobs to SIF format")
			go interruptCleanup(imgName)

			if err := convertDockerToSIF(imgCache, imageURI, cachedImgPath, tmpDir, noHTTPS, false, ociAuth); err != nil {
				return fmt.Errorf("while building SIF from layers: %v", err)
			}
			sylog.Infof("Build complete: %s", name)
		}

		// Perms are 777 *prior* to umask in order to allow image to be
		// executed with its leading shebang like a script
		dstFile, err := os.OpenFile(name, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0777)
		if err != nil {
			return fmt.Errorf("while opening destination file: %v", err)
		}
		defer dstFile.Close()

		srcFile, err := os.Open(cachedImgPath)
		if err != nil {
			return fmt.Errorf("unable to open file for reading: %s: %v", name, err)
		}
		defer srcFile.Close()

		// Copy SIF from cache
		_, err = io.Copy(dstFile, srcFile)
		if err != nil {
			return fmt.Errorf("failed while copying files: %v", err)
		}
	}

	return nil
}

func convertDockerToSIF(imgCache *cache.Handle, image, cachedImgPath, tmpDir string, noHTTPS, noCache bool, authConf *ocitypes.DockerAuthConfig) error {
	if imgCache == nil {
		return fmt.Errorf("image cache is undefined")
	}

	b, err := build.NewBuild(
		image,
		build.Config{
			Dest:   cachedImgPath,
			Format: "sif",
			Opts: types.Options{
				TmpDir:           tmpDir,
				NoCache:          noCache,
				NoTest:           true,
				NoHTTPS:          noHTTPS,
				DockerAuthConfig: authConf,
				ImgCache:         imgCache,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("unable to create new build: %v", err)
	}

	return b.Full()
}
