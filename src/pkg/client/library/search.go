// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"fmt"
)

// SearchLibrary will search the library for a given query and display results
func SearchLibrary(value string, libraryURL string, authToken string) error {
	if len(value) < 3 {
		return fmt.Errorf("Bad query '%s'. You must search for at least 3 characters", value)
	}

	results, err := search(libraryURL, authToken, value)
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
