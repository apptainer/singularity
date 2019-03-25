// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
	"gopkg.in/cheggaaa/pb.v1"
)

// Timeout for an image pull in seconds - could be a large download...
const pullTimeout = 1800

// IsNetPullRef returns true if the provided string is a valid url
// reference for a pull operation.
func IsNetPullRef(libraryRef string) bool {
	match, _ := regexp.MatchString("^http(s)?://", libraryRef)
	return match
}

// DownloadImage will retrieve an image from the Container Library,
// saving it into the specified file
func DownloadImage(filePath string, libraryURL string, Force bool) error {

	if !IsNetPullRef(libraryURL) {
		return fmt.Errorf("Not a valid url reference: %s", libraryURL)
	}
	if filePath == "" {
		refParts := strings.Split(libraryURL, "/")
		filePath = refParts[len(refParts)-1]
		sylog.Infof("Download filename not provided. Downloading to: %s\n", filePath)
	}

	url := libraryURL
	sylog.Debugf("Pulling from URL: %s\n", url)

	if !Force {
		if _, err := os.Stat(filePath); err == nil {
			return fmt.Errorf("image file already exists - will not overwrite")
		}
	}

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
		return fmt.Errorf("The requested image was not found in the library")
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
	bar := pb.New(int(bodySize)).SetUnits(pb.U_BYTES)
	if sylog.GetLevel() < 0 {
		bar.NotPrint = true
	}
	bar.ShowTimeLeft = true
	bar.ShowSpeed = true
	bar.Start()

	// create proxy reader
	bodyProgress := bar.NewProxyReader(res.Body)

	// Write the body to file
	_, err = io.Copy(out, bodyProgress)
	if err != nil {
		return err
	}

	bar.Finish()

	sylog.Debugf("Download complete\n")

	return nil

}
