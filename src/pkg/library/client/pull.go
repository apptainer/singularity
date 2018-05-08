/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package client

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/singularityware/singularity/src/pkg/sylog"
	"gopkg.in/cheggaaa/pb.v1"
)

// DownloadImage will retrieve an image from the Container Library,
// saving it into the specified file
func DownloadImage(filePath string, libraryRef string, libraryURL string, Force bool) error {

	if !isLibraryPullRef(libraryRef) {
		return fmt.Errorf("Not a valid library reference: %s", libraryRef)
	}

	if filePath == "" {
		_, _, container, tags := parseLibraryRef(libraryRef)
		filePath = fmt.Sprintf("%s_%s.sif", container, tags[0])
		sylog.Infof("Download filename not provided. Downloading to: %s\n", filePath)
	}

	url := libraryURL + "/v1/imagefile/" + strings.TrimPrefix(libraryRef, "library://")

	sylog.Debugf("Pulling from URL: %s\n", url)

	if !Force {
		if _, err := os.Stat(filePath); err == nil {
			return fmt.Errorf("image file already exists - will not overwrite")
		}
	}

	// Perms are 777 *prior* to umask
	out, err := os.OpenFile(filePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 777)
	if err != nil {
		return err
	}
	defer out.Close()

	sylog.Debugf("Created output file: %s\n", filePath)

	res, err := http.Get(url)
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

	sylog.Debugf("OK response received, beginning body download\n", filePath)

	bodySize := res.ContentLength
	bar := pb.New(int(bodySize)).SetUnits(pb.U_BYTES)
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
