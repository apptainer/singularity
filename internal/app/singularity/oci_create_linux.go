// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/opencontainers/runtime-tools/generate"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/oci"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/exec"
)

// OciCreate creates a container from an OCI bundle
func OciCreate(containerID string, args *OciArgs) error {
	starter := buildcfg.LIBEXECDIR + "/singularity/bin/starter"

	_, err := getState(containerID)
	if err == nil {
		return fmt.Errorf("%s already exists", containerID)
	}

	os.Clearenv()

	absBundle, err := filepath.Abs(args.BundlePath)
	if err != nil {
		return fmt.Errorf("failed to determine bundle absolute path: %s", err)
	}

	if err := os.Chdir(absBundle); err != nil {
		return fmt.Errorf("failed to change directory to %s: %s", absBundle, err)
	}

	engineConfig := oci.NewConfig()
	generator := generate.Generator{Config: &engineConfig.OciConfig.Spec}
	engineConfig.SetBundlePath(absBundle)
	engineConfig.SetLogPath(args.LogPath)
	engineConfig.SetLogFormat(args.LogFormat)
	engineConfig.SetPidFile(args.PidFile)

	// load config.json from bundle path
	configJSON := filepath.Join(absBundle, "config.json")
	fb, err := os.Open(configJSON)
	if err != nil {
		return fmt.Errorf("failed to open %s: %s", configJSON, err)
	}

	data, err := ioutil.ReadAll(fb)
	if err != nil {
		return fmt.Errorf("failed to read %s: %s", configJSON, err)
	}

	fb.Close()

	if err := json.Unmarshal(data, generator.Config); err != nil {
		return fmt.Errorf("failed to parse %s: %s", configJSON, err)
	}

	Env := []string{sylog.GetEnvVar()}

	engineConfig.EmptyProcess = args.EmptyProcess
	engineConfig.SyncSocket = args.SyncSocketPath

	commonConfig := &config.Common{
		ContainerID:  containerID,
		EngineName:   oci.Name,
		EngineConfig: engineConfig,
	}

	configData, err := json.Marshal(commonConfig)
	if err != nil {
		sylog.Fatalf("%s", err)
	}

	procName := fmt.Sprintf("Singularity OCI %s", containerID)
	cmd, err := exec.PipeCommand(starter, []string{procName}, Env, configData)
	if err != nil {
		return err
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}
