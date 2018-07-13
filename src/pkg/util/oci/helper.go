// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package oci

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/sif/pkg/sif"
)

// LoadConfigSpec loads the config.json oci runtime spec
// from the provided path to a SIF.
func LoadConfigSpec(Path string) (spec *specs.Spec, err error) {
	configJSON, err := os.Open(Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("JSON specification file %s not found", Path)
		}
		return nil, err
	}
	defer configJSON.Close()

	// Load the SIF
	fimg, err := sif.LoadContainerFp(configJSON, true)
	if err != nil {
		return nil, fmt.Errorf("while loading SIF file: %s", err)
	}
	defer fimg.UnloadContainer()

	name := "oci-runtime-spec"
	var bn [128]byte
	copy(bn[:], name)
	var data []byte

	// Search for the SIF data object with the config.json
	// MUST be named oci-runtime-spec
	for _, v := range fimg.DescrArr {
		if v.Used == false {
			continue
		} else if v.Name == bn {
			object, _, err := fimg.GetFromDescrID(v.ID)
			if err != nil {
				return nil, fmt.Errorf("no json file found: %s", err)
			}
			data = fimg.Filedata[object.Fileoff : object.Fileoff+object.Filelen]
			break
		}
	}

	if err = json.Unmarshal(data, &spec); err != nil {
		return nil, err
	}
	return spec, validateConfigSpec(spec)
}

func validateConfigSpec(spec *specs.Spec) error {
	if spec.Process.Cwd == "" {
		return fmt.Errorf("Cwd property MUST not be empty")
	}
	if !filepath.IsAbs(spec.Process.Cwd) {
		return fmt.Errorf("Cwd MUST be an absolute path")
	}
	if len(spec.Process.Args) == 0 {
		return fmt.Errorf("args MUST not be empty")
	}
	return nil
}
