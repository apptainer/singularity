// Copyright (c) 2018, Sylabs Inc. All rights reserved.
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
	"github.com/sylabs/singularity/src/docs"
	"github.com/sylabs/singularity/src/pkg/buildcfg"
	"github.com/sylabs/singularity/src/pkg/sylog"
	"github.com/sylabs/singularity/src/pkg/util/exec"

	"github.com/sylabs/singularity/src/runtime/engines/config"
	"github.com/sylabs/singularity/src/runtime/engines/config/oci"
	"github.com/sylabs/singularity/src/runtime/engines/singularity"
)

var (
	labels      bool
	deffile     bool
	runscript   bool
	test        bool
	environment bool
	helpfile    bool
)

func init() {
	InspectCmd.Flags().SetInterspersed(false)

	InspectCmd.Flags().BoolVarP(&labels, "labels", "a", false, "Show the labels associated with the image (default)")
	InspectCmd.Flags().SetAnnotation("labels", "envkey", []string{"LABELS"})

	InspectCmd.Flags().BoolVarP(&deffile, "deffile", "d", false, "Show the Singularity recipe file that was used to generate the image")
	InspectCmd.Flags().SetAnnotation("deffile", "envkey", []string{"DEFFILE"})

	InspectCmd.Flags().BoolVarP(&runscript, "runscript", "r", false, "Show the runscript for the image")
	InspectCmd.Flags().SetAnnotation("runscript", "envkey", []string{"RUNSCRIPT"})

	InspectCmd.Flags().BoolVarP(&test, "test", "t", false, "Show the test script for the image")
	InspectCmd.Flags().SetAnnotation("test", "envkey", []string{"TEST"})

	InspectCmd.Flags().BoolVarP(&environment, "environment", "e", false, "Show the environment settings for the image")
	InspectCmd.Flags().SetAnnotation("environment", "envkey", []string{"ENVIRONMENT"})

	InspectCmd.Flags().BoolVarP(&helpfile, "helpfile", "H", false, "Inspect the runscript helpfile, if it exists")
	InspectCmd.Flags().SetAnnotation("helpfile", "envkey", []string{"HELPFILE"})

	SingularityCmd.AddCommand(InspectCmd)
}

// InspectCmd represents the build command
var InspectCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args: cobra.ExactArgs(1),

	Use:     docs.InspectUse,
	Short:   docs.InspectShort,
	Long:    docs.InspectLong,
	Example: docs.InspectExample,

	Run: func(cmd *cobra.Command, args []string) {

		// Sanity check
		if _, err := os.Stat(args[0]); err != nil {
			sylog.Fatalf("container not found: %s", err)
		}

		abspath, err := filepath.Abs(args[0])
		name := filepath.Base(abspath)

		a := []string{"/bin/cat"}

		if helpfile {
			sylog.Debugf("Inspection of helpfile selected.")
			a = append(a, ".singularity.d/runscript.help")
		}

		if deffile {
			sylog.Debugf("Inspection of deffile selected.")
			a = append(a, ".singularity.d/Singularity")
		}

		if runscript {
			sylog.Debugf("Inspection of runscript selected.")
			a = append(a, ".singularity.d/runscript")
		}

		if test {
			sylog.Debugf("Inspection of test selected.")
			a = append(a, ".singularity.d/test")
		}

		if environment {
			sylog.Debugf("Inspection of envrionment selected.")
			a = append(a, ".singularity.d/env/90-environment.sh")
		}

		// default to labels if nothing was appended
		if labels || len(a) == 1 {
			sylog.Debugf("Inspection of labels as default.")
			a = append(a, ".singularity.d/labels.json")
		}

		starter := buildcfg.SBINDIR + "/starter-suid"
		procname := "Singularity inspect"
		Env := []string{sylog.GetEnvVar(), "SRUNTIME=singularity"}

		engineConfig := singularity.NewConfig()
		ociConfig := &oci.Config{}
		generator := generate.Generator{Config: &ociConfig.Spec}
		engineConfig.OciConfig = ociConfig

		generator.SetProcessArgs(a)
		engineConfig.SetImage(abspath)

		cfg := &config.Common{
			EngineName:   singularity.Name,
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
	TraverseChildren: true,
}
