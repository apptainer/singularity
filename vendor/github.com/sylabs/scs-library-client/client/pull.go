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
	"net/url"
	"strings"

	jsonresp "github.com/sylabs/json-resp"
)

// DownloadImage will retrieve an image from the Container Library, saving it
// into the specified io.Writer. The timeout value for this operation is set
// within the context. It is recommended to use a large value (ie. 1800 seconds)
// to prevent timeout when downloading large images.
func (c *Client) DownloadImage(ctx context.Context, w io.Writer, arch, path, tag string, callback func(int64, io.Reader, io.Writer) error) error {

	if arch != "" && !c.apiAtLeast(ctx, APIVersionV2ArchTags) {
		c.Logger.Logf("This library does not support architecture specific tags")
		c.Logger.Logf("The image returned may not be the requested architecture")
	}

	if strings.Contains(path, ":") {
		return fmt.Errorf("malformed image path: %s", path)
	}

	if tag == "" {
		tag = "latest"
	}

	apiPath := fmt.Sprintf("/v1/imagefile/%s:%s", path, tag)
	apiURL, err := url.Parse(apiPath)
	if err != nil {
		return fmt.Errorf("error constructing API url: %v", err)
	}
	q := url.Values{}
	q.Add("arch", arch)
	apiURL.RawQuery = q.Encode()

	c.Logger.Logf("Pulling from URL: %s", apiURL.String())

	req, err := c.newRequest(http.MethodGet, apiURL.Path, apiURL.RawQuery, nil)
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
