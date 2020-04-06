// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/sylabs/singularity/internal/pkg/instance"
	"github.com/sylabs/singularity/internal/pkg/runtime/engine/oci"
	"github.com/sylabs/singularity/pkg/ociruntime"
	"github.com/sylabs/singularity/pkg/runtime/engine/config"
	"github.com/sylabs/singularity/pkg/sylog"
)

// OciArgs contains CLI arguments
type OciArgs struct {
	BundlePath     string
	LogPath        string
	LogFormat      string
	SyncSocketPath string
	PidFile        string
	FromFile       string
	KillSignal     string
	KillTimeout    uint32
	EmptyProcess   bool
	ForceKill      bool
}

func getCommonConfig(containerID string) (*config.Common, error) {
	commonConfig := config.Common{
		EngineConfig: &oci.EngineConfig{},
	}

	file, err := instance.Get(containerID, instance.OciSubDir)
	if err != nil {
		return nil, fmt.Errorf("no container found with name %s", containerID)
	}

	if err := json.Unmarshal(file.Config, &commonConfig); err != nil {
		return nil, fmt.Errorf("failed to read %s container configuration: %s", containerID, err)
	}

	return &commonConfig, nil
}

func getEngineConfig(containerID string) (*oci.EngineConfig, error) {
	commonConfig, err := getCommonConfig(containerID)
	if err != nil {
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

func exitContainer(ctx context.Context, containerID string, delete bool) {
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
		if err := OciDelete(ctx, containerID); err != nil {
			sylog.Errorf("%s", err)
		}
	}
}
