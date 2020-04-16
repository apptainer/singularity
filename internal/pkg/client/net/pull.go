// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package net

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/sylabs/singularity/internal/pkg/cache"
	"github.com/sylabs/singularity/internal/pkg/client"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/pkg/sylog"
	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
)

// Timeout for an image pull in seconds - could be a large download...
const pullTimeout = 1800

// IsNetPullRef returns true if the provided string is a valid url
// reference for a pull operation.
func IsNetPullRef(netRef string) bool {
	match, _ := regexp.MatchString("^http(s)?://", netRef)
	return match
}

// DownloadImage will retrieve an image from an http(s) URI,
// saving it into the specified file
func DownloadImage(ctx context.Context, filePath string, netURL string) error {

	if !IsNetPullRef(netURL) {
		return fmt.Errorf("not a valid url reference: %s", netURL)
	}
	if filePath == "" {
		refParts := strings.Split(netURL, "/")
		filePath = refParts[len(refParts)-1]
		sylog.Infof("Download filename not provided. Downloading to: %s\n", filePath)
	}

	url := netURL
	sylog.Debugf("Pulling from URL: %s\n", url)

	httpClient := &http.Client{
		Timeout: pullTimeout * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", useragent.Value())

	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return fmt.Errorf("the requested image was not found")
	}

	if res.StatusCode != http.StatusOK {
		buf := new(bytes.Buffer)
		buf.ReadFrom(res.Body)
		s := buf.String()
		return fmt.Errorf("Download did not succeed: %d %s\n\t",
			res.StatusCode, s)
	}

	sylog.Debugf("OK response received, beginning body download\n")

	// Perms are 777 *prior* to umask
	out, err := os.OpenFile(filePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0777)
	if err != nil {
		return err
	}
	defer out.Close()

	pb := client.ProgressBarCallback(ctx)

	err = pb(res.ContentLength, res.Body, out)

	if err != nil {
		// Delete incomplete image file in the event of failure
		// we get here e.g. if the context is canceled by Ctrl-C
		res.Body.Close()
		out.Close()
		sylog.Infof("Cleaning up incomplete download: %s", filePath)
		if err := os.Remove(filePath); err != nil {
			sylog.Errorf("Error while removing incomplete download: %v", err)
		}
		return err
	}

	sylog.Debugf("Download complete\n")

	return nil
}

// pull will pull a http(s) image into the cache if directTo="", or a specific file if directTo is set.
func pull(ctx context.Context, imgCache *cache.Handle, directTo, pullFrom string) (imagePath string, err error) {
	// We will cache using a sha256 over the URL and the date of the file that
	// is to be fetched, as returned by an HTTP HEAD call and the Last-Modified
	// header. If no date is available, use the current date-time, which will
	// effectively result in no caching.
	imageDate := time.Now().String()

	req, err := http.NewRequest("HEAD", pullFrom, nil)
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
	h.Write([]byte(pullFrom + imageDate))
	hash := hex.EncodeToString(h.Sum(nil))
	sylog.Debugf("Image hash for cache is: %s", hash)

	if directTo != "" {
		sylog.Infof("Downloading network image")
		if err := DownloadImage(ctx, directTo, pullFrom); err != nil {
			return "", fmt.Errorf("unable to Download Image: %v", err)
		}
		imagePath = directTo

	} else {
		cacheEntry, err := imgCache.GetEntry(cache.NetCacheType, hash)
		if err != nil {
			return "", fmt.Errorf("unable to check if %v exists in cache: %v", hash, err)
		}
		defer cacheEntry.CleanTmp()

		if !cacheEntry.Exists {
			sylog.Infof("Downloading network image")
			err := DownloadImage(ctx, cacheEntry.TmpPath, pullFrom)
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

		imagePath = cacheEntry.Path
	}

	return imagePath, nil
}

// Pull will pull a http(s) image to the cache or direct to a temporary file if cache is disabled
func Pull(ctx context.Context, imgCache *cache.Handle, pullFrom string, tmpDir string) (imagePath string, err error) {

	directTo := ""

	if imgCache.IsDisabled() {
		file, err := ioutil.TempFile(tmpDir, "sbuild-tmp-cache-")
		if err != nil {
			return "", fmt.Errorf("unable to create tmp file: %v", err)
		}
		directTo = file.Name()
		sylog.Infof("Downloading library image to tmp cache: %s", directTo)
	}

	return pull(ctx, imgCache, directTo, pullFrom)
}

// PullToFile will pull an http(s) image to the specified location, through the cache, or directly if cache is disabled
func PullToFile(ctx context.Context, imgCache *cache.Handle, pullTo, pullFrom, tmpDir string) (imagePath string, err error) {

	directTo := ""
	if imgCache.IsDisabled() {
		directTo = pullTo
		sylog.Debugf("Cache disabled, pulling directly to: %s", directTo)
	}

	src, err := pull(ctx, imgCache, directTo, pullFrom)
	if err != nil {
		return "", fmt.Errorf("error fetching image to cache: %v", err)
	}

	if directTo == "" {
		// mode is before umask if pullTo doesn't exist
		err = fs.CopyFileAtomic(src, pullTo, 0777)
		if err != nil {
			return "", fmt.Errorf("error copying image out of cache: %v", err)
		}
	}

	return pullTo, nil
}
