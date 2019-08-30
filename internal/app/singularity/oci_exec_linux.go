// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/sylabs/singularity/pkg/ociruntime"

	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/runtime/engine/oci"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/exec"
)

// OciExec executes a command in a container
func OciExec(containerID string, cmdArgs []string) error {
	starter := buildcfg.LIBEXECDIR + "/singularity/bin/starter"

	commonConfig, err := getCommonConfig(containerID)
	if err != nil {
		return fmt.Errorf("%s doesn't exist", containerID)
	}

	engineConfig := commonConfig.EngineConfig.(*oci.EngineConfig)

	switch engineConfig.GetState().Status {
	case ociruntime.Running, ociruntime.Paused:
	default:
		args := strings.Join(cmdArgs, " ")
		return fmt.Errorf("cannot execute command %q, container '%s' is not running", args, containerID)
	}

	engineConfig.Exec = true
	engineConfig.OciConfig.SetProcessArgs(cmdArgs)

	os.Clearenv()

	configData, err := json.Marshal(commonConfig)
	if err != nil {
		sylog.Fatalf("%s", err)
	}

	Env := []string{sylog.GetEnvVar()}

	procName := fmt.Sprintf("Singularity OCI %s", containerID)
	return exec.Pipe(starter, []string{procName}, Env, configData)
}
