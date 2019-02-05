// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/opencontainers/runtime-tools/generate"
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config/oci"
	singularityConfig "github.com/sylabs/singularity/internal/pkg/runtime/engines/singularity/config"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/exec"
)

func init() {
	SingularityCmd.AddCommand(AppsCmd)
}

const listAppsCommand = "for app in ${SINGULARITY_MOUNTPOINT}/scif/apps/*; do\n    if [ -d \"$app/scif\" ]; then\n        APPNAME=`basename $app`\n        echo \"$APPNAME\"\n    fi\ndone"

// AppsCmd singularity apps <container path>
var AppsCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Use:                   docs.AppsUse,
	Short:                 docs.AppsUse,
	Long:                  docs.AppsUse,
	Example:               docs.AppsUse,

	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Sanity check
		if _, err := os.Stat(args[0]); err != nil {
			sylog.Fatalf("Container not found: %s", err)
		}

		// apps prints the apps installed in the container
		abspath, err := filepath.Abs(args[0])
		if err != nil {
			sylog.Fatalf("While getting absolute path: %s", err)
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
			sylog.Fatalf("CLI Failed to marshal CommonEngineConfig: %s\n", err)
		}

		if err := exec.Pipe(starter, []string{procname}, Env, configData); err != nil {
			sylog.Fatalf("%s", err)
		}
	},
}
