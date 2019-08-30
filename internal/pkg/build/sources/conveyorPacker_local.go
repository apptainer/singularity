// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"fmt"
	"path/filepath"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/build/types"
	"github.com/sylabs/singularity/pkg/image"
	"github.com/sylabs/singularity/pkg/util/loop"
)

// LocalConveyor only needs to hold the conveyor to have the needed data to pack
type LocalConveyor struct {
	src string
	b   *types.Bundle
}

// LocalPacker ...
type LocalPacker interface {
	Pack() (*types.Bundle, error)
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

	info := new(loop.Info64)

	//	fmt.Printf("OBJJSON   : %+v\n", b.JSONObjects)
	//	fmt.Printf("PATH      : %s\n", src)
	//
	//	b.JSONObjects["labels"] = []byte(`test-foo-label: = "hello world"
	//new-labels: "123"`)
	//
	//	fmt.Printf("OBJJSON   : %s\n", string(b.JSONObjects["labels"]))
	//
	//	fmt.Printf("BBBBBBBB  : %+v\n", b)
	//
	//	fmt.Printf("BBBB PATH : %s\n", b.Path)
	//
	//	//	labels := make(map[string]string)
	//
	//	//	if err = getExistingLabels(labels, b); err != nil {
	//	//		return err
	//	//	}
	//
	//	//	if err = addBuildLabels(labels, b); err != nil {
	//	//		return err
	//	//	}
	//
	//	//	if b.RunSection("labels") && len(b.Recipe.ImageData.Labels) > 0 {
	//	//		sylog.Infof("Adding labels")
	//	//
	//	//		// add new labels to new map and check for collisions
	//	//		for key, value := range b.Recipe.ImageData.Labels {
	//	//			// check if label already exists
	//	//			if _, ok := labels[key]; ok {
	//	//				// overwrite collision if it exists and force flag is set
	//	//				if b.Opts.Force {
	//	//					labels[key] = value
	//	//				} else {
	//	//					sylog.Warningf("Label: %s already exists and force option is false, not overwriting", key)
	//	//				}
	//	//			} else {
	//	//				// set if it doesnt
	//	//				labels[key] = value
	//	//			}
	//	//		}
	//	//	}
	//
	//	fmt.Printf("INFOOOOOOOOOOOOOOOOOOOO: %+v\n", b.Recipe.ImageData.Labels)
	//
	//	labels := make(map[string]map[string][]byte, 1)
	//	//labels["labels"] = string(b.JSONObjects["labels"])
	//	labels["labels"] = make(map[string][]byte, 1)
	//
	//	labels["labels"] = b.JSONObjects
	//
	//	// make new map into json
	//	text, err := json.MarshalIndent(labels, "", "\t")
	//	//text, err := json.MarshalIndent(b.JSONObjects, "", "\t")
	//	if err != nil {
	//		return nil, fmt.Errorf("HEKEDOLEOD: %s", err)
	//	}
	//
	//	fmt.Printf("TEXT   : %s\n", string(text))

	//	err = ioutil.WriteFile(filepath.Join(b.Rootfs(), "/.singularity.d/labels.json"), []byte(text), 0644)
	//	if err != nil {
	//		return nil, fmt.Errorf("BARRDIDIID: %s", err)
	//	}

	switch imageObject.Type {
	case image.SIF:
		sylog.Debugf("Packing from SIF")

		return &SIFPacker{
			srcfile: src,
			b:       b,
		}, nil
	case image.SQUASHFS:
		sylog.Debugf("Packing from Squashfs")

		info.Offset = imageObject.Partitions[0].Offset
		info.SizeLimit = imageObject.Partitions[0].Size

		return &SquashfsPacker{
			srcfile: src,
			b:       b,
			info:    info,
		}, nil
	case image.EXT3:
		sylog.Debugf("Packing from Ext3")

		info.Offset = imageObject.Partitions[0].Offset
		info.SizeLimit = imageObject.Partitions[0].Size

		return &Ext3Packer{
			srcfile: src,
			b:       b,
			info:    info,
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

// Get just stores the source
func (cp *LocalConveyorPacker) Get(b *types.Bundle) (err error) {
	// insert base metadata before unpacking fs
	if err = makeBaseEnv(b.Rootfs()); err != nil {
		return fmt.Errorf("while inserting base environment: %v", err)
	}

	cp.src = filepath.Clean(b.Recipe.Header["from"])

	cp.LocalPacker, err = GetLocalPacker(cp.src, b)
	return err
}
