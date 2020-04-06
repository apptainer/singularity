// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package files

import (
	"fmt"
	"regexp"

	"github.com/sylabs/singularity/pkg/sylog"
)

var hostRegex = `^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])$`

// Hostname creates a hostname content with provided hostname and returns it
func Hostname(hostname string) (content []byte, err error) {
	sylog.Verbosef("Creating hostname content\n")
	if hostname == "" {
		return content, fmt.Errorf("no hostname provided")
	}
	r := regexp.MustCompile(hostRegex)
	if !r.MatchString(hostname) {
		return content, fmt.Errorf("%s is not a valid hostname", hostname)
	}
	line := fmt.Sprintf("%s\n", hostname)
	content = append(content, line...)
	return content, nil
}
