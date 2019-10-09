// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/build/types"
	shub "github.com/sylabs/singularity/pkg/client/shub"
)

// ShubConveyorPacker only needs to hold the conveyor to have the needed data to pack.
type ShubConveyorPacker struct {
	recipe types.Definition
	b      *types.Bundle
	LocalPacker
}

// Get downloads container from Singularityhub.
func (cp *ShubConveyorPacker) Get(ctx context.Context, b *types.Bundle) (err error) {
	sylog.Debugf("Getting container from Shub")

	cp.b = b

	src := `shub://` + b.Recipe.Header["from"]

	// create file for image download
	f, err := ioutil.TempFile(cp.b.TmpDir, "shub-img")
	if err != nil {
		return
	}
	defer f.Close()

	shubURIRef, err := shub.ShubParseReference(src)
	if err != nil {
		return fmt.Errorf("failed to parse shub uri: %s", err)
	}

	// Get the image manifest
	manifest, err := shub.GetManifest(shubURIRef, cp.b.Opts.NoHTTPS)
	if err != nil {
		return fmt.Errorf("failed to get manifest for: %s: %s", src, err)
	}

	// get image from singularity hub
	if err := shub.DownloadImage(manifest, f.Name(), src, true, cp.b.Opts.NoHTTPS); err != nil {
		return fmt.Errorf("unable to get image from: %s: %v", src, err)
	}

	// insert base metadata before unpacking fs
	if err = makeBaseEnv(cp.b.RootfsPath); err != nil {
		return fmt.Errorf("while inserting base environment: %v", err)
	}

	cp.LocalPacker, err = GetLocalPacker(f.Name(), cp.b)

	return err
}

// CleanUp removes any tmpfs owned by the conveyorPacker on the filesystem
func (cp *ShubConveyorPacker) CleanUp() {
	cp.b.Remove()
}
