/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package libexec

import (
	"fmt"
)

// PullImage is the function that is responsible for pulling an image from a Sylabs library. This will
// eventually be integrated with the build system as a builder, but for now this is the palce to put it
func PullImage(image string, library string) {
	fmt.Printf("Pulling image: \"%s\" from library: \"%s\"\n", image, library)
}
