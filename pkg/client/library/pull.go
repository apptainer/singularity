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
	"strings"
	"time"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
	pb "gopkg.in/cheggaaa/pb.v1"
)

// Timeout for an image pull in seconds - could be a large download...
const pullTimeout = 1800

// DownloadImage will retrieve an image from the Container Library,
// saving it into the specified file
func DownloadImage(filePath string, libraryRef string, libraryURL string, Force bool, authToken string) error {

	if !IsLibraryPullRef(libraryRef) {
		return fmt.Errorf("Not a valid library reference: %s", libraryRef)
	}

	if filePath == "" {
		_, _, container, tags := parseLibraryRef(libraryRef)
		filePath = fmt.Sprintf("%s_%s.sif", container, tags[0])
		sylog.Infof("Download filename not provided. Downloading to: %s\n", filePath)
	}

	libraryRef = strings.TrimPrefix(libraryRef, "library://")

	if !strings.Contains(libraryRef, ":") {
		sylog.Infof("No tag specified, using ':latest'")
		libraryRef += ":latest"
	}

	url := libraryURL + "/v1/imagefile/" + libraryRef

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

	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
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
		jRes, err := ParseErrorBody(res.Body)
		if err != nil {
			jRes = ParseErrorResponse(res)
		}
		return fmt.Errorf("Download did not succeed: %d %s\n\t%v",
			jRes.Error.Code, jRes.Error.Status, jRes.Error.Message)
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

	// create proxy reader
	bodyProgress := bar.NewProxyReader(res.Body)

	bar.Start()

	// Write the body to file
	_, err = io.Copy(out, bodyProgress)
	if err != nil {
		return err
	}

	bar.Finish()

	sylog.Debugf("Download complete\n")

	return nil

}
