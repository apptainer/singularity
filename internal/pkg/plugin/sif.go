// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package plugin

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/sylabs/sif/pkg/sif"
	"github.com/sylabs/singularity/pkg/image"
	pluginapi "github.com/sylabs/singularity/pkg/plugin"
)

const (
	// pluginBinaryName is the name of the plugin binary within the
	// SIF file
	pluginBinaryName = "plugin.so"
	// pluginManifestName is the name of the plugin manifest within
	// the SIF file
	pluginManifestName = "plugin.manifest"
)

// isPluginFile checks if the image.Image contains the sections which
// make up a valid plugin. A plugin sif file should have the following
// format:
//
// DESCR[0]: Sifplugin
//   - Datatype: sif.DataPartition
//   - Fstype:   sif.FsRaw
//   - Parttype: sif.PartData
// DESCR[1]: Sifmanifest
//   - Datatype: sif.DataGenericJSON
func isPluginFile(img *image.Image) bool {
	if img.Type != image.SIF {
		return false
	}

	part, _ := img.GetAllPartitions()
	if len(part) != 1 || len(img.Sections) != 1 {
		return false
	}

	// check binary object
	if part[0].Name != pluginBinaryName {
		return false
	} else if part[0].AllowedUsage&image.DataUsage == 0 {
		return false
	} else if part[0].Type != image.RAW {
		return false
	}

	// check manifest
	if img.Sections[0].Name != pluginManifestName {
		return false
	} else if img.Sections[0].AllowedUsage&image.DataUsage == 0 {
		return false
	} else if img.Sections[0].Type != uint32(sif.DataGenericJSON) {
		return false
	}

	return true
}

// getManifest will extract the Manifest data from the input FileImage.
func getManifest(img *image.Image) (pluginapi.Manifest, error) {
	var manifest pluginapi.Manifest

	r, err := getManifestReader(img)
	if err != nil {
		return manifest, err
	}

	if err := json.NewDecoder(r).Decode(&manifest); err != nil {
		return manifest, fmt.Errorf("while decoding JSON manifest: %s", err)
	}

	return manifest, nil
}

func getBinaryReader(img *image.Image) (io.Reader, error) {
	return image.NewPartitionReader(img, pluginBinaryName, -1)
}

func getManifestReader(img *image.Image) (io.Reader, error) {
	return image.NewSectionReader(img, pluginManifestName, -1)
}
