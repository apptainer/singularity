// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package files

import (
	"fmt"
	"net"

	"github.com/sylabs/singularity/pkg/sylog"
)

// ResolvConf creates a resolv.conf content with provided dns list and returns it
func ResolvConf(dns []string) (content []byte, err error) {
	sylog.Verbosef("Creating resolv.conf content\n")
	if len(dns) == 0 {
		return content, fmt.Errorf("no dns ip provided")
	}
	for _, ip := range dns {
		if net.ParseIP(ip) == nil {
			return content, fmt.Errorf("dns ip %s is not a valid IP address", ip)
		}
		line := fmt.Sprintf("nameserver %s\n", ip)
		content = append(content, line...)
	}
	return content, nil
}
