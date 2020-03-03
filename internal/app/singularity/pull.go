// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"context"
	"errors"
	"fmt"
	ocitypes "github.com/containers/image/v5/types"
	"github.com/sylabs/singularity/internal/pkg/build"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	ociclient "github.com/sylabs/singularity/internal/pkg/client/oci"
	"github.com/sylabs/singularity/internal/pkg/oras"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/build/types"
	shub "github.com/sylabs/singularity/pkg/client/shub"
	"github.com/vbauerster/mpb/v4"
	"github.com/vbauerster/mpb/v4/decor"
	"io"
)

var (
	// ErrLibraryPullUnsigned indicates that the interactive portion of the pull was aborted.
	ErrLibraryPullUnsigned = errors.New("failed to verify container")
)

// PullShub will download a image from shub, and cache it. Next time
// that container is downloaded this will just use that cached image.
func PullShub(imgCache *cache.Handle, filePath string, shubRef string, noHTTPS bool) (err error) {
	shubURI, err := shub.ShubParseReference(shubRef)
	if err != nil {
		return fmt.Errorf("failed to parse shub uri: %s", err)
	}

	// Get the image manifest
	manifest, err := shub.GetManifest(shubURI, noHTTPS)
	if err != nil {
		return fmt.Errorf("failed to get manifest for: %s: %s", shubRef, err)
	}

	if imgCache.IsDisabled() {
		// Dont use cached image
		if err := shub.DownloadImage(manifest, filePath, shubRef, true, noHTTPS); err != nil {
			return err
		}
	} else {
		cacheEntry, err := imgCache.GetEntry(cache.ShubCacheType, manifest.Commit)
		if err != nil {
			return fmt.Errorf("unable to check if %v exists in cache: %v", manifest.Commit, err)
		}
		if !cacheEntry.Exists {
			sylog.Infof("Downloading shub image")
			go interruptCleanup(func(){
				_ = cacheEntry.Abort()
			})

			err := shub.DownloadImage(manifest, cacheEntry.TmpPath, shubRef, true, noHTTPS)
			if err != nil {
				return err
			}

			err = cacheEntry.Finalize()
			if err != nil {
				return err
			}

		} else {
			sylog.Infof("Use image from cache")
		}

		cacheEntry.CopyTo(filePath)

	}

	return nil
}

// printProgress is called to display progress bar while downloading image from library.
func printProgress(totalSize int64, r io.Reader, w io.Writer) error {
	p := mpb.New()
	bar := p.AddBar(totalSize,
		mpb.PrependDecorators(
			decor.Counters(decor.UnitKiB, "%.1f / %.1f"),
		),
		mpb.AppendDecorators(
			decor.Percentage(),
			decor.AverageSpeed(decor.UnitKiB, " % .1f "),
			decor.AverageETA(decor.ET_STYLE_GO),
		),
	)

	// create proxy reader
	bodyProgress := bar.ProxyReader(r)

	// Write the body to file
	_, err := io.Copy(w, bodyProgress)
	if err != nil {
		return err
	}

	return nil
}

// OrasPull will download the image specified by the provided oci reference and store
// it at the location specified by file, it will use credentials if supplied
func OrasPull(ctx context.Context, imgCache *cache.Handle, name, ref string, force bool, ociAuth *ocitypes.DockerAuthConfig) error {
	hash, err := oras.ImageSHA(ctx, ref, ociAuth)
	if err != nil {
		return fmt.Errorf("failed to get checksum for %s: %s", ref, err)
	}

	if imgCache.IsDisabled() {
		// Dont use cached image
		if err := oras.DownloadImage(name, ref, ociAuth); err != nil {
			return fmt.Errorf("unable to Download Image: %v", err)
		}
	} else {
		cacheEntry, err := imgCache.GetEntry(cache.OrasCacheType, hash)
		if err != nil {
			return fmt.Errorf("unable to check if %v exists in cache: %v", hash, err)
		}
		if !cacheEntry.Exists {
			sylog.Infof("Downloading oras image")
			go interruptCleanup(func() {
				_ = cacheEntry.Abort()
			})

			if err := oras.DownloadImage(cacheEntry.TmpPath, ref, ociAuth); err != nil {
				return fmt.Errorf("unable to Download Image: %v", err)
			}
			if cacheFileHash, err := oras.ImageHash(cacheEntry.TmpPath); err != nil {
				return fmt.Errorf("error getting ImageHash: %v", err)
			} else if cacheFileHash != hash {
				_ = cacheEntry.Abort()
				return fmt.Errorf("cached file hash(%s) and expected hash(%s) does not match", cacheFileHash, hash)
			}

			err = cacheEntry.Finalize()
			if err != nil {
				return err
			}

		} else {
			sylog.Infof("Using cached image")

		}

		cacheEntry.CopyTo(name)

	}

	sylog.Infof("Pull complete: %s\n", name)

	return nil
}

// OciPull will build a SIF image from the specified oci URI
func OciPull(ctx context.Context, imgCache *cache.Handle, name, imageURI, tmpDir string, ociAuth *ocitypes.DockerAuthConfig, noHTTPS, noCleanUp bool) error {
	sysCtx := &ocitypes.SystemContext{
		OCIInsecureSkipTLSVerify:    noHTTPS,
		DockerInsecureSkipTLSVerify: ocitypes.NewOptionalBool(noHTTPS),
		DockerAuthConfig:            ociAuth,
	}

	hash, err := ociclient.ImageSHA(ctx, imageURI, sysCtx)
	if err != nil {
		return fmt.Errorf("failed to get checksum for %s: %s", imageURI, err)
	}

	if imgCache.IsDisabled() {
		if err := convertDockerToSIF(ctx, imgCache, imageURI, name, tmpDir, noHTTPS, noCleanUp, ociAuth); err != nil {
			return fmt.Errorf("while building SIF from layers: %v", err)
		}
	} else {

		cacheEntry, err := imgCache.GetEntry(cache.OciTempCacheType, hash)
		if err != nil {
			return fmt.Errorf("unable to check if %v exists in cache: %v", hash, err)
		}
		if !cacheEntry.Exists {
			sylog.Infof("Converting OCI blobs to SIF format")
			go interruptCleanup(func(){
				_ = cacheEntry.Abort()
			})

			if err := convertDockerToSIF(ctx, imgCache, imageURI, cacheEntry.TmpPath, tmpDir, noHTTPS, noCleanUp, ociAuth); err != nil {
				return fmt.Errorf("while building SIF from layers: %v", err)
			}

			err = cacheEntry.Finalize()
			if err != nil {
				return err
			}

		} else {
			sylog.Infof("Use image from cache")
		}

		cacheEntry.CopyTo(name)

	}

	return nil
}

func convertDockerToSIF(ctx context.Context, imgCache *cache.Handle, image, cachedImgPath, tmpDir string, noHTTPS, noCleanUp bool, authConf *ocitypes.DockerAuthConfig) error {
	if imgCache == nil {
		return fmt.Errorf("image cache is undefined")
	}

	b, err := build.NewBuild(
		image,
		build.Config{
			Dest:      cachedImgPath,
			Format:    "sif",
			NoCleanUp: noCleanUp,
			Opts: types.Options{
				TmpDir:           tmpDir,
				NoCache:          imgCache.IsDisabled(),
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

	return b.Full(ctx)
}

