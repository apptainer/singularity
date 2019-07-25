// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/build/types"
	shub "github.com/sylabs/singularity/pkg/client/shub"
)

// ShubConveyorPacker only needs to hold the conveyor to have the needed data to pack
type ShubConveyorPacker struct {
	recipe types.Definition
	b      *types.Bundle
	LocalPacker
}

// Get downloads container from Singularityhub
func (cp *ShubConveyorPacker) Get(b *types.Bundle) (err error) {
	sylog.Debugf("Getting container from Shub")

	cp.b = b

	src := `shub://` + b.Recipe.Header["from"]

	//create file for image download
	f, err := ioutil.TempFile(cp.b.Path, "shub-img")
	if err != nil {
		return
	}
	defer f.Close()

	cp.b.FSObjects["shubImg"] = f.Name()

	// get image from singularity hub
	if err := shub.DownloadImage(cp.b.FSObjects["shubImg"], src, true, cp.b.Opts.NoHTTPS); err != nil {
		return fmt.Errorf("unable to get image from %s: %v", src, err)
	}

	// insert base metadata before unpacking fs
	if err = makeBaseEnv(cp.b.Rootfs()); err != nil {
		return fmt.Errorf("while inserting base environment: %v", err)
	}

	cp.LocalPacker, err = GetLocalPacker(cp.b.FSObjects["shubImg"], cp.b)

	return err
}

// CleanUp removes any tmpfs owned by the conveyorPacker on the filesystem
func (cp *ShubConveyorPacker) CleanUp() {
	os.RemoveAll(cp.b.Path)
}
