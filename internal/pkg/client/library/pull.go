// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package library

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"runtime"

	scs "github.com/sylabs/scs-library-client/client"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/vbauerster/mpb/v4"
	"github.com/vbauerster/mpb/v4/decor"
)

var (
	// ErrLibraryPullUnsigned indicates that the interactive portion of the pull was aborted.
	ErrLibraryPullUnsigned = errors.New("failed to verify container")
)

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

func Pull(ctx context.Context, imgCache *cache.Handle, pullFrom string, arch string, tmpDir string, scsConfig *scs.Config, keystoreURI string) (string, error) {
	c, err := scs.NewClient(scsConfig)
	if err != nil {
		return "", fmt.Errorf("unable to initialize client library: %v", err)
	}

	imageRef := NormalizeLibraryRef(pullFrom)

	libraryImage, err := c.GetImage(ctx, runtime.GOARCH, imageRef)
	if err == scs.ErrNotFound {
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

		if err = DownloadImageNoProgress(ctx, c, imagePath, runtime.GOARCH, imageRef); err != nil {
			return "", fmt.Errorf("unable to download image: %v", err)
		}

	} else {
		cacheEntry, err := imgCache.GetEntry(cache.LibraryCacheType, libraryImage.Hash)
		if err != nil {
			return "", fmt.Errorf("unable to check if %v exists in cache: %v", libraryImage.Hash, err)
		}
		if !cacheEntry.Exists {
			sylog.Infof("Downloading library image")

			if err := DownloadImageNoProgress(ctx, c, cacheEntry.TmpPath, runtime.GOARCH, imageRef); err != nil {
				return "", fmt.Errorf("unable to download image: %v", err)
			}

			if cacheFileHash, err := scs.ImageHash(cacheEntry.TmpPath); err != nil {
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

// PullToFile will build a SIF image from the specified oci URI and place it at the specified dest
func PullToFile(ctx context.Context, imgCache *cache.Handle, pullTo, pullFrom, arch string, tmpDir string, scsConfig *scs.Config, keystoreURI string) (string, error) {

	src, err := Pull(ctx, imgCache, pullFrom, arch, tmpDir, scsConfig, keystoreURI)
	if err != nil {
		return "", fmt.Errorf("error fetching image to cache: %v", err)
	}

	err = fs.CopyFile(src, pullTo, 0755)
	if err != nil {
		return "", fmt.Errorf("error fetching image to cache: %v", err)
	}

	return pullTo, nil
}
