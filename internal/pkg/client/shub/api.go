// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package shub

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/sylabs/singularity/pkg/sylog"
	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
)

const (
	defaultRegistry string = `https://singularity-hub.org`
	shubAPIRoute    string = "/api/container/"
	// URINotSupported if we are using a non default registry error out for now
	URINotSupported string = "Only the default Singularity Hub registry is supported for now"
)

// URI stores the various components of a singularityhub URI
type URI struct {
	registry  string
	user      string
	container string
	tag       string
	digest    string
}

func (s *URI) String() string {
	return s.registry + s.user + "/" + s.container + s.tag + s.digest
}

// APIResponse holds the information returned from the Shub API
type APIResponse struct {
	Image   string `json:"image"`
	Name    string `json:"name"`
	Tag     string `json:"tag"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

// GetManifest will return the image manifest for a container uri
// from Singularity Hub.
func GetManifest(uri URI, noHTTPS bool) (APIResponse, error) {
	// Create a new http Hub client
	httpc := http.Client{
		Timeout: 30 * time.Second,
	}

	if uri.registry != defaultRegistry+shubAPIRoute {
		uri.registry = "https://" + uri.registry
	}

	// Create the request, add headers context
	url, err := url.Parse(uri.registry + uri.user + "/" + uri.container + uri.tag + uri.digest)
	if err != nil {
		return APIResponse{}, err
	}

	if noHTTPS {
		url.Scheme = "http"
	}

	req, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		return APIResponse{}, err
	}
	req.Header.Set("User-Agent", useragent.Value())

	sylog.Debugf("shub request: %s", req.URL.String())

	// Do the request, if status isn't success, return error
	res, err := httpc.Do(req)
	if res == nil {
		return APIResponse{}, fmt.Errorf("no response received from singularity hub")
	}
	if res.StatusCode == http.StatusNotFound {
		return APIResponse{}, fmt.Errorf("the requested manifest was not found in singularity hub")
	}
	sylog.Debugf("%s response received, beginning manifest download\n", res.Status)

	if err != nil {
		return APIResponse{}, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		err = errors.New(res.Status)
		return APIResponse{}, err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return APIResponse{}, err
	}

	var manifest APIResponse

	err = json.Unmarshal(body, &manifest)
	sylog.Debugf("manifest image name: %v\n", manifest.Name)

	return manifest, err
}
