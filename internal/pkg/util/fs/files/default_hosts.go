// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package files

var defaultContent = `127.0.0.1   localhost
::1         localhost ip6-localhost ip6-loopback
ff02::1     ip6-allnodes
ff02::2     ip6-allrouters
`

// DefaultHosts creates the default hosts file.
func DefaultHosts() []byte {
	return []byte(defaultContent)
}
