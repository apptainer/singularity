// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package shub

import (
	"errors"
	"regexp"
	"strings"
)

// isShubPullRef returns true if the provided string is a valid Shub
// reference for a pull operation.
func isShubPullRef(shubRef string) bool {
	// define regex for each URI component
	registryRegexp := `([-.a-zA-Z0-9/]{1,64}\/)?`           // target is very open, outside registry
	nameRegexp := `([-a-zA-Z0-9]{1,39}\/)`                  // target valid github usernames
	containerRegexp := `([-_.a-zA-Z0-9]{1,64})`             // target valid github repo names
	tagRegexp := `(:[-_.a-zA-Z0-9]{1,64})?`                 // target is very open, file extensions or branch names
	digestRegexp := `((\@[a-f0-9]{32})|(\@[a-f0-9]{40}))?$` // target file md5 has, git commit hash, git branch

	// expression is anchored
	shubRegex, err := regexp.Compile(`^(shub://)` + registryRegexp + nameRegexp + containerRegexp + tagRegexp + digestRegexp + `$`)
	if err != nil {
		return false
	}

	found := shubRegex.FindString(shubRef)

	// sanity check
	// if found string is not equal to the input, input isn't a valid URI
	return shubRef == found
}

// ParseReference accepts a valid Shub reference string and parses its content
// It will return an error if the given URI is not valid,
// otherwise it will parse the contents into a URI struct
func ParseReference(src string) (URI, error) {
	uri := URI{}

	ShubRef := strings.TrimPrefix(src, "shub://")
	refParts := strings.Split(ShubRef, "/")

	if l := len(refParts); l > 2 {
		// more than two pieces indicates a custom registry
		uri.registry = strings.Join(refParts[:l-2], "/") + shubAPIRoute
		uri.user = refParts[l-2]
		src = refParts[l-1]
	} else if l == 2 {
		// two pieces means default registry
		uri.registry = defaultRegistry + shubAPIRoute
		uri.user = refParts[l-2]
		src = refParts[l-1]
	} else if l < 2 {
		return URI{}, errors.New("not a valid Shub reference")
	}

	// look for an @ and split if it exists
	if strings.Contains(src, `@`) {
		refParts = strings.Split(src, `@`)
		uri.digest = `@` + refParts[1]
		src = refParts[0]
	}

	// look for a : and split if it exists
	if strings.Contains(src, `:`) {
		refParts = strings.Split(src, `:`)
		uri.tag = `:` + refParts[1]
		src = refParts[0]
	}

	// container name is left over after other parts are split from it
	uri.container = src

	if uri.tag == "" && uri.digest == "" {
		uri.tag = ":latest"
	}

	return uri, nil
}
