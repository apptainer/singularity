/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package build

// Provisioner is the interface used to represent how we convert any image
// source into a chroot tree on disk. All necessary input (URL, etc...) should be
// set up when we're creating the specific data structure.
type Provisioner interface {
	Provision(string path) image.Sandbox
}

/*type PName string

type PConstructor func([]interface{}) Provisioner{}

var PConstructors = map[PName]PConstructor
*/
