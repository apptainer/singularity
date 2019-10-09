// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/build/types"
	"github.com/sylabs/singularity/pkg/image"
)

// LocalConveyor only needs to hold the conveyor to have the needed data to pack
type LocalConveyor struct {
	src string
	b   *types.Bundle
}

// LocalPacker ...
type LocalPacker interface {
	Pack(context.Context) (*types.Bundle, error)
}

// LocalConveyorPacker only needs to hold the conveyor to have the needed data to pack
type LocalConveyorPacker struct {
	LocalConveyor
	LocalPacker
}

// GetLocalPacker ...
func GetLocalPacker(src string, b *types.Bundle) (LocalPacker, error) {

	imageObject, err := image.Init(src, false)
	if err != nil {
		return nil, err
	}

	switch imageObject.Type {
	case image.SIF:
		sylog.Debugf("Packing from SIF")

		return &SIFPacker{
			srcFile: src,
			b:       b,
			img:     imageObject,
		}, nil
	case image.SQUASHFS:
		sylog.Debugf("Packing from Squashfs")

		return &SquashfsPacker{
			srcfile: src,
			b:       b,
			img:     imageObject,
		}, nil
	case image.EXT3:
		sylog.Debugf("Packing from Ext3")

		return &Ext3Packer{
			srcfile: src,
			b:       b,
			img:     imageObject,
		}, nil
	case image.SANDBOX:
		sylog.Debugf("Packing from Sandbox")

		return &SandboxPacker{
			srcdir: src,
			b:      b,
		}, nil
	default:
		return nil, fmt.Errorf("invalid image format")
	}
}

// Get just stores the source.
func (cp *LocalConveyorPacker) Get(ctx context.Context, b *types.Bundle) (err error) {
	// insert base metadata before unpacking fs
	if err = makeBaseEnv(b.RootfsPath); err != nil {
		return fmt.Errorf("while inserting base environment: %v", err)
	}

	cp.src = filepath.Clean(b.Recipe.Header["from"])

	cp.LocalPacker, err = GetLocalPacker(cp.src, b)
	return err
}
