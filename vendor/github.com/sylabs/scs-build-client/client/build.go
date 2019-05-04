// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	jsonresp "github.com/sylabs/json-resp"
)

// Submit sends a build job to the Build Service. The context controls the
// lifetime of the request.
func (c *Client) Submit(ctx context.Context, d Definition, libraryRef string, libraryURL string) (bi BuildInfo, err error) {

	b, err := json.Marshal(BuildRequest{
		Definition: d,
		LibraryRef: libraryRef,
		LibraryURL: libraryURL,
	})
	if err != nil {
		return
	}

	req, err := c.newRequest(http.MethodPost, "/v1/build", "", bytes.NewReader(b))
	if err != nil {
		return
	}
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	c.Logger.Logf("Sending build request to %s", req.URL.String())

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	err = jsonresp.ReadResponse(res.Body, &bi)
	if err == nil {
		c.Logger.Logf("Build response - id: %s, libref: %s", bi.ID, bi.LibraryRef)
	}
	return
}
