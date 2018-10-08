// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package uri

import (
	"fmt"
	"strings"
)

const (
	// Library is the keyword for a library ref
	Library = "library"
	// Shub is the keyword for a shub ref
	Shub = "shub"
)

// validURIs contains a list of known uris
var validURIs = map[string]bool{
	"library":        true,
	"shub":           true,
	"docker":         true,
	"docker-archive": true,
	"docker-daemon":  true,
	"oci":            true,
	"oci-archive":    true,
}

// IsValidURI returns whether or not the given source is valid
func IsValidURI(source string) (valid bool, err error) {

	u := strings.SplitN(source, ":", 2)

	if len(u) != 2 {
		return false, fmt.Errorf("Invalid URI %s", source)
	}

	if _, ok := validURIs[u[0]]; ok {
		return true, nil
	}

	return false, fmt.Errorf("Invalid URI %s", source)
}

// NameFromURI turns a transport:ref URI into a name containing the top-level identifier
// of the image. For example, docker://godlovedc/lolcow returns lolcow
//
// Returns "" when not in transport:ref format
func NameFromURI(uri string) string {
	uriSplit := strings.SplitN(uri, ":", 2) // split URI into transport:ref:tag
	if len(uriSplit) == 1 {
		return ""
	}

	ref := strings.TrimLeft(uriSplit[1], "/") // Trim leading "/" characters
	refSplit := strings.Split(ref, "/")       // Split ref into parts

	// Default tag is latest
	tags := []string{"latest"}
	container := refSplit[len(refSplit)-1]

	if strings.Contains(container, ":") {
		imageParts := strings.Split(container, ":")
		container = imageParts[0]
		tags = []string{imageParts[1]}
		if strings.Contains(tags[0], ",") {
			tags = strings.Split(tags[0], ",")
		}
	}

	return fmt.Sprintf("%s_%s.sif", container, tags[0])
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
