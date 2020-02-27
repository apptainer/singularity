// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
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
	"github.com/vbauerster/mpb/v4"
	"github.com/vbauerster/mpb/v4/decor"
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
