// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"context"
	"fmt"

	"github.com/sylabs/singularity/pkg/build/types"
	"github.com/sylabs/singularity/pkg/image"
	"github.com/sylabs/singularity/pkg/image/unpacker"
)

// SquashfsPacker holds the locations of where to pack from and to, aswell as image offset info
type SquashfsPacker struct {
	srcfile string
	b       *types.Bundle
	img     *image.Image
}

// Pack puts relevant objects in a Bundle!
func (p *SquashfsPacker) Pack(context.Context) (*types.Bundle, error) {
	// create a reader for rootfs partition
	reader, err := image.NewPartitionReader(p.img, "", 0)
	if err != nil {
		return nil, fmt.Errorf("could not extract root filesystem: %s", err)
	}

	s := unpacker.NewSquashfs()

	// extract root filesystem
	if err := s.ExtractAll(reader, p.b.RootfsPath); err != nil {
		return nil, fmt.Errorf("root filesystem extraction failed: %s", err)
	}

	return p.b, nil
}
