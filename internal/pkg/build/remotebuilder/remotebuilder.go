// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

//
// NOTE: This package uses a different version of the definition struct and
// definition parser than the rest of the image build system in order to maintain
// compatibility with the remote builder.
//

package remotebuilder

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
	jsonresp "github.com/sylabs/json-resp"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	types "github.com/sylabs/singularity/pkg/build/legacy"
	client "github.com/sylabs/singularity/pkg/client/library"
	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
)

// CloudURI holds the URI of the Library web front-end.
const CloudURI = "https://cloud.sylabs.io"

// RemoteBuilder contains the build request and response
type RemoteBuilder struct {
	Client     http.Client
	ImagePath  string
	LibraryURL string
	Definition types.Definition
	BuilderURL *url.URL
	AuthToken  string
	Force      bool
	IsDetached bool
}

func (rb *RemoteBuilder) setAuthHeader(h http.Header) {
	if rb.AuthToken != "" {
		h.Set("Authorization", fmt.Sprintf("Bearer %s", rb.AuthToken))
	}
}

// New creates a RemoteBuilder with the specified details.
func New(imagePath, libraryURL string, d types.Definition, isDetached, force bool, builderAddr, authToken string) (rb *RemoteBuilder, err error) {
	builderURL, err := url.Parse(builderAddr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse builder address")
	}

	rb = &RemoteBuilder{
		Client: http.Client{
			Timeout: 30 * time.Second,
		},
		ImagePath:  imagePath,
		Force:      force,
		LibraryURL: libraryURL,
		Definition: d,
		IsDetached: isDetached,
		BuilderURL: builderURL,
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
		fmt.Printf("Alternatively, you can access it from a browser at:\n\t%v/library/%v\n", CloudURI, libraryRefRaw)
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

		// Do not try to download image if not complete or image size is 0
		if !rd.IsComplete {
			return errors.New("build has not completed")
		}
		if rd.ImageSize <= 0 {
			return errors.New("build image size <= 0")
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
	h.Set("User-Agent", useragent.Value())

	c, resp, err := websocket.DefaultDialer.Dial(url, h)
	if err != nil {
		sylog.Debugf("websocket dial err - %s, partial response: %+v", err, resp)
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
			sylog.Debugf("websocket read message err - %s", err)
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
func (rb *RemoteBuilder) doBuildRequest(ctx context.Context, d types.Definition, libraryRef string) (rd types.ResponseData, err error) {
	if libraryRef != "" && !client.IsLibraryPushRef(libraryRef) {
		err = fmt.Errorf("invalid library reference: %v", rb.ImagePath)
		sylog.Warningf("%v", err)
		return types.ResponseData{}, err
	}

	b, err := json.Marshal(types.RequestData{
		Definition: d,
		LibraryRef: libraryRef,
		LibraryURL: rb.LibraryURL,
	})
	if err != nil {
		return
	}

	req, err := http.NewRequest(http.MethodPost, rb.BuilderURL.String()+"/v1/build", bytes.NewReader(b))
	if err != nil {
		return
	}
	req = req.WithContext(ctx)
	rb.setAuthHeader(req.Header)
	req.Header.Set("User-Agent", useragent.Value())
	req.Header.Set("Content-Type", "application/json")
	sylog.Debugf("Sending build request to %s", req.URL.String())

	res, err := rb.Client.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	err = jsonresp.ReadResponse(res.Body, &rd)
	if err == nil {
		sylog.Debugf("Build response - id: %s, wsurl: %s, libref: %s",
			rd.ID.Hex(), rd.WSURL, rd.LibraryRef)
	}
	return
}

// doStatusRequest gets the status of a build from the Remote Build Service
func (rb *RemoteBuilder) doStatusRequest(ctx context.Context, id bson.ObjectId) (rd types.ResponseData, err error) {
	req, err := http.NewRequest(http.MethodGet, rb.BuilderURL.String()+"/v1/build/"+id.Hex(), nil)
	if err != nil {
		return
	}
	req = req.WithContext(ctx)
	rb.setAuthHeader(req.Header)
	req.Header.Set("User-Agent", useragent.Value())

	res, err := rb.Client.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	err = jsonresp.ReadResponse(res.Body, &rd)
	return
}
