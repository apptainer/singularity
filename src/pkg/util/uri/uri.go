// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package uri

import (
	"strings"
)

// NameFromURI turns a transport:ref URI into a name containing the top-level identifier
// of the image. For example, docker://godlovedc/lolcow returns lolcow
//
// Returns "" when not in transport:ref format
func NameFromURI(uri string) string {
	uriSplit := strings.Split(uri, ":") // split URI into transport:ref:tag
	if len(uriSplit) == 1 {
		return ""
	}

	ref := uriSplit[1] // possibly split ref into components
	refSplit := strings.Split(ref, "/")

	return refSplit[len(refSplit)-1] // return last element of ref
}

// SplitURI splits a URI into it's components which can be used directly through containers/image
//
// Examples:
//   docker://ubuntu -> docker, //ubuntu
//   docker://ubuntu:18.04 -> docker, //ubuntu:18.04
//   oci-archive:path/to/archive -> oci-archive, path/to/archive
//   ubuntu -> "", ubuntu
func SplitURI(uri string) (transport string, ref string) {
	uriSplit := strings.SplitN(uri, ":", 2)
	if len(uriSplit) == 1 {
		return "", uriSplit[0]
	}

	return uriSplit[0], uriSplit[1]
}
