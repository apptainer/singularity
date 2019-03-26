// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"encoding/json"
	"os"

	"github.com/sylabs/singularity/internal/pkg/instance"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/oci"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/ociruntime"
)

// OciArgs contains CLI arguments
type OciArgs struct {
	BundlePath     string
	LogPath        string
	LogFormat      string
	SyncSocketPath string
	EmptyProcess   bool
	PidFile        string
	FromFile       string
	KillSignal     string
}

func getCommonConfig(containerID string) (*config.Common, error) {
	commonConfig := config.Common{
		EngineConfig: &oci.EngineConfig{},
	}

	file, err := instance.Get(containerID, instance.OciSubDir)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(file.Config, &commonConfig); err != nil {
		return nil, err
	}

	return &commonConfig, nil
}

func getEngineConfig(containerID string) (*oci.EngineConfig, error) {
	commonConfig := config.Common{
		EngineConfig: &oci.EngineConfig{},
	}

	file, err := instance.Get(containerID, instance.OciSubDir)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(file.Config, &commonConfig); err != nil {
		return nil, err
	}

	return commonConfig.EngineConfig.(*oci.EngineConfig), nil
}

func getState(containerID string) (*ociruntime.State, error) {
	engineConfig, err := getEngineConfig(containerID)
	if err != nil {
		return nil, err
	}
	return &engineConfig.State, nil
}

func exitContainer(containerID string, delete bool) {
	state, err := getState(containerID)
	if err != nil {
		if !delete {
			sylog.Errorf("%s", err)
			os.Exit(1)
		}
		return
	}

	if state.ExitCode != nil {
		defer os.Exit(*state.ExitCode)
	}

	if delete {
		if err := OciDelete(containerID); err != nil {
			sylog.Errorf("%s", err)
		}
	}
}
