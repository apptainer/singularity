// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the LICENSE.md file
// distributed with the sources of this project regarding your rights to use or distribute this
// software.

package client

import (
	"context"
	"net/http"

	jsonresp "github.com/sylabs/json-resp"
)

const pathVersion = "/version"

// VersionInfo contains version information.
type VersionInfo struct {
	Version string `json:"version"`
}

// GetVersion gets version information from the build service. The context
// controls the lifetime of the request.
func (c *Client) GetVersion(ctx context.Context) (vi VersionInfo, err error) {
	req, err := c.newRequest(http.MethodGet, pathVersion, nil)
	if err != nil {
		return VersionInfo{}, err
	}

	res, err := c.HTTPClient.Do(req.WithContext(ctx))
	if err != nil {
		return VersionInfo{}, err
	}
	defer res.Body.Close()

	if err := jsonresp.ReadResponse(res.Body, &vi); err != nil {
		return VersionInfo{}, err
	}
	return vi, nil
}
