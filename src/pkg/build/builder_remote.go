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
	"strings"
	"time"

	"github.com/globalsign/mgo/bson"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/singularityware/singularity/src/pkg/library/client"
	"github.com/singularityware/singularity/src/pkg/sylog"
)

// RequestData contains the info necessary for submitting a build to a remote service
type RequestData struct {
	Definition  `json:"definition"`
	LibraryRef  string `json:"libraryRef"`
	LibraryURL  string `json:"libraryURL"`
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
	WSURL         string        `json:"wsURL,omitempty" bson:"-"`
	LibraryRef    string        `json:"libraryRef"`
	LibraryURL    string        `json:"libraryURL"`
	CallbackURL   string        `json:"callbackURL"`
}

// RemoteBuilder contains the build request and response
type RemoteBuilder struct {
	Client     http.Client
	ImagePath  string
	Force      bool
	LibraryURL string
	Definition Definition
	IsDetached bool
	HTTPAddr   string
	AuthToken  string
}

func (rb *RemoteBuilder) setAuthHeader(h http.Header) {
	if rb.AuthToken != "" {
		h.Set("Authorization", fmt.Sprintf("Bearer %s", rb.AuthToken))
	}
}

// NewRemoteBuilder creates a RemoteBuilder with the specified details.
func NewRemoteBuilder(imagePath, libraryURL string, d Definition, isDetached bool, httpAddr, authToken string) (rb *RemoteBuilder) {
	rb = &RemoteBuilder{
		Client: http.Client{
			Timeout: 30 * time.Second,
		},
		ImagePath:  imagePath,
		LibraryURL: libraryURL,
		Definition: d,
		IsDetached: isDetached,
		HTTPAddr:   httpAddr,
		AuthToken:  authToken,
	}

	return
}

// Build is responsible for making the request via the REST API to the remote builder
func (rb *RemoteBuilder) Build(ctx context.Context) (err error) {
	var libraryRef string

	if strings.HasPrefix(rb.ImagePath, "library://") {
		// Image destination is Library.
		libraryRef = rb.ImagePath
	}

	// Send build request to Remote Build Service
	rd, err := rb.doBuildRequest(ctx, rb.Definition, libraryRef)
	if err != nil {
		err = errors.Wrap(err, "failed to post request to remote build service")
		sylog.Warningf("%v", err)
		return err
	}

	// If we're doing an detached build, print help on how to download the image
	libraryRefRaw := strings.TrimPrefix(rd.LibraryRef, "library://")
	if rb.IsDetached {
		fmt.Printf("Build submitted! Once it is complete, the image can be retrieved by running:\n")
		fmt.Printf("\tsingularity pull --library %v library://%v\n\n", rd.LibraryURL, libraryRefRaw)
		fmt.Printf("Alternatively, you can access it from a browser at:\n\t%v/library/%v\n", rd.LibraryURL, libraryRefRaw)
	}

	// If we're doing an attached build, stream output and then download the resulting file
	if !rb.IsDetached {
		err = rb.streamOutput(ctx, rd.WSURL)
		if err != nil {
			err = errors.Wrap(err, "failed to stream output from remote build service")
			sylog.Warningf("%v", err)
			return err
		}

		// Get build status
		rd, err = rb.doStatusRequest(ctx, rd.ID)
		if err != nil {
			err = errors.Wrap(err, "failed to get status from remote build service")
			sylog.Warningf("%v", err)
			return err
		}

		// If image destination is local file, pull image.
		if !strings.HasPrefix(rb.ImagePath, "library://") {
			err = client.DownloadImage(rb.ImagePath, rd.LibraryRef, rd.LibraryURL, rb.Force, rb.AuthToken)
			if err != nil {
				err = errors.Wrap(err, "failed to pull image file")
				sylog.Warningf("%v", err)
				return err
			}
		}
	}

	return nil
}

// streamOutput attaches via websocket and streams output to the console
func (rb *RemoteBuilder) streamOutput(ctx context.Context, url string) (err error) {
	h := http.Header{}
	rb.setAuthHeader(h)
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
func (rb *RemoteBuilder) doBuildRequest(ctx context.Context, d Definition, libraryRef string) (rd ResponseData, err error) {
	if libraryRef != "" && !client.IsLibraryPushRef(libraryRef) {
		err = fmt.Errorf("invalid library reference: %v", rb.ImagePath)
		sylog.Warningf("%v", err)
		return ResponseData{}, err
	}

	b, err := json.Marshal(RequestData{
		Definition: d,
		LibraryRef: libraryRef,
		LibraryURL: rb.LibraryURL,
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
	rb.setAuthHeader(req.Header)
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
	rb.setAuthHeader(req.Header)

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
