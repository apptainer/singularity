// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"fmt"
	"io/ioutil"
	"os/exec"
)

// ArchConveyor only needs to hold the conveyor to have the needed data to pack
type ArchConveyor struct {
	recipe Definition
	src    string
	tmpfs  string
}

// ArchConveyorPacker only needs to hold the conveyor to have the needed data to pack
type ArchConveyorPacker struct {
	ArchConveyor
}

// Get just stores the source
func (c *ArchConveyor) Get(recipe Definition) (err error) {
	c.recipe = recipe

	//check for pacstrap on system
	pacstrapPath, err := exec.LookPath("pacstrap")
	if err != nil {
		return fmt.Errorf("pacstrap is not in PATH: %v", err)
	}

	c.tmpfs, err = ioutil.TempDir("", "temp-arch-")
	if err != nil {
		return
	}

	return nil
}

// Pack puts relevant objects in a Bundle!
func (cp *ArchConveyorPacker) Pack() (b *Bundle, err error) {

	b, err = NewBundle(cp.tmpfs)
	if err != nil {
		return
	}

	b.Recipe = cp.recipe

	return b, nil
}
