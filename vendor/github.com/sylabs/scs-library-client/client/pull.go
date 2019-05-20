// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	jsonresp "github.com/sylabs/json-resp"
)

// DownloadImage will retrieve an image from the Container Library, saving it
// into the specified io.Writer. The timeout value for this operation is set
// within the context. It is recommended to use a large value (ie. 1800 seconds)
// to prevent timeout when downloading large images.
func (c *Client) DownloadImage(ctx context.Context, w io.Writer, path, tag string, callback func(int64, io.Reader, io.Writer) error) error {

	if strings.Contains(path, ":") {
		return fmt.Errorf("malformed image path: %s", path)
	}

	if tag == "" {
		tag = "latest"
	}

	url := fmt.Sprintf("/v1/imagefile/%s:%s", path, tag)

	c.Logger.Logf("Pulling from URL: %s", url)

	req, err := c.newRequest(http.MethodGet, url, "", nil)
	if err != nil {
		return err
	}

	res, err := c.HTTPClient.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return fmt.Errorf("requested image was not found in the library")
	}

	if res.StatusCode != http.StatusOK {
		err := jsonresp.ReadError(res.Body)
		if err != nil {
			return fmt.Errorf("download did not succeed: %v", err)
		}
		return fmt.Errorf("unexpected http status code: %d", res.StatusCode)
	}

	c.Logger.Logf("OK response received, beginning body download")

	if callback != nil {
		err = callback(res.ContentLength, res.Body, w)
	} else {
		_, err = io.Copy(w, res.Body)
	}
	if err != nil {
		return err
	}

	c.Logger.Logf("Download complete")

	return nil

}
