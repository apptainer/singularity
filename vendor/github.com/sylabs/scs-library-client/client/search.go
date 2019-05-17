// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// Search searches library by name, returns any matching collections,
// containers, entities, or images.
func (c *Client) Search(ctx context.Context, value string) (*SearchResults, error) {
	url := fmt.Sprintf("/v1/search?value=%s", url.QueryEscape(value))

	resJSON, _, err := c.apiGet(ctx, url)
	if err != nil {
		return nil, err
	}

	var res SearchResponse
	if err := json.Unmarshal(resJSON, &res); err != nil {
		return nil, fmt.Errorf("error decoding results: %v", err)
	}

	return &res.Data, nil
}

// searchLibrary will search the library for a given query and display results
func (c *Client) searchLibrary(ctx context.Context, value string) error {
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
