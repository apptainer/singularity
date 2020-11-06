// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sylabs/singularity/pkg/build/types"
	"github.com/sylabs/singularity/pkg/image"
	"github.com/sylabs/singularity/pkg/sylog"
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
func GetLocalPacker(ctx context.Context, src string, b *types.Bundle) (LocalPacker, error) {

	imageObject, err := image.Init(src, false)
	if err != nil {
		return nil, err
	}

	switch imageObject.Type {
	case image.SIF:
		sylog.Debugf("Packing from SIF")
		// Retrieve list of required fingerprints from definition, if any
		fps := []string{}
		if fpHdr, ok := b.Recipe.Header["fingerprints"]; ok {
			// Remove trailing comment
			fpHdr = strings.Split(fpHdr, "#")[0]
			fpHdr = strings.TrimSpace(fpHdr)
			if fpHdr != "" {
				fps = strings.Split(fpHdr, ",")
				for i, v := range fps {
					fps[i] = strings.TrimSpace(v)
				}
			}
		}
		// Check if the SIF matches the `fingerprints:` specified in the build, if there are any
		if len(fps) > 0 {
			err := checkSIFFingerprint(ctx, src, fps, b.Opts.KeyServerOpts...)
			if err != nil {
				return nil, fmt.Errorf("while checking fingerprint: %s", err)
			}
		} else {
			// Otherwise do a verification and make failures warn, like for push
			err := verifySIF(ctx, src, b.Opts.KeyServerOpts...)
			if err != nil {
				sylog.Warningf("%s", err)
				sylog.Warningf("Bootstrap image could not be verified, but build will continue.")
			}
		}
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

	cp.LocalPacker, err = GetLocalPacker(ctx, cp.src, b)
	return err
}
