// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package net

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/sylabs/singularity/internal/pkg/client/cache"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
	"github.com/vbauerster/mpb/v4/decor"
	"github.com/vbauerster/mpb/v4"
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
func DownloadImage(filePath string, netURL string) error {

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

	client := &http.Client{
		Timeout: pullTimeout * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", useragent.Value())

	res, err := client.Do(req)
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

	sylog.Debugf("Created output file: %s\n", filePath)

	bodySize := res.ContentLength
	p := mpb.New()
	bar := p.AddBar(bodySize,
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
	bodyProgress := bar.ProxyReader(res.Body)

	// Write the body to file
	_, err = io.Copy(out, bodyProgress)
	if err != nil {
		return err
	}

	sylog.Debugf("Download complete\n")

	return nil

}

func Pull(imgCache *cache.Handle, pullFrom string, tmpDir string) (imagePath string, err error) {
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

	if imgCache.IsDisabled() {
		file, err := ioutil.TempFile(tmpDir, "sbuild-tmp-cache-")
		if err != nil {
			return "", fmt.Errorf("unable to create tmp file: %v", err)
		}
		imagePath = file.Name()
		sylog.Infof("Downloading image to tmp cache: %s", imagePath)

		// Dont use cached image
		if err := DownloadImage(imagePath, pullFrom); err != nil {
			return "", fmt.Errorf("unable to Download Image: %v", err)
		}
	} else {

		cacheEntry, err := imgCache.GetEntry(cache.NetCacheType, hash)
		if err != nil {
			return "", fmt.Errorf("unable to check if %v exists in cache: %v", hash, err)
		}

		if !cacheEntry.Exists {
			sylog.Infof("Downloading network image")
			err := DownloadImage(cacheEntry.TmpPath, pullFrom)
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

// PullToFile will fetch an image from the specified URI and place it at the specified dest
func PullToFile(imgCache *cache.Handle, pullTo, pullFrom, tmpDir string) (sifFile string, err error) {

	src, err := Pull(imgCache, pullFrom, tmpDir)
	if err != nil {
		return "", fmt.Errorf("error fetching image to cache: %v", err)
	}

	err = fs.CopyFile(src, pullTo, 0755)
	if err != nil {
		return "", fmt.Errorf("error fetching image to cache: %v", err)
	}

	return pullTo, nil
}
