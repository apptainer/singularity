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

	golog "github.com/go-log/log"
	keyclient "github.com/sylabs/scs-key-client/client"
	"github.com/sylabs/scs-library-client/client"
	"github.com/sylabs/sif/pkg/sif"
	"github.com/sylabs/singularity/pkg/sylog"
	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
	"github.com/vbauerster/mpb/v4"
	"github.com/vbauerster/mpb/v4/decor"
)

var (
	// ErrLibraryUnsigned indicated that the image intended to be used is
	// not signed, nor has an override for requiring a signature been provided
	ErrLibraryUnsigned = errors.New("image is not signed")
)

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
		// Check if the container has a valid signature.
		c := keyclient.Config{
			BaseURL:   keyServerURL,
			AuthToken: authToken,
			UserAgent: useragent.Value(),
		}
		if err := Verify(ctx, file, OptVerifyUseKeyServer(&c)); err != nil {
			sylog.Warningf("%v", err)
			return ErrLibraryUnsigned
		}
	} else {
		sylog.Warningf("Skipping container verifying")
	}

	libraryClient, err := client.NewClient(&client.Config{
		BaseURL:   libraryURI,
		AuthToken: authToken,
		Logger:    (golog.Logger)(sylog.DebugLogger{}),
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
