// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"context"
	"fmt"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/oras"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/build/types"
)

// OrasConveyorPacker only needs to hold a packer to pack the image it pulls
// as well as extra information about the library it's pulling from.
type OrasConveyorPacker struct {
	LocalPacker
}

// Get downloads container from Singularityhub
func (cp *OrasConveyorPacker) Get(ctx context.Context, b *types.Bundle) (err error) {
	sylog.Debugf("Getting container from registry using ORAS")

	// uri with leading // for oras handlers to consume
	ref := "//" + b.Recipe.Header["from"]
	// full uri for name determination and output
	fullRef := "oras:" + ref

	hash, err := oras.ImageSHA(ctx, ref, b.Opts.DockerAuthConfig)
	if err != nil {
		return fmt.Errorf("failed to get SHA of %v: %v", fullRef, err)
	}

	cacheEntry, err := b.Opts.ImgCache.GetEntry(cache.OrasCacheType, hash)
	if err != nil{
		return fmt.Errorf("unable to check if %v exists in cache: %v", hash, err)
	}

	if !cacheEntry.Exists {
		sylog.Infof("Downloading image with ORAS")

		if err := oras.DownloadImage(cacheEntry.TmpPath, ref, b.Opts.DockerAuthConfig); err != nil {
			return fmt.Errorf("unable to Download Image: %v", err)
		}

		if cacheFileHash, err := oras.ImageHash(cacheEntry.TmpPath); err != nil {
			return fmt.Errorf("error getting ImageHash: %v", err)
		} else if cacheFileHash != hash {
			return fmt.Errorf("cached file hash(%s) and expected hash(%s) does not match", cacheFileHash, hash)
		}

		err = cacheEntry.Finalize()
		if err != nil {
			return err
		}

	}

	// insert base metadata before unpacking fs
	if err = makeBaseEnv(b.RootfsPath); err != nil {
		return fmt.Errorf("while inserting base environment: %v", err)
	}

	cp.LocalPacker, err = GetLocalPacker(cacheEntry.Path, b)
	return err
}
