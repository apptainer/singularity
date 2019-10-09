// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"context"
	"net/http"

	jsonresp "github.com/sylabs/json-resp"
)

// GetStatus gets the status of a build from the Build Service by build ID
func (c *Client) GetStatus(ctx context.Context, buildID string) (bi BuildInfo, err error) {
	req, err := c.newRequest(http.MethodGet, "/v1/build/"+buildID, nil)
	if err != nil {
		return
	}
	req = req.WithContext(ctx)

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	err = jsonresp.ReadResponse(res.Body, &bi)
	return
}
