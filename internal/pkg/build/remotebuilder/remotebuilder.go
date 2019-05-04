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
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	buildclient "github.com/sylabs/scs-build-client/client"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	client "github.com/sylabs/singularity/pkg/client/library"
	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
)

// CloudURI holds the URI of the Library web front-end.
const CloudURI = "https://cloud.sylabs.io"

// RemoteBuilder contains the build request and response
type RemoteBuilder struct {
	BuildClient *buildclient.Client
	AuthToken   string
	ImagePath   string
	LibraryURL  string
	Definition  buildclient.Definition
	Force       bool
	IsDetached  bool
}

func New(imagePath, libraryURL string, d buildclient.Definition, isDetached, force bool, builderAddr, authToken string) (rb *RemoteBuilder, err error) {
	// Get a Build Service client.
	c, err := buildclient.New(&buildclient.Config{
		BaseURL:   builderAddr,
		AuthToken: authToken,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		UserAgent: useragent.Value(),
	})
	if err != nil {
		return nil, err
	}
	return &RemoteBuilder{
		BuildClient: c,
		AuthToken:   authToken,
		ImagePath:   imagePath,
		Force:       force,
		LibraryURL:  libraryURL,
		Definition:  d,
		IsDetached:  isDetached,
	}, nil
}

// Build is responsible for making the request via the REST API to the remote builder
func (rb *RemoteBuilder) Build(ctx context.Context) (err error) {
	var libraryRef string

	if strings.HasPrefix(rb.ImagePath, "library://") {
		// Image destination is Library.
		libraryRef = rb.ImagePath
	}

	// Send build request to Build Service
	bi, err := rb.doBuildRequest(ctx, rb.Definition, libraryRef)
	if err != nil {
		err = errors.Wrap(err, "failed to post request to remote build service")
		sylog.Warningf("%v", err)
		return err
	}

	// If we're doing an detached build, print help on how to download the image
	libraryRefRaw := strings.TrimPrefix(bi.LibraryRef, "library://")
	if rb.IsDetached {
		fmt.Printf("Build submitted! Once it is complete, the image can be retrieved by running:\n")
		fmt.Printf("\tsingularity pull --library %v library://%v\n\n", bi.LibraryURL, libraryRefRaw)
		fmt.Printf("Alternatively, you can access it from a browser at:\n\t%v/library/%v\n", CloudURI, libraryRefRaw)
		return nil
	}

	// If we're doing an attached build, stream output and then download the resulting file
	err = rb.streamOutput(ctx, bi.ID)
	if err != nil {
		err = errors.Wrap(err, "failed to stream output from remote build service")
		sylog.Warningf("%v", err)
		return err
	}

	// Get build status
	bi, err = rb.BuildClient.GetStatus(ctx, bi.ID)
	if err != nil {
		err = errors.Wrap(err, "failed to get status from remote build service")
		sylog.Warningf("%v", err)
		return err
	}

	// Do not try to download image if not complete or image size is 0
	if !bi.IsComplete {
		return errors.New("build has not completed")
	}
	if bi.ImageSize <= 0 {
		return errors.New("build image size <= 0")
	}

	// If image destination is local file, pull image.
	if !strings.HasPrefix(rb.ImagePath, "library://") {
		err = client.DownloadImage(rb.ImagePath, bi.LibraryRef, bi.LibraryURL, rb.Force, rb.AuthToken)
		if err != nil {
			err = errors.Wrap(err, "failed to pull image file")
			sylog.Warningf("%v", err)
			return err
		}
	}

	return nil
}

// stdoutLogger implements the buildclient.OutputReader interface and writes
// messages to stdout
type stdoutLogger struct{}

// Read implements the buildclient.OutputReader Read interface, writing messages
// to the console/terminal
func (c stdoutLogger) Read(messageType int, msg []byte) (int, error) {
	// Print to terminal
	switch messageType {
	case websocket.TextMessage:
		fmt.Printf("%s", msg)
	case websocket.BinaryMessage:
		fmt.Print("Ignoring binary message")
	}
	return len(msg), nil
}

// streamOutput attaches via websocket and streams output to the console
func (rb *RemoteBuilder) streamOutput(ctx context.Context, buildID string) error {
	var outputLogger stdoutLogger
	return rb.BuildClient.GetOutput(ctx, buildID, outputLogger)
}

// doBuildRequest creates a new build on a Build Service
func (rb *RemoteBuilder) doBuildRequest(ctx context.Context, d buildclient.Definition, libraryRef string) (bi buildclient.BuildInfo, err error) {
	if libraryRef != "" && !client.IsLibraryPushRef(libraryRef) {
		err = fmt.Errorf("invalid library reference: %v", rb.ImagePath)
		sylog.Warningf("%v", err)
		return buildclient.BuildInfo{}, err
	}

	return rb.BuildClient.Submit(ctx, rb.Definition, libraryRef, rb.LibraryURL)
}
