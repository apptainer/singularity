// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/opencontainers/runtime-tools/generate"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config/oci"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/exec"
	singularityConfig "github.com/sylabs/singularity/pkg/runtime/engines/singularity/config"
)

const listAppsCommand = "for app in ${SINGULARITY_MOUNTPOINT}/scif/apps/*; do\n    if [ -d \"$app/scif\" ]; then\n        APPNAME=`basename $app`\n        echo \"$APPNAME\"\n    fi\ndone"

// ListApps will list all the apps in the container path (cpath).
func ListApps(cpath string) error {
	// apps prints the apps installed in the container
	abspath, err := filepath.Abs(cpath)
	if err != nil {
		return fmt.Errorf("while getting absolute path: %s", err)
	}
	name := filepath.Base(abspath)

	a := []string{"/bin/sh", "-c", listAppsCommand}
	starter := buildcfg.LIBEXECDIR + "/singularity/bin/starter-suid"
	procname := "Singularity apps"
	Env := []string{sylog.GetEnvVar()}

	engineConfig := singularityConfig.NewConfig()
	ociConfig := &oci.Config{}
	generator := generate.Generator{Config: &ociConfig.Spec}
	engineConfig.OciConfig = ociConfig

	generator.SetProcessArgs(a)
	generator.SetProcessCwd("/")
	engineConfig.SetImage(abspath)

	cfg := &config.Common{
		EngineName:   singularityConfig.Name,
		ContainerID:  name,
		EngineConfig: engineConfig,
	}

	configData, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal CommonEngineConfig: %s", err)
	}

	if err := exec.Pipe(starter, []string{procname}, Env, configData); err != nil {
		return err
	}
	return nil
}
