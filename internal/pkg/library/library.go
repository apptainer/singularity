// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package library

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sylabs/scs-library-client/client"
)

const defaultTag = "latest"

type progressCallback func(int64, io.Reader, io.Writer) error

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
func DownloadImage(ctx context.Context, c *client.Client, imagePath, arch, libraryRef string, callback progressCallback) error {
	// reassemble "stripped" library ref for scs-library-client
	validLibraryRef := "library:///" + libraryRef

	// parse library ref
	r, err := client.Parse(validLibraryRef)
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
		// delete incomplete image file in the event of failure
		os.Remove(imagePath)

		return fmt.Errorf("error downloading image: %v", err)
	}

	return nil
}

// DownloadImageNoProgress downloads an image from the library without
// displaying a progress bar while doing so
func DownloadImageNoProgress(ctx context.Context, c *client.Client, imagePath, arch, libraryRef string) error {
	return DownloadImage(ctx, c, imagePath, arch, libraryRef, nil)
}

// SearchLibrary searches the library and outputs results to stdout
func SearchLibrary(ctx context.Context, c *client.Client, value string) error {
	if len(value) < 3 {
		return fmt.Errorf("bad query '%s'. You must search for at least 3 characters", value)
	}

	searchSpec := map[string]string{
		"value": value,
	}

	results, err := c.Search(ctx, searchSpec)
	if err != nil {
		return err
	}

	numEntities := len(results.Entities)
	numCollections := len(results.Collections)
	numContainers := len(results.Containers)

	if numEntities > 0 {
		fmt.Printf("Found %d users for '%s'\n", numEntities, value)
		for _, ent := range results.Entities {
			fmt.Printf("\t%s\n", ent.LibraryURI())
		}
		fmt.Printf("\n")
	} else {
		fmt.Printf("No users found for '%s'\n\n", value)
	}

	if numCollections > 0 {
		fmt.Printf("Found %d collections for '%s'\n", numCollections, value)
		for _, col := range results.Collections {
			fmt.Printf("\t%s\n", col.LibraryURI())
		}
		fmt.Printf("\n")
	} else {
		fmt.Printf("No collections found for '%s'\n\n", value)
	}

	if numContainers > 0 {
		fmt.Printf("Found %d containers for '%s'\n", numContainers, value)
		for _, con := range results.Containers {
			fmt.Printf("\t%s\n", con.LibraryURI())
			if len(con.ImageTags) != 0 {
				fmt.Printf("\t\tTags: %s\n", con.TagList())
			} else if len(con.Images) > 0 {
				fmt.Printf("\t\tImage ID: %s (no tag)\n", con.Images)
			}
			fmt.Printf("\n")
		}
		fmt.Printf("\n")

	} else {
		fmt.Printf("No containers found for '%s'\n\n", value)
	}

	return nil
}
