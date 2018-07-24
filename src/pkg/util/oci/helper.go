// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package oci

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/sylabs/sif/pkg/sif"
)

const (
	// ConfigSpec holds the name for the OCI runtime spec file
	ConfigSpec = "config.json"
)

// LoadConfigSpec loads the config.json oci runtime spec
// from the provided path to a SIF.
func LoadConfigSpec(Path string) (spec *specs.Spec, err error) {
	// load the SIF (singularity image file)
	fimg, err := sif.LoadContainer(Path, false)
	if err != nil {
		sylog.Fatalf("Error loading SIF %s:\t%s", Path, err)
	}

	// lookup of a descriptor of type DataGenericJSON
	descr := sif.Descriptor{
		Datatype: sif.DataGenericJSON,
	}
	d, match, _ := fimg.GetFromDescr(descr)
	if err != nil {
		sylog.Fatalf("%s", err)
	}
	if match != 1 && d.GetName() != ConfigSpec {
		sylog.Infof("SIF bundle doesn't contains a OCI runtime spec")
		return
	}

	// if found, retrieve the OCI spec from file
	data := fimg.Filedata[d.Fileoff : d.Fileoff+d.Filelen]

	if err = json.Unmarshal(data, &spec); err != nil {
		return nil, err
	}

	// unload the SIF container
	if err = fimg.UnloadContainer(); err != nil {
		sylog.Fatalf("UnloadContainer(fimg):\t%s", err)
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
