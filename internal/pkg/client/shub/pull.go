// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package shub

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/sylabs/singularity/internal/pkg/client"

	jsonresp "github.com/sylabs/json-resp"
	"github.com/sylabs/singularity/internal/pkg/cache"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/pkg/sylog"
	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
)

// Timeout for an image pull in seconds (2 hours)
const pullTimeout = 7200

// DownloadImage image will download a shub image to a path. This will not try
// to cache it, or use cache.
func DownloadImage(ctx context.Context, manifest APIResponse, filePath, shubRef string, force, noHTTPS bool) error {
	sylog.Debugf("Downloading container from Shub")
	if !force {
		if _, err := os.Stat(filePath); err == nil {
			return fmt.Errorf("image file already exists: %q - will not overwrite", filePath)
		}
	}

	// use custom parser to make sure we have a valid shub URI
	if ok := isShubPullRef(shubRef); !ok {
		sylog.Fatalf("Invalid shub URI")
	}

	shubURI, err := ParseReference(shubRef)
	if err != nil {
		return fmt.Errorf("failed to parse shub uri: %v", err)
	}

	if filePath == "" {
		filePath = fmt.Sprintf("%s_%s.simg", shubURI.container, shubURI.tag)
		sylog.Infof("Download filename not provided. Downloading to: %s\n", filePath)
	}

	// Get the image based on the manifest
	httpc := http.Client{
		Timeout: pullTimeout * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, manifest.Image, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", useragent.Value())

	if noHTTPS {
		req.URL.Scheme = "http"
	}

	// Do the request, if status isn't success, return error
	resp, err := httpc.Do(req)
	if resp == nil {
		return fmt.Errorf("no response received from singularity hub")
	}
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("the requested image was not found in singularity hub")
	}
	sylog.Debugf("%s response received, beginning image download\n", resp.Status)

	if resp.StatusCode != http.StatusOK {
		err := jsonresp.ReadError(resp.Body)
		if err != nil {
			return fmt.Errorf("download did not succeed: %s", err.Error())
		}
		return fmt.Errorf("download did not succeed: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	// Perms are 777 *prior* to umask
	out, err := os.OpenFile(filePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0777)
	if err != nil {
		return err
	}
	defer out.Close()

	sylog.Debugf("Created output file: %s\n", filePath)

	// Write the body to file
	pb := client.ProgressBarCallback(ctx)
	err = pb(resp.ContentLength, resp.Body, out)
	if err != nil {
		// Delete incomplete image file in the event of failure
		// we get here e.g. if the context is canceled by Ctrl-C
		resp.Body.Close()
		out.Close()
		sylog.Infof("Cleaning up incomplete download: %s", filePath)
		if err := os.Remove(filePath); err != nil {
			sylog.Errorf("Error while removing incomplete download: %v", err)
		}
		return err
	}
	out.Close()

	st, err := os.Stat(out.Name())
	if err != nil {
		return fmt.Errorf("error checking output file %s: %v", out.Name(), err)
	}

	// Simple check to make sure image received is the correct size
	if resp.ContentLength == -1 {
		sylog.Warningf("unknown image length")
	} else if st.Size() != resp.ContentLength {
		return fmt.Errorf("image received is not the right size. supposed to be: %v actually: %v", resp.ContentLength, st.Size())
	}

	sylog.Debugf("Download complete: %s\n", filePath)

	return nil
}

// pull will pull an oras image into the cache if directTo="", or a specific file if directTo is set.
func pull(ctx context.Context, imgCache *cache.Handle, directTo, pullFrom string, noHTTPS bool) (imagePath string, err error) {
	shubURI, err := ParseReference(pullFrom)
	if err != nil {
		return "", fmt.Errorf("failed to parse shub uri: %s", err)
	}

	// Get the image manifest
	manifest, err := GetManifest(shubURI, noHTTPS)
	if err != nil {
		return "", fmt.Errorf("failed to get manifest for: %s: %s", pullFrom, err)
	}

	if directTo != "" {
		sylog.Infof("Downloading shub image")
		if err := DownloadImage(ctx, manifest, directTo, pullFrom, true, noHTTPS); err != nil {
			return "", err
		}
		imagePath = directTo
	} else {
		cacheEntry, err := imgCache.GetEntry(cache.ShubCacheType, manifest.Commit)
		if err != nil {
			return "", fmt.Errorf("unable to check if %v exists in cache: %v", manifest.Commit, err)
		}
		defer cacheEntry.CleanTmp()
		if !cacheEntry.Exists {
			sylog.Infof("Downloading shub image")

			err := DownloadImage(ctx, manifest, cacheEntry.TmpPath, pullFrom, true, noHTTPS)
			if err != nil {
				return "", err
			}

			err = cacheEntry.Finalize()
			if err != nil {
				return "", err
			}
			imagePath = cacheEntry.Path
		} else {
			sylog.Infof("Use cached image")
			imagePath = cacheEntry.Path
		}

	}

	return imagePath, nil
}

// Pull will pull a shub image to the cache or direct to a temporary file if cache is disabled
func Pull(ctx context.Context, imgCache *cache.Handle, pullFrom, tmpDir string, noHTTPS bool) (imagePath string, err error) {

	directTo := ""

	if imgCache.IsDisabled() {
		file, err := ioutil.TempFile(tmpDir, "sbuild-tmp-cache-")
		if err != nil {
			return "", fmt.Errorf("unable to create tmp file: %v", err)
		}
		directTo = file.Name()
		sylog.Infof("Downloading shub image to tmp cache: %s", directTo)
	}

	return pull(ctx, imgCache, directTo, pullFrom, noHTTPS)

}

// PullToFile will pull a shub image to the specified location, through the cache, or directly if cache is disabled
func PullToFile(ctx context.Context, imgCache *cache.Handle, pullTo, pullFrom, tmpDir string, noHTTPS bool) (imagePath string, err error) {

	directTo := ""
	if imgCache.IsDisabled() {
		directTo = pullTo
		sylog.Debugf("Cache disabled, pulling directly to: %s", directTo)
	}

	src, err := pull(ctx, imgCache, directTo, pullFrom, noHTTPS)
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
