/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.
  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package build

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"gopkg.in/mgo.v2/bson"
)

// RequestData contains the info necessary for submitting a build to a remote service
type RequestData struct {
	Definition `json:"definition"`
	IsDetached bool `json:"isDetached"`
}

// ResponseData contains the details of an individual build
type ResponseData struct {
	ID           bson.ObjectId `json:"id"`
	SubmitTime   time.Time     `json:"submitTime"`
	IsComplete   bool          `json:"isComplete"`
	CompleteTime *time.Time    `json:"completeTime,omitempty"`
	IsDetached   bool          `json:"isDetached"`
	WSURL        string        `json:"wsURL,omitempty"`
	ImageURL     string        `json:"imageURL,omitempty"`
	Definition   Definition    `json:"definition"`
}

// RemoteBuilder contains the build request and response
type RemoteBuilder struct {
	Client     http.Client
	ImagePath  string
	Definition Definition
	IsDetached bool
	HTTPAddr   string
	AuthHeader string
}

// NewRemoteBuilder creates a RemoteBuilder with the specified details.
func NewRemoteBuilder(imagePath string, d Definition, isDetached bool, httpAddr, authToken string) (rb *RemoteBuilder) {
	rb = &RemoteBuilder{
		Client: http.Client{
			Timeout: 30 * time.Second,
		},
		ImagePath:  imagePath,
		Definition: d,
		IsDetached: isDetached,
		HTTPAddr:   httpAddr,
	}
	if authToken != "" {
		rb.AuthHeader = fmt.Sprintf("Bearer %s", authToken)
	}

	return
}

// Build is responsible for making the request via the REST API to the remote builder
func (rb *RemoteBuilder) Build(ctx context.Context) (err error) {
	// Open the image file, since there isn't much point in doing the remote build if we can't write
	// out the image.
	f, err := os.OpenFile(rb.ImagePath, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		err = errors.Wrap(err, "failed to open image file")
		return
	}
	defer f.Close()

	// Send build request to Remote Build Service
	rd, err := rb.doBuildRequest(ctx, rb.Definition, rb.IsDetached)
	if err != nil {
		err = errors.Wrap(err, "failed to post request to remote build service")
		sylog.Warningf("%v", err)
		return
	}

	// If we're doing an attached build, stream output and then download the resulting file
	if !rb.IsDetached {
		err = rb.streamOutput(ctx, rd.WSURL)
		if err != nil {
			err = errors.Wrap(err, "failed to stream output from remote build service")
			sylog.Warningf("%v", err)
			return
		}
	}

	// TODO: if the build is detached, do we poll status until the build is complete? Return immediately?
	rd, err = rb.doStatusRequest(ctx, rd.ID)
	if err != nil {
		err = errors.Wrap(err, "failed to get status from remote build service")
		sylog.Warningf("%v", err)
		return
	}

	// Retrieve the built image file
	err = rb.doPullRequest(ctx, rd.ImageURL, f)

	return
}

// streamOutput attaches via websocket and streams output to the console
func (rb *RemoteBuilder) streamOutput(ctx context.Context, url string) (err error) {
	h := http.Header{}
	if rb.AuthHeader != "" {
		h.Set("Authorization", rb.AuthHeader)
	}
	c, _, err := websocket.DefaultDialer.Dial(url, h)
	if err != nil {
		return err
	}
	defer c.Close()

	for {
		// Read from websocket
		mt, msg, err := c.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				return nil
			} else {
				return err
			}
		}

		// Print to terminal
		switch mt {
		case websocket.TextMessage:
			fmt.Printf("%s", msg)
		case websocket.BinaryMessage:
			fmt.Print("Ignoring binary message")
		}
	}
}

// doBuildRequest creates a new build on a Remote Build Service
func (rb *RemoteBuilder) doBuildRequest(ctx context.Context, d Definition, isDetached bool) (rd ResponseData, err error) {
	b, err := json.Marshal(RequestData{
		Definition: d,
		IsDetached: isDetached,
	})
	if err != nil {
		return
	}

	url := url.URL{Scheme: "http", Host: rb.HTTPAddr, Path: "/v1/build"}
	req, err := http.NewRequest(http.MethodPost, url.String(), bytes.NewReader(b))
	if err != nil {
		return
	}
	req = req.WithContext(ctx)
	if rb.AuthHeader != "" {
		req.Header.Set("Authorization", rb.AuthHeader)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := rb.Client.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		err = errors.New(res.Status)
		return
	}

	err = json.NewDecoder(res.Body).Decode(&rd)
	return
}

// doStatusRequest gets the status of a build from the Remote Build Service
func (rb *RemoteBuilder) doStatusRequest(ctx context.Context, id bson.ObjectId) (rd ResponseData, err error) {
	url := url.URL{Scheme: "http", Host: rb.HTTPAddr, Path: "/v1/build/" + id.Hex()}
	req, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		return
	}
	req = req.WithContext(ctx)
	if rb.AuthHeader != "" {
		req.Header.Set("Authorization", rb.AuthHeader)
	}

	res, err := rb.Client.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		err = errors.New(res.Status)
		return
	}

	err = json.NewDecoder(res.Body).Decode(&rd)
	return
}

// doPullRequest retrieves an image from the specified URL and saves it to the specified path
func (rb *RemoteBuilder) doPullRequest(ctx context.Context, url string, r io.Writer) (err error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return
	}
	req = req.WithContext(ctx)
	if rb.AuthHeader != "" {
		req.Header.Set("Authorization", rb.AuthHeader)
	}

	res, err := rb.Client.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		err = errors.New(res.Status)
		return
	}

	_, err = io.Copy(r, res.Body)
	if err != nil {
		err = errors.Wrap(err, "failed to write image")
	}
	return
}
