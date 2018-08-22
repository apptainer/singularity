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

	util "github.com/singularityware/singularity/src/pkg/library/client"
	"github.com/singularityware/singularity/src/pkg/sylog"
	pb "gopkg.in/cheggaaa/pb.v1"
)

// Timeout for an image pull in seconds
const pullTimeout = 1800

// DownloadImage will retrieve an image from the Container Singularityhub,
// saving it into the specified file
func DownloadImage(filePath string, shubRef string, Force bool) (err error) {
	sylog.Debugf("Downloading container from Shub")

	sc := ShubClient{FilePath: filePath}

	//use custom parser to make sure we have a valid shub URI
	sc.ShubURI, err = ShubParseReference(shubRef)
	if err != nil {
		sylog.Fatalf("Invalid shub URI: %v", err)
		return
	}

	if filePath == "" {
		filePath = fmt.Sprintf("%s_%s.simg", sc.ShubURI.container, sc.ShubURI.tag)
		sylog.Infof("Download filename not provided. Downloading to: %s\n", filePath)
	}

	if !Force {
		if _, err := os.Stat(filePath); err == nil {
			return fmt.Errorf("image file already exists - will not overwrite")
		}
	}

	// Get the image manifest
	if err = sc.getManifest(); err != nil {
		sylog.Fatalf("Failed to get manifest from Shub: %v", err)
		return
	}

	// Get the image based on the manifest
	resp, err := http.Get(sc.ShubAPIResponse.Image)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("The requested image was not found in singularity hub")
	}

	if resp.StatusCode != http.StatusOK {
		jRes, err := util.ParseErrorBody(resp.Body)
		if err != nil {
			jRes = util.ParseErrorResponse(resp)
		}
		return fmt.Errorf("Download did not succeed: %d %s\n\t%v",
			jRes.Error.Code, jRes.Error.Status, jRes.Error.Message)
	}

	sylog.Debugf("OK response received, beginning image download\n")

	// Perms are 777 *prior* to umask
	out, err := os.OpenFile(filePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0777)
	if err != nil {
		return err
	}
	defer out.Close()

	sylog.Debugf("Created output file: %s\n", filePath)

	bodySize := resp.ContentLength
	bar := pb.New(int(bodySize)).SetUnits(pb.U_BYTES)
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
	//Simple check to make sure image received is the correct size
	if bytesWritten != resp.ContentLength {
		return fmt.Errorf("Image received is not the right size. Supposed to be: %v  Actually: %v", resp.ContentLength, bytesWritten)
	}

	bar.Finish()

	sylog.Debugf("Download complete\n")

	return err
}
