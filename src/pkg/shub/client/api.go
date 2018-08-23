// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/user-agent"
)

const (
	defaultRegistry string = `singularity-hub.org`
	shubAPIRoute    string = "/api/container/"
	//URINotSupported if we are using a non default registry error out for now
	URINotSupported string = "Only the default Singularity Hub registry is suported for now"
)

// ShubURI stores the various components of a singularityhub URI
type ShubURI struct {
	registry  string
	user      string
	container string
	tag       string
	digest    string
}

func (s *ShubURI) String() string {
	return s.registry + s.user + "/" + s.container + s.tag + s.digest
}

// ShubAPIResponse holds the information returned from the Shub API
type ShubAPIResponse struct {
	Image   string `json:"image"`
	Name    string `json:"name"`
	Tag     string `json:"tag"`
	Version string `json:"version"`
}

// getManifest will return the image manifest for a container uri
// from Singularity Hub.
func getManifest(uri ShubURI) (manifest ShubAPIResponse, err error) {

	// Create a new Singularity Hub client
	sc := http.Client{
		Timeout: 30 * time.Second,
	}

	if uri.registry != fmt.Sprintf("%s%s", defaultRegistry, shubAPIRoute) {
		return ShubAPIResponse{}, errors.New(URINotSupported)
	}

	// Format the http address, coinciding with the image uri
	httpAddr := fmt.Sprintf("www.%s", uri.String())

	// Create the request, add headers context
	url := url.URL{
		Scheme: "https",
		Host:   strings.Split(httpAddr, `/`)[0],     //split url to match format, first half
		Path:   strings.SplitN(httpAddr, `/`, 2)[1], //second half
	}

	req, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		return ShubAPIResponse{}, err
	}
	req.Header.Set("User-Agent", useragent.Value)

	// Do the request, if status isn't success, return error
	res, err := sc.Do(req)
	sylog.Debugf("response: %v\n", res)

	if err != nil {
		return ShubAPIResponse{}, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		err = errors.New(res.Status)
		return ShubAPIResponse{}, err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return ShubAPIResponse{}, err
	}

	err = json.Unmarshal(body, &manifest)
	sylog.Debugf("manifest: %v\n", manifest.Image)
	if err != nil {
		return ShubAPIResponse{}, err
	}

	return
}
