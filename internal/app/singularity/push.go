// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/sylabs/scs-library-client/client"
	"github.com/sylabs/sif/pkg/sif"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/signing"
	pb "gopkg.in/cheggaaa/pb.v1"
)

var (
	// ErrLibraryUnsigned indicated that the image intended to be used is
	// not signed, nor has an override for requiring a signature been provided
	ErrLibraryUnsigned = errors.New("image is not signed")
)

type progressCallback struct {
	bar *pb.ProgressBar
	r   io.Reader
}

func (c *progressCallback) InitUpload(totalSize int64, r io.Reader) {
	// create and start bar
	c.bar = pb.New64(totalSize).SetUnits(pb.U_BYTES)
	c.bar.ShowTimeLeft = true
	c.bar.ShowSpeed = true

	c.bar.Start()
	c.r = c.bar.NewProxyReader(r)
}

func (c *progressCallback) GetReader() io.Reader {
	return c.r
}

func (c *progressCallback) Finish() {
	c.bar.Finish()
}

// LibraryPush will upload the image specified by file to the library specified by libraryURI.
// Before uploading, the image will be checked for a valid signature, unless specified not to by the
// unauthenticated bool
func LibraryPush(ctx context.Context, file, dest, authToken, libraryURI, keyServerURL, remoteWarning string, unauthenticated bool) error {
	// Push to library requires a valid authToken
	if authToken == "" {
		return fmt.Errorf("couldn't push image to library: %v", remoteWarning)
	}

	if _, err := os.Stat(file); os.IsNotExist(err) {
		return fmt.Errorf("unable to open: %v: %v", file, err)
	}

	arch, err := sifArch(file)
	if err != nil {
		return err
	}

	if !unauthenticated {
		// check if the container is signed
		imageSigned, err := signing.IsSigned(ctx, file, keyServerURL, authToken)
		if err != nil {
			// err will be: "unable to verify container: %v", err
			sylog.Warningf("%v", err)
		}

		// if its not signed, print a warning
		if !imageSigned {
			return ErrLibraryUnsigned
		}
	} else {
		sylog.Warningf("Skipping container verifying")
	}

	libraryClient, err := client.NewClient(&client.Config{
		BaseURL:   libraryURI,
		AuthToken: authToken,
	})
	if err != nil {
		return fmt.Errorf("error initializing library client: %v", err)
	}

	// split library ref into components
	r, err := client.Parse(dest)
	if err != nil {
		return fmt.Errorf("error parsing destination: %v", err)
	}

	// open image for uploading
	f, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("error opening image %s for reading: %v", file, err)
	}
	defer f.Close()

	return libraryClient.UploadImage(ctx, f, r.Host+r.Path, arch, r.Tags, "No Description", &progressCallback{})
}

func sifArch(filename string) (string, error) {
	fimg, err := sif.LoadContainer(filename, true)
	if err != nil {
		return "", fmt.Errorf("unable to open: %v: %v", filename, err)
	}
	arch := sif.GetGoArch(string(fimg.Header.Arch[:sif.HdrArchLen-1]))
	if arch == "unknown" {
		return arch, fmt.Errorf("unknown architecture in SIF file")
	}
	return arch, nil
}
