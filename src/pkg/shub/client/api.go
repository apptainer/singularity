// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	sytypes "github.com/singularityware/singularity/src/pkg/build/types"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/user-agent"
)

const (
	defaultRegistry string = `singularity-hub.org/api/container/`
)

// ShubURI stores the various components of a singularityhub URI
type ShubURI struct {
	registry   string
	user       string
	container  string
	tag        string
	digest     string
	defaultReg bool
}

// ShubAPIResponse holds the information returned from the Shub API
type ShubAPIResponse struct {
	Image   string `json:"image"`
	Name    string `json:"name"`
	Tag     string `json:"tag"`
	Version string `json:"version"`
}

// ShubClient holds the information for interacting with Singularity Hub API
type ShubClient struct {
	FilePath string
	Tmpfile  string
	ShubURI
	*ShubAPIResponse
	*sytypes.Bundle
	ShubURL string
}

// fetchImage downloads an image from Singularity Hub
func (s *ShubClient) fetchImage(bundle *sytypes.Bundle) (err error) {
	// Create temporary download name
	tmpfile, err := ioutil.TempFile(bundle.Path, "shub-container")
	sylog.Debugf("\nCreating temporary image file %v\n", tmpfile.Name())
	if err != nil {
		return err
	}
	defer tmpfile.Close()

	// Get the image based on the manifest
	resp, err := http.Get(s.ShubAPIResponse.Image)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Write the body to file
	bytesWritten, err := io.Copy(tmpfile, resp.Body)
	if err != nil {
		return err
	}
	//Simple check to make sure image received is the correct size
	if bytesWritten != resp.ContentLength {
		return fmt.Errorf("Image received is not the right size. Supposed to be: %v  Actually: %v", resp.ContentLength, bytesWritten)
	}

	s.Tmpfile = tmpfile.Name()
	return nil
}

// getManifest will return the image manifest for a container uri
// from Singularity Hub.
func (s *ShubClient) getManifest() (err error) {

	// Create a new Singularity Hub client
	sc := http.Client{
		Timeout: 30 * time.Second,
	}

	//if we are using a non default registry error out for now
	if !s.ShubURI.defaultReg {
		return errors.New("Only the default Singularity Hub registry is suported for now")
	}

	// Format the http address, coinciding with the image uri
	httpAddr := fmt.Sprintf("www.%s", s.ShubURI.String())

	// Create the request, add headers context
	url := url.URL{
		Scheme: "https",
		Host:   strings.Split(httpAddr, `/`)[0],     //split url to match format, first half
		Path:   strings.SplitN(httpAddr, `/`, 2)[1], //second half
	}

	req, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", useragent.Value)

	// Do the request, if status isn't success, return error
	res, err := sc.Do(req)
	sylog.Debugf("response: %v\n", res)

	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		err = errors.New(res.Status)
		return err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, &s.ShubAPIResponse)
	sylog.Debugf("manifest: %v\n", s.ShubAPIResponse.Image)
	if err != nil {
		return err
	}

	return nil
}

// CleanUp removes any tmpfs owned by the ShubClient on the filesystem
func (s *ShubClient) CleanUp() {
	os.RemoveAll(s.Bundle.Path)
}
