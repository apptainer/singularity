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
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	buildclient "github.com/sylabs/scs-build-client/client"
	client "github.com/sylabs/scs-library-client/client"
	library "github.com/sylabs/singularity/internal/pkg/library"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	types "github.com/sylabs/singularity/pkg/build/legacy"
	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
)

// CloudURI holds the URI of the Library web front-end.
const CloudURI = "https://cloud.sylabs.io"

// RemoteBuilder contains the build request and response
type RemoteBuilder struct {
	BuildClient         *buildclient.Client
	ImagePath           string
	LibraryURL          string
	Definition          types.Definition
	BuilderURL          *url.URL
	AuthToken           string
	Force               bool
	IsDetached          bool
	BuilderRequirements map[string]string
}

// New creates a RemoteBuilder with the specified details.
func New(imagePath, libraryURL string, d types.Definition, isDetached, force bool, builderAddr, authToken string) (rb *RemoteBuilder, err error) {
	bc, err := buildclient.New(&buildclient.Config{
		BaseURL:   builderAddr,
		AuthToken: authToken,
		UserAgent: useragent.Value(),
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	})
	if err != nil {
		return nil, err
	}

	return &RemoteBuilder{
		BuildClient: bc,
		ImagePath:   imagePath,
		Force:       force,
		LibraryURL:  libraryURL,
		Definition:  d,
		IsDetached:  isDetached,
		AuthToken:   authToken,
		// TODO - set CPU architecture, RAM requirements, singularity version, etc.
		//BuilderRequirements: map[string]string{},
	}, nil
}

// Build is responsible for making the request via scs-build-client to the builder
func (rb *RemoteBuilder) Build(ctx context.Context) (err error) {
	var libraryRef string

	if strings.HasPrefix(rb.ImagePath, "library://") {
		// Image destination is Library.
		libraryRef = rb.ImagePath
	}

	if libraryRef != "" && !client.IsLibraryPushRef(libraryRef) {
		return fmt.Errorf("invalid library reference: %s", rb.ImagePath)
	}

	br := buildclient.BuildRequest{
		LibraryRef:          libraryRef,
		LibraryURL:          rb.LibraryURL,
		DefinitionRaw:       rb.Definition.Raw,
		BuilderRequirements: rb.BuilderRequirements,
	}

	bi, err := rb.BuildClient.Submit(ctx, br)
	if err != nil {
		return errors.Wrap(err, "failed to post request to remote build service")
	}
	sylog.Debugf("Build response - id: %s, libref: %s", bi.ID, bi.LibraryRef)

	// If we're doing an detached build, print help on how to download the image
	libraryRefRaw := strings.TrimPrefix(bi.LibraryRef, "library://")
	if rb.IsDetached {
		fmt.Printf("Build submitted! Once it is complete, the image can be retrieved by running:\n")
		fmt.Printf("\tsingularity pull --library %s library://%s\n\n", bi.LibraryURL, libraryRefRaw)
		fmt.Printf("Alternatively, you can access it from a browser at:\n\t%s/library/%s\n", CloudURI, libraryRefRaw)
		return nil
	}

	// We're doing an attached build, stream output and then download the resulting file
	var outputLogger stdoutLogger
	err = rb.BuildClient.GetOutput(ctx, bi.ID, outputLogger)
	if err != nil {
		return errors.Wrap(err, "failed to stream output from remote build service")
	}

	// Get build status
	bi, err = rb.BuildClient.GetStatus(ctx, bi.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get status from remote build service")
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
		f, err := os.OpenFile(rb.ImagePath, os.O_CREATE|os.O_RDWR, 0777)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to open file %s for writing", rb.ImagePath))
		}
		defer f.Close()

		c, err := client.NewClient(&client.Config{
			BaseURL:   bi.LibraryURL,
			AuthToken: rb.AuthToken,
		})
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("error initializing library client: %v", err))
		}

		if err = library.DownloadImageNoProgress(ctx, c, rb.ImagePath, bi.LibraryRef); err != nil {
			return errors.Wrap(err, "failed to pull image file")
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
