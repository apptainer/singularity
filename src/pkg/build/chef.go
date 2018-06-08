/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package build

import (
	"fmt"
)

// validChefs contains of list of know Chefs
var validChefs = map[string]bool{
    "SIF":      true,
    "sandbox":  true,
}

// Chef is responsible for cooking up an image from a kitchen
type Chef interface {
	Cook(*Kitchen, string) (error)
}

// IsValidChef returns whether or not the given chef is valid
func IsValidChef(c string) (valid bool, err error) {
	if _, ok := validChefs[c]; ok {
		return true, nil
	}

	return false, fmt.Errorf("Invalid chef %s", c)
}
