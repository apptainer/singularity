/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package build

import (
    "strings"
    "fmt"
)

// validPullFurnishers contains a list of known PullFurnishers
var validPullFurnishers = map[string]bool {
    "library":     	true,
	"docker":      	true,
	"shub":        	true,
	"debootstrap": 	true,
	"yum":			true,
	"squashfs":    	true,
	"sif":			true,
}

// Puller is responsible for downloading from remote sources (library, shub, docker...)
type Puller interface {
	Pull(string) (error)
}

// Furnisher is the type which is responsible for installing the chroot directory,
// metadata directory, and potentially other files/directories within the Kitchen
type Furnisher interface {
	Furnish() (*Kitchen, error)
}

// PullFurnisher describes an interface that a PullFurnisher type must implement
type PullFurnisher interface {
    Puller
    Furnisher
}

// IsValidPullFurnisher returns whether or not the given source is valid
func IsValidPullFurnisher(source string) (valid bool, err error) {
	u := strings.SplitN(source, ":", 2)

	if len(u) != 2 {
		return false, nil
	}

	if _, ok := validPullFurnishers[u[0]]; ok {
		return true, nil
	}

	return false, fmt.Errorf("Invalid pullFurnisher %s", source)
}
