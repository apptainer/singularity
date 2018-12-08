// Copyright (c) 2018, Sylabs Inc. All rights reserved.
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

	"github.com/sylabs/singularity/internal/pkg/sylog"
	util "github.com/sylabs/singularity/pkg/client/library"
	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
	pb "gopkg.in/cheggaaa/pb.v1"
)

// Timeout for an image pull in seconds (2 hours)
const pullTimeout = 7200

// DownloadImage will retrieve an image from the Container Singularityhub,
// saving it into the specified file
func DownloadImage(filePath string, shubRef string, force, noHTTPS bool) (err error) {
	sylog.Debugf("Downloading container from Shub")

	// use custom parser to make sure we have a valid shub URI
	if ok := isShubPullRef(shubRef); !ok {
		sylog.Fatalf("Invalid shub URI")
	}

	ShubURI, err := shubParseReference(shubRef)
	if err != nil {
		return fmt.Errorf("Failed to parse shub URI: %v", err)
	}

	if filePath == "" {
		filePath = fmt.Sprintf("%s_%s.simg", ShubURI.container, ShubURI.tag)
		sylog.Infof("Download filename not provided. Downloading to: %s\n", filePath)
	}

	if !force {
		if _, err := os.Stat(filePath); err == nil {
			return fmt.Errorf("image file already exists - will not overwrite")
		}
	}

	// Get the image manifest
	manifest, err := getManifest(ShubURI, noHTTPS)
	if err != nil {
		return fmt.Errorf("Failed to get manifest from Shub: %v", err)
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
		return fmt.Errorf("No response received from singularity hub")
	}
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("The requested image was not found in singularity hub")
	}
	sylog.Debugf("%s response received, beginning image download\n", resp.Status)

	if resp.StatusCode != http.StatusOK {
		jRes, err := util.ParseErrorBody(resp.Body)
		if err != nil {
			jRes = util.ParseErrorResponse(resp)
		}
		return fmt.Errorf("Download did not succeed: %d %s\n\t%v",
			jRes.Error.Code, jRes.Error.Status, jRes.Error.Message)
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
	if bytesWritten != resp.ContentLength {
		return fmt.Errorf("Image received is not the right size. Supposed to be: %v  Actually: %v", resp.ContentLength, bytesWritten)
	}

	bar.Finish()

	sylog.Debugf("Download complete\n")

	return err
}
