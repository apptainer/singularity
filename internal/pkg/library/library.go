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

// getDownloadTag parses tag(s) from Ref struct
func getDownloadTag(tags []string) string {
	if len(tags) > 0 {
		return tags[0]
	}

	// if library ref does not contain a tag, use "latest" as default
	return defaultTag
}

// ParseLegacyLibraryRef is intended to ensure library refs formatted as
// "library://image:tag" are properly reformatted for passing to
// client.Parse(). Library refs that do not match this pattern are passed
// through verbatim for later processing.
func ParseLegacyLibraryRef(libraryRef string) string {
	if !strings.HasPrefix(libraryRef, "library://") {
		return libraryRef
	}

	parsedLibraryRef := libraryRef[10:]
	if strings.HasPrefix(parsedLibraryRef, "/") {
		return libraryRef
	}

	if !strings.Contains(parsedLibraryRef, "/") {
		return fmt.Sprintf("library:///%s", parsedLibraryRef)
	}
	return libraryRef
}

// DownloadImage is a helper function to wrap library image download operation
func DownloadImage(ctx context.Context, c *client.Client, imagePath, libraryRef string, callback progressCallback) error {

	// handle legacy library refs (ie. "library://image:tag")
	validLibraryRef := ParseLegacyLibraryRef(libraryRef)

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

	// call library client to download image
	err = c.DownloadImage(ctx, f, r.Host+r.Path, getDownloadTag(r.Tags), callback)
	if err != nil {
		// delete incomplete image file in the event of failure
		os.Remove(imagePath)

		return fmt.Errorf("error downloading image: %v", err)
	}

	return nil
}

// DownloadImageNoProgress downloads an image from the library without
// displaying a progress bar while doing so
func DownloadImageNoProgress(ctx context.Context, c *client.Client, imagePath, libraryRef string) error {
	return DownloadImage(ctx, c, imagePath, libraryRef, nil)
}

// SearchLibrary searches the library and outputs results to stdout
func SearchLibrary(ctx context.Context, c *client.Client, value string) error {
	if len(value) < 3 {
		return fmt.Errorf("bad query '%s'. You must search for at least 3 characters", value)
	}

	results, err := c.Search(ctx, value)
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
			fmt.Printf("\t\tTags: %s\n", con.TagList())
		}
		fmt.Printf("\n")

	} else {
		fmt.Printf("No containers found for '%s'\n\n", value)
	}

	return nil
}
