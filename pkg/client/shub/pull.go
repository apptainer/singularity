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
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/uri"
	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
	pb "gopkg.in/cheggaaa/pb.v1"
)

// Timeout for an image pull in seconds (2 hours)
const pullTimeout = 7200

// DownloadImage will retrieve an image from the Container Singularityhub,
// saving it into the specified file
func DownloadImage(filePath string, shubRef string, force, noHTTPS bool) (err error) {
	if !force {
		if _, err := os.Stat(filePath); err == nil {
			return fmt.Errorf("image file already exists - will not overwrite")
		}
	}

	imageName := uri.GetName(shubRef)
	imagePath := cache.ShubImage("hash", imageName)

	exists, err := cache.ShubImageExists("hash", imageName)
	if err != nil {
		return fmt.Errorf("unable to check if %v exists: %v", imagePath, err)
	}
	if !exists {
		sylog.Infof("Downloading shub image")
		err := downloadImage(imagePath, shubRef, true, noHTTPS)
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

	// Copy SIF from cache
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("while copying image from cache: %v", err)
	}

	return nil
}

func downloadImage(filePath, shubRef string, force, noHTTPS bool) error {
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
		err := jsonresp.ReadError(resp.Body)
		if err != nil {
			return fmt.Errorf("Download did not succeed: %s", err.Error())
		}
		return fmt.Errorf("Download did not succeed: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
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

	return nil
}
