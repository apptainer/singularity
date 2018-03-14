/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package build

import (
	"fmt"

	"github.com/singularityware/singularity/pkg/image"
)

// CachedBuilder is the object that satisfies the Builder interface which is in charge
// of quickly builder an image from a URI (i.e. docker://, shub://, etc...)
type CachedBuilder struct {
	P         Provisioner
	imagePath string
	Image     image.Image
}

func NewCachedBuilder(image string, uri string) (c *CachedBuilder, err error) {
	fmt.Printf("Building a cached image (%s) from source (%s)\n", image, uri)
	c = &CachedBuilder{
		imagePath: image,
	}

	c.P, err = NewProvisionerFromURI(uri)
	if err != nil {
		return nil, err
	}

	return c, err
}

func (c *CachedBuilder) Build() {
	c.P.Provision(c.imagePath)
}
