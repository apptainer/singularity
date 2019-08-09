// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	jsonresp "github.com/sylabs/json-resp"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
	pb "gopkg.in/cheggaaa/pb.v1"
)

// Timeout for an image pull in seconds (2 hours)
const pullTimeout = 7200

// DownloadImage image will download a shub image to a path. This will not try
// to cache it, or use cache.
func DownloadImage(manifest ShubAPIResponse, filePath, shubRef string, force, noHTTPS bool) error {
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

	shubURI, err := ShubParseReference(shubRef)
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

	bodySize := resp.ContentLength
	bar := pb.New(int(bodySize)).SetUnits(pb.U_BYTES)
	if sylog.GetLevel() < 0 {
		bar.NotPrint = true
	}
	bar.ShowTimeLeft = true
	bar.ShowSpeed = true
	bar.Start()

	// create proxy reader
	bodyProgress := bar.NewProxyReader(resp.Body)

	// Write the body to file
	bytesWritten, err := io.Copy(out, bodyProgress)
	if err != nil {
		return err
	}

	// Simple check to make sure image received is the correct size
	if resp.ContentLength == -1 {
		sylog.Warningf("unknown image length")
	} else if bytesWritten != resp.ContentLength {
		return fmt.Errorf("image received is not the right size. supposed to be: %v actually: %v", resp.ContentLength, bytesWritten)
	}

	bar.Finish()

	sylog.Debugf("Download complete: %s\n", filePath)

	return nil
}
