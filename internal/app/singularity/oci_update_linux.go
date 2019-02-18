// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity/internal/pkg/cgroups"
	"github.com/sylabs/singularity/pkg/ociruntime"
)

// OciUpdate updates container cgroups resources
func OciUpdate(containerID string, args *OciArgs) error {
	var reader io.Reader

	state, err := getState(containerID)
	if err != nil {
		return err
	}

	if state.State.Status != ociruntime.Running && state.State.Status != ociruntime.Created {
		return fmt.Errorf("container %s is neither running nor created", containerID)
	}

	if args.FromFile == "" {
		return fmt.Errorf("you must specify --from-file")
	}

	resources := &specs.LinuxResources{}
	manager := &cgroups.Manager{Pid: state.State.Pid}

	if args.FromFile == "-" {
		reader = os.Stdin
	} else {
		f, err := os.Open(args.FromFile)
		if err != nil {
			return err
		}
		reader = f
	}

	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read cgroups config file: %s", err)
	}

	if err := json.Unmarshal(data, resources); err != nil {
		return err
	}

	return manager.UpdateFromSpec(resources)
}
