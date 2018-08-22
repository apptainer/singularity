// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"fmt"
	"regexp"
	"strings"
)

// ShubParseReference accepts a Shub reference string and parses its content
// It will return an error if the given URI is not valid,
// otherwise it will parse the contents into a ShubURI struct
func ShubParseReference(src string) (uri ShubURI, err error) {

	//define regex for each URI component
	registryRegexp := `([-a-zA-Z0-9/]{1,64}\/)?` //target is very open, outside registry
	nameRegexp := `([-a-zA-Z0-9]{1,39}\/)`       //target valid github usernames
	containerRegexp := `([-_.a-zA-Z0-9]{1,64})`  //target valid github repo names
	tagRegexp := `(:[-_.a-zA-Z0-9]{1,64})?`      //target is very open, file extensions or branch names
	digestRegexp := `(\@[a-f0-9]{32})?`          //target md5 sum hash

	//expression is anchored
	shubRegex, err := regexp.Compile(`^\/\/` + registryRegexp + nameRegexp + containerRegexp + tagRegexp + digestRegexp + `$`)
	if err != nil {
		return uri, err
	}

	found := shubRegex.FindString(src)

	//sanity check
	//if found string is not equal to the input, input isn't a valid URI
	if strings.Compare(src, found) != 0 {
		return uri, fmt.Errorf("Source string is not a valid URI: %s", src)
	}

	//strip `//` from start of src
	src = src[2:]

	pieces := strings.SplitAfterN(src, `/`, -1)
	if l := len(pieces); l > 2 {
		//more than two pieces indicates a custom registry
		uri.defaultReg = false
		uri.registry = strings.Join(pieces[:l-2], "")
		uri.user = pieces[l-2]
		src = pieces[l-1]
	} else if l == 2 {
		//two pieces means default registry
		uri.defaultReg = true
		uri.registry = defaultRegistry
		uri.user = pieces[l-2]
		src = pieces[l-1]
	}

	//look for an @ and split if it exists
	if strings.Contains(src, `@`) {
		pieces = strings.Split(src, `@`)
		uri.digest = `@` + pieces[1]
		src = pieces[0]
	}

	//look for a : and split if it exists
	if strings.Contains(src, `:`) {
		pieces = strings.Split(src, `:`)
		uri.tag = `:` + pieces[1]
		src = pieces[0]
	}

	//container name is left over after other parts are split from it
	uri.container = src

	return uri, nil
}

func (s *ShubURI) String() string {
	return s.registry + s.user + s.container + s.tag + s.digest
}
