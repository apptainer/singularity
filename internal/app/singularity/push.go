// Copyright (c) 2020, Control Command Inc. All rights reserved.
// Copyright (c) 2019-2020, Sylabs Inc. All rights reserved.
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

	keyclient "github.com/sylabs/scs-key-client/client"
	"github.com/sylabs/scs-library-client/client"
	"github.com/sylabs/sif/pkg/sif"
	"github.com/sylabs/singularity/pkg/sylog"
	"github.com/vbauerster/mpb/v4"
	"github.com/vbauerster/mpb/v4/decor"
)

var (
	// ErrLibraryUnsigned indicated that the image intended to be used is
	// not signed, nor has an override for requiring a signature been provided
	ErrLibraryUnsigned = errors.New("image is not signed")
)

// LibraryPushSpec describes how a source image file should be pushed to a library server
type LibraryPushSpec struct {
	// SourceFile is the path to the container image to be pushed to the library
	SourceFile string
	// DestRef is the destination reference that the container image will be pushed to in the library
	DestRef string
	// Description is an optional string that describes the container image
	Description string
	// AllowUnsigned must be set to true to allow push of an unsigned container image to succeed
	AllowUnsigned bool
}

type progressCallback struct {
	bar *mpb.Bar
	r   io.Reader
}

func (c *progressCallback) InitUpload(totalSize int64, r io.Reader) {
	// create bar
	p := mpb.New()
	c.bar = p.AddBar(totalSize,
		mpb.PrependDecorators(
			decor.Counters(decor.UnitKiB, "%.1f / %.1f"),
		),
		mpb.AppendDecorators(
			decor.Percentage(),
			decor.AverageSpeed(decor.UnitKiB, " % .1f "),
			decor.AverageETA(decor.ET_STYLE_GO),
		),
	)
	c.r = c.bar.ProxyReader(r)
}

func (c *progressCallback) GetReader() io.Reader {
	return c.r
}

func (c *progressCallback) Finish() {
}

// LibraryPush will upload an image file according to the provided LibraryPushSpec
// Before uploading, the image will be checked for a valid signature unless AllowUnsigned is true
func LibraryPush(ctx context.Context, pushSpec LibraryPushSpec, libraryConfig *client.Config, keyConfig *keyclient.Config) error {
	if _, err := os.Stat(pushSpec.SourceFile); os.IsNotExist(err) {
		return fmt.Errorf("unable to open: %v: %v", pushSpec.SourceFile, err)
	}

	arch, err := sifArch(pushSpec.SourceFile)
	if err != nil {
		return err
	}

	if !pushSpec.AllowUnsigned {
		// Check if the container has a valid signature.
		if err := Verify(ctx, pushSpec.SourceFile, OptVerifyUseKeyServer(keyConfig)); err != nil {
			sylog.Warningf("%v", err)
			return ErrLibraryUnsigned
		}
	} else {
		sylog.Warningf("Skipping container verification")
	}

	libraryClient, err := client.NewClient(libraryConfig)
	if err != nil {
		return fmt.Errorf("error initializing library client: %v", err)
	}

	// split library ref into components
	r, err := client.Parse(pushSpec.DestRef)
	if err != nil {
		return fmt.Errorf("error parsing destination: %v", err)
	}

	// open image for uploading
	f, err := os.Open(pushSpec.SourceFile)
	if err != nil {
		return fmt.Errorf("error opening image %s for reading: %v", pushSpec.SourceFile, err)
	}
	defer f.Close()

	return libraryClient.UploadImage(ctx, f, r.Host+r.Path, arch, r.Tags, pushSpec.Description, &progressCallback{})
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
