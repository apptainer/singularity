// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/globalsign/mgo/bson"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	library "github.com/singularityware/singularity/src/pkg/library/client"
	"github.com/singularityware/singularity/src/pkg/sylog"
)

// RequestData contains the info necessary for submitting a build to a remote service
type RequestData struct {
	Definition `json:"definition"`
	// LibraryRef is an optional Library reference to push image after build
	LibraryRef string `json:"libraryRef"`
	// CallbackURL is an optional HTTP callback URL to be called on build completion
	CallbackURL string `json:"callbackURL"`
}

// ResponseData contains the details of an individual build
type ResponseData struct {
	ID            bson.ObjectId `json:"id"`
	CreatedBy     string        `json:"createdBy"`
	SubmitTime    time.Time     `json:"submitTime"`
	StartTime     *time.Time    `json:"startTime,omitempty" bson:",omitempty"`
	IsComplete    bool          `json:"isComplete"`
	CompleteTime  *time.Time    `json:"completeTime,omitempty"`
	ImageSize     int64         `json:"imageSize,omitempty"`
	ImageChecksum string        `json:"imageChecksum,omitempty"`
	Definition    Definition    `json:"definition"`
	CallbackURL   string        `json:"callbackURL"`
	ImageURL      string        `json:"imageURL,omitempty" bson:"-"`
	WSURL         string        `json:"wsURL,omitempty" bson:"-"`
}

// RemoteBuilder contains the build request and response
type RemoteBuilder struct {
	Client     http.Client
	ImageRef   string
	Definition Definition
	IsDetached bool
	HTTPAddr   string
	AuthHeader string
}

// NewRemoteBuilder creates a RemoteBuilder with the specified details.
func NewRemoteBuilder(imageRef string, d Definition, isDetached bool, httpAddr, authToken string) (rb *RemoteBuilder) {
	rb = &RemoteBuilder{
		Client: http.Client{
			Timeout: 30 * time.Second,
		},
		ImageRef:   imageRef,
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

	// Check if the library reference is valid.
	if !library.IsLibraryPushRef(rb.ImageRef) {
		return fmt.Errorf("Not a valid library reference: %s", rb.ImageRef)
	}

	_, _, container, _ := library.ParseLibraryRef(rb.ImageRef)
	wd, err := os.Getwd()
	if err != nil {
		sylog.Errorf("%v", err)
	}

	imagePath := wd + "/" + container

	// Send build request to Remote Build Service
	rd, err := rb.doBuildRequest(ctx)
	if err != nil {
		err = errors.Wrap(err, "failed to post request to remote build service")
		sylog.Warningf("%v\n", err)
		return
	}

	fmt.Printf("Build submited with ID:\t%s\n", rd.ID)

	// If we're doing an attached build, stream output and then download the resulting file
	if !rb.IsDetached {
		err = rb.streamOutput(ctx, rd.WSURL)
		if err != nil {
			err = errors.Wrap(err, "failed to stream output from remote build service")
			sylog.Warningf("%v\n", err)
			return
		}

		rd, err = rb.doStatusRequest(ctx, rd.ID)
		if err != nil {
			err = errors.Wrap(err, "failed to get status from remote build service")
			sylog.Warningf("%v\n", err)
			return
		}

		// Retrieve the built image file
		authToken := strings.TrimPrefix(rb.AuthHeader, "Bearer ")
		err = library.DownloadImage(imagePath, rb.ImageRef, "https://library-test.sylabs.io/library", false, authToken)
		if err != nil {
			sylog.Fatalf("%v\n", err)
			return
		}
	}

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
		// Check if context has expired
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Read from websocket
		mt, msg, err := c.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				return nil
			}
			return err
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
func (rb *RemoteBuilder) doBuildRequest(ctx context.Context) (rd ResponseData, err error) {
	b, err := json.Marshal(RequestData{
		Definition: rb.Definition,
		LibraryRef: rb.ImageRef,
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
