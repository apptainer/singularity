// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package library

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/sylabs/singularity/pkg/sylog"

	scslibrary "github.com/sylabs/scs-library-client/client"
	"github.com/sylabs/singularity/internal/pkg/client"
)

const defaultTag = "latest"

// Default empty description set by older pushes, which is not informative to display
// Comparison will be lower case as the GUI / code has used different capitalisation through time.
const noDescription = "no description"

// NormalizeLibraryRef strips off leading "library://" prefix, if any, and
// appends the default tag (latest) if none specified.
func NormalizeLibraryRef(libraryRef string) string {
	ir := strings.TrimPrefix(libraryRef, "library://")
	if !strings.Contains(ir, ":") {
		return ir + ":" + defaultTag
	}
	return ir
}

// DownloadImage is a helper function to wrap library image download operation
func DownloadImage(ctx context.Context, c *scslibrary.Client, imagePath, arch, libraryRef string, callback client.ProgressCallback) error {
	// reassemble "stripped" library ref for scs-library-client
	validLibraryRef := "library:///" + libraryRef

	// parse library ref
	r, err := scslibrary.Parse(validLibraryRef)
	if err != nil {
		return fmt.Errorf("error parsing library ref: %v", err)
	}

	// open destination file for writing
	f, err := os.OpenFile(imagePath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0777)
	if err != nil {
		return fmt.Errorf("error opening file %s for writing: %v", imagePath, err)
	}
	defer f.Close()

	var tag string
	if len(r.Tags) > 0 {
		tag = r.Tags[0]
	}

	// call library client to download image
	err = c.DownloadImage(ctx, f, arch, r.Path, tag, callback)
	if err != nil {
		// Delete incomplete image file in the event of failure
		// we get here e.g. if the context is canceled by Ctrl-C
		sylog.Debugf("Cleaning up incomplete download: %s", imagePath)
		if err := os.Remove(imagePath); err != nil {
			sylog.Errorf("Error while removing incomplete download: %v", err)
		}

		return fmt.Errorf("error downloading image: %v", err)
	}

	return nil
}

// DownloadImageNoProgress downloads an image from the library without
// displaying a progress bar while doing so
func DownloadImageNoProgress(ctx context.Context, c *scslibrary.Client, imagePath, arch, libraryRef string) error {
	return DownloadImage(ctx, c, imagePath, arch, libraryRef, nil)
}

// SearchLibrary searches the library and outputs results to stdout
func SearchLibrary(ctx context.Context, c *scslibrary.Client, value, arch string, signed bool) error {
	if len(value) < 3 {
		return fmt.Errorf("bad query '%s'. You must search for at least 3 characters", value)
	}

	searchSpec := map[string]string{
		"value": value,
		"arch":  arch,
	}

	if signed {
		searchSpec["signed"] = "true"
	}

	results, err := c.Search(ctx, searchSpec)
	if err != nil {
		return err
	}

	numImages := len(results.Images)

	if numImages > 0 {
		imageList := []string{}
		for _, img := range results.Images {
			tagSpec := img.Hash
			if len(img.Tags) > 0 {
				tagSpec = strings.Join(img.Tags, ",")
			}
			imageItem := fmt.Sprintf("\tlibrary://%s/%s/%s:%s", img.EntityName, img.CollectionName, img.ContainerName, tagSpec)
			if img.Description != "" && strings.ToLower(img.Description) != noDescription {
				imageItem = imageItem + fmt.Sprintf("\n\t\t%s", img.Description)
			}
			if len(img.Fingerprints) > 0 {
				imageItem = imageItem + fmt.Sprintf("\n\t\tSigned by: %s", strings.Join(img.Fingerprints, ","))
			}
			imageList = append(imageList, imageItem)
		}
		sort.Strings(imageList)
		fmt.Printf("Found %d container images for %s matching %q:\n\n", numImages, arch, value)
		fmt.Println(strings.Join(imageList, "\n\n"))
		fmt.Printf("\n")
	} else {
		fmt.Printf("No container images found for %s matching %q.\n\n", arch, value)
	}

	return nil
}
