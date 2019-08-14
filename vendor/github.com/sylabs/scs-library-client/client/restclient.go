// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	jsonresp "github.com/sylabs/json-resp"
)

var (
	// ErrNotFound is returned by when a resource is not found (http status 404)
	ErrNotFound = errors.New("not found")
)

func (c *Client) apiGet(ctx context.Context, path string) (objJSON []byte, err error) {
	c.Logger.Logf("apiGet calling %s", path)
	return c.doGETRequest(ctx, path)
}

func (c *Client) apiCreate(ctx context.Context, url string, o interface{}) (objJSON []byte, err error) {
	c.Logger.Logf("apiCreate calling %s", url)
	return c.doPOSTRequest(ctx, url, o)
}

func (c *Client) apiUpdate(ctx context.Context, url string, o interface{}) (objJSON []byte, err error) {
	c.Logger.Logf("apiUpdate calling %s", url)
	return c.doPUTRequest(ctx, url, o)
}

func (c *Client) doGETRequest(ctx context.Context, path string) (objJSON []byte, err error) {
	return c.commonRequestHandler(ctx, "GET", path, nil, []int{http.StatusOK})
}

func (c *Client) doPUTRequest(ctx context.Context, path string, o interface{}) (objJSON []byte, err error) {
	return c.commonRequestHandler(ctx, "PUT", path, o, []int{http.StatusOK, http.StatusNoContent})
}

func (c *Client) doPOSTRequest(ctx context.Context, path string, o interface{}) (objJSON []byte, err error) {
	return c.commonRequestHandler(ctx, "POST", path, o, []int{http.StatusOK, http.StatusCreated})
}

func (c *Client) commonRequestHandler(ctx context.Context, method string, path string, o interface{}, acceptedStatusCodes []int) (objJSON []byte, err error) {
	var payload io.Reader

	// only PUT and POST methods
	if method != "GET" && method != "DELETE" {
		s, err := json.Marshal(o)
		if err != nil {
			return []byte{}, fmt.Errorf("error encoding object to JSON:\n\t%v", err)
		}
		payload = bytes.NewBuffer(s)
	}

	// split url containing query into component pieces (path and raw query)
	u, err := url.Parse(path)
	if err != nil {
		return []byte{}, fmt.Errorf("error parsing url:\n\t%v", err)
	}

	req, err := c.newRequest(method, u.Path, u.RawQuery, payload)
	if err != nil {
		return []byte{}, fmt.Errorf("error creating %s request:\n\t%v", method, err)
	}

	res, err := c.HTTPClient.Do(req.WithContext(ctx))
	if err != nil {
		return []byte{}, fmt.Errorf("error making request to server:\n\t%v", err)
	}
	defer res.Body.Close()

	// check http status code
	if res.StatusCode == http.StatusNotFound {
		return []byte{}, ErrNotFound
	}
	if !isValidStatusCode(res.StatusCode, acceptedStatusCodes) {
		err := jsonresp.ReadError(res.Body)
		if err != nil {
			return []byte{}, fmt.Errorf("request did not succeed: %v", err)
		}
		return []byte{}, fmt.Errorf("request did not succeed: http status code: %d", res.StatusCode)
	}
	objJSON, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return []byte{}, fmt.Errorf("error reading response from server:\n\t%v", err)
	}
	return objJSON, nil
}

func isValidStatusCode(statusCode int, acceptedStatusCodes []int) bool {
	for _, value := range acceptedStatusCodes {
		if value == statusCode {
			return true
		}
	}
	return false
}
