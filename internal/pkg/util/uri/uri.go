// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
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
	// HTTP is the keyword for http ref
	HTTP = "http"
	// HTTPS is the keyword for https ref
	HTTPS = "https"
	// Oras is the keyword for an oras ref
	Oras = "oras"
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
	"http":           true,
	"https":          true,
	"oras":           true,
}

// IsValid returns whether or not the given source is valid
func IsValid(source string) (valid bool, err error) {
	u := strings.SplitN(source, ":", 2)

	if len(u) != 2 {
		return false, fmt.Errorf("invalid uri %s", source)
	}

	if _, ok := validURIs[u[0]]; ok {
		return true, nil
	}

	return false, fmt.Errorf("invalid uri %s", source)
}

// GetName turns a transport:ref URI into a name containing the top-level identifier
// of the image. For example, docker://sylabsio/lolcow returns lolcow
//
// Returns "" when not in transport:ref format
func GetName(uri string) string {
	transport, ref := Split(uri)
	if transport == "" {
		return ""
	}

	ref = strings.TrimLeft(ref, "/")    // Trim leading "/" characters
	refSplit := strings.Split(ref, "/") // Split ref into parts

	if transport == HTTP || transport == HTTPS {
		imageName := refSplit[len(refSplit)-1]
		return imageName
	}

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

// Split splits a URI into it's components which can be used directly through containers/image
//
// This can be tricky if there is no type but a file name contains a colon.
//
// Examples:
//   docker://ubuntu -> docker, //ubuntu
//   docker://ubuntu:18.04 -> docker, //ubuntu:18.04
//   oci-archive:path/to/archive -> oci-archive, path/to/archive
//   ubuntu -> "", ubuntu
//   ubuntu:18.04.img -> "", ubuntu:18.04.img
func Split(uri string) (transport string, ref string) {
	uriSplit := strings.SplitN(uri, ":", 2)
	if len(uriSplit) == 1 {
		// no colon
		return "", uri
	}

	if strings.HasPrefix(uriSplit[1], "//") {
		// the format was ://, so try it whether or not valid URI
		return uriSplit[0], uriSplit[1]
	}

	if ok, err := IsValid(uri); ok && err == nil {
		// also accept recognized URIs
		return uriSplit[0], uriSplit[1]
	}

	return "", uri
}
