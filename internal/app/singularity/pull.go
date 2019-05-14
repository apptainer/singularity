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
	"strings"

	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/reference"
	"github.com/containerd/containerd/remotes/docker"
	ocitypes "github.com/containers/image/types"
	"github.com/deislabs/oras/pkg/content"
	orasctx "github.com/deislabs/oras/pkg/context"
	"github.com/deislabs/oras/pkg/oras"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sylabs/singularity/internal/pkg/build"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	ociclient "github.com/sylabs/singularity/internal/pkg/client/oci"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/uri"
	"github.com/sylabs/singularity/pkg/build/types"
	client "github.com/sylabs/singularity/pkg/client/library"
	"github.com/sylabs/singularity/pkg/signing"
	"github.com/sylabs/singularity/pkg/sypgp"
)

var (
	// ErrLibraryPullAbort indicates that the interactive portion of the
	// pull was aborted
	ErrLibraryPullAbort = errors.New("library pull aborted")
)

// LibraryPull will download the image specified by file from the library specified by libraryURI.
// After downloading, the image will be checked for a valid signature and removed if it does not contain one,
// unless specified not to by the unauthenticated bool
func LibraryPull(name, ref, transport, fullURI, libraryURI, keyServerURL, authToken string, force, unauthenticated bool) error {
	if !force {
		if _, err := os.Stat(name); err == nil {
			return fmt.Errorf("image file already exists: %q - will not overwrite", name)
		}
	}

	libraryImage, err := client.GetImage(libraryURI, authToken, fullURI)
	if err != nil {
		return fmt.Errorf("while getting image info: %v", err)
	}

	// required in order to properly allow for library pulls without transport in uri
	// otherwise uri becomes malformed see https://github.com/sylabs/singularity/pull/2683
	var imageName string
	if transport == "" {
		imageName = uri.GetName("library://" + fullURI)
	} else {
		imageName = uri.GetName(fullURI)
	}
	imagePath := cache.LibraryImage(libraryImage.Hash, imageName)
	exists, err := cache.LibraryImageExists(libraryImage.Hash, imageName)
	if err != nil {
		return fmt.Errorf("unable to check if %v exists: %v", imagePath, err)
	}
	if !exists {
		sylog.Infof("Downloading library image")
		if err = client.DownloadImage(imagePath, fullURI, libraryURI, true, authToken); err != nil {
			return fmt.Errorf("unable to Download Image: %v", err)
		}

		if cacheFileHash, err := client.ImageHash(imagePath); err != nil {
			return fmt.Errorf("Error getting ImageHash: %v", err)
		} else if cacheFileHash != libraryImage.Hash {
			return fmt.Errorf("Cached File Hash(%s) and Expected Hash(%s) does not match", cacheFileHash, libraryImage.Hash)
		}
	}

	// Perms are 777 *prior* to umask
	dstFile, err := os.OpenFile(name, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0777)
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	defer dstFile.Close()

	srcFile, err := os.OpenFile(imagePath, os.O_RDONLY, 0444)
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	defer srcFile.Close()

	// Copy SIF from cache
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	var retErr error
	// check if we pulled from the library, if so; is it signed?
	if !unauthenticated {
		imageSigned, err := signing.IsSigned(name, keyServerURL, 0, false, authToken, true)
		if err != nil {
			// err will be: "unable to verify container: %v", err
			sylog.Warningf("%v", err)
			// if theres a warning, return set error to indicate exit 1
			retErr = ErrLibraryUnsigned
		}
		// if container is not signed, print a warning
		if !imageSigned {
			fmt.Fprintf(os.Stderr, "This image is not signed, and thus its contents cannot be verified.\n")
			resp, err := sypgp.AskQuestion("Do you with to proceed? [N/y] ")
			if err != nil {
				return fmt.Errorf("unable to parse input: %v", err)
			}
			if resp == "" || resp != "y" && resp != "Y" {
				fmt.Fprintf(os.Stderr, "Aborting.\n")
				err := os.Remove(name)
				if err != nil {
					return fmt.Errorf("unabel to delete the container: %v", err)
				}
				// set error to exit with status 10 after replying no
				return ErrLibraryPullAbort
			}
		}
	} else {
		sylog.Warningf("Skipping container verification")
	}

	sylog.Infof("Download complete: %s\n", name)

	return retErr
}

// OrasPull will download the image specified by the provided oci reference and store
// it at the location specified by file, it will use credentials if supplied
func OrasPull(name, ref string, force bool, ociAuth *ocitypes.DockerAuthConfig) error {
	ref = strings.TrimPrefix(ref, "//")

	spec, err := reference.Parse(ref)
	if err != nil {
		return fmt.Errorf("Unable to parse oci reference: %s", err)
	}

	// append default tag if no object exists
	if spec.Object == "" {
		spec.Object = SifDefaultTag
		sylog.Infof("No tag or digest found, using default: %s", SifDefaultTag)
	}

	credFn := func(_ string) (string, string, error) {
		return ociAuth.Username, ociAuth.Password, nil
	}
	resolver := docker.NewResolver(docker.ResolverOptions{Credentials: credFn})

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("Failed to get working directory: %s", err)
	}

	store := content.NewFileStore(wd)
	defer store.Close()

	store.AllowPathTraversalOnWrite = true
	store.DisableOverwrite = !force

	allowedMediaTypes := oras.WithAllowedMediaTypes([]string{SifLayerMediaType})
	handlerFunc := func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		if desc.MediaType == SifLayerMediaType {
			nameOld, _ := content.ResolveName(desc)
			_ = store.MapPath(nameOld, name)
		}
		return nil, nil
	}
	pullHandler := oras.WithPullBaseHandler(images.HandlerFunc(handlerFunc))

	_, _, err = oras.Pull(orasctx.Background(), resolver, spec.String(), store, allowedMediaTypes, pullHandler)
	if err != nil {
		return fmt.Errorf("unable to pull from registry: %s", err)
	}

	// ensure container is executable
	if err := os.Chmod(name, 0755); err != nil {
		return fmt.Errorf("unable to set image perms: %s", err)
	}
	sylog.Infof("Download complete: %s\n", name)

	return nil
}

// OciPull will build a SIF image from the specified oci URI
func OciPull(name, imageURI, tmpDir string, ociAuth *ocitypes.DockerAuthConfig, force, noHTTPS bool) error {
	if !force {
		if _, err := os.Stat(name); err == nil {
			return fmt.Errorf("Image file already exists - will not overwrite")
		}
	}

	sysCtx := &ocitypes.SystemContext{
		OCIInsecureSkipTLSVerify:    noHTTPS,
		DockerInsecureSkipTLSVerify: noHTTPS,
		DockerAuthConfig:            ociAuth,
	}

	sum, err := ociclient.ImageSHA(imageURI, sysCtx)
	if err != nil {
		return fmt.Errorf("failed to get checksum for %s: %s", imageURI, err)
	}

	imgName := uri.GetName(imageURI)
	cachedImgPath := cache.OciTempImage(sum, imgName)

	exists, err := cache.OciTempExists(sum, imgName)
	if err != nil {
		return fmt.Errorf("unable to check if %s exists: %s", imgName, err)
	}
	if !exists {
		sylog.Infof("Converting OCI blobs to SIF format")
		if err := convertDockerToSIF(imageURI, cachedImgPath, tmpDir, noHTTPS, ociAuth); err != nil {
			return fmt.Errorf("%v", err)
		}
	} else {
		sylog.Infof("Using cached image")
	}

	// Perms are 777 *prior* to umask
	dstFile, err := os.OpenFile(name, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0777)
	if err != nil {
		return fmt.Errorf("unable to open file for writing: %s: %v", name, err)
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

	return nil
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
		},
	)
	if err != nil {
		return fmt.Errorf("Unable to create new build: %v", err)
	}

	return b.Full()
}
