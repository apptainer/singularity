// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/sylabs/singularity/internal/pkg/build/types"
)

// ScratchConveyor only needs to hold the conveyor to have the needed data to pack
type ScratchConveyor struct {
	b *types.Bundle
}

// ScratchConveyorPacker only needs to hold the conveyor to have the needed data to pack
type ScratchConveyorPacker struct {
	ScratchConveyor
}

// Get just stores the source
func (c *ScratchConveyor) Get(b *types.Bundle) (err error) {
	c.b = b

	return nil
}

// Pack puts relevant objects in a Bundle!
func (cp *ScratchConveyorPacker) Pack() (b *types.Bundle, err error) {
	err = cp.insertBaseEnv()
	if err != nil {
		return nil, fmt.Errorf("While inserting base environment: %v", err)
	}

	err = cp.insertRunScript()
	if err != nil {
		return nil, fmt.Errorf("While inserting runscript: %v", err)
	}

	return cp.b, nil
}

func (c *ScratchConveyor) insertBaseEnv() (err error) {
	if err = makeBaseEnv(c.b.Rootfs()); err != nil {
		return
	}
	return nil
}

func (cp *ScratchConveyorPacker) insertRunScript() (err error) {
	ioutil.WriteFile(filepath.Join(cp.b.Rootfs(), "/.singularity.d/runscript"), []byte("#!/bin/sh\n"), 0755)
	if err != nil {
		return
	}

	return nil
}

// CleanUp removes any tmpfs owned by the conveyorPacker on the filesystem
func (c *ScratchConveyor) CleanUp() {
	os.RemoveAll(c.b.Path)
}
