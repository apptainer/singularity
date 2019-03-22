// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/opencontainers/runtime-tools/generate"
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/exec"

	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config/oci"
	singularityConfig "github.com/sylabs/singularity/internal/pkg/runtime/engines/singularity/config"
)

var (
	labels      bool
	deffile     bool
	runscript   bool
	testfile    bool
	environment bool
	helpfile    bool
	jsonfmt     bool
)

func init() {
	InspectCmd.Flags().SetInterspersed(false)

	InspectCmd.Flags().StringVar(&AppName, "app", "", "inspect a specific app")
	InspectCmd.Flags().SetAnnotation("app", "envkey", []string{"APP"})

	InspectCmd.Flags().BoolVarP(&labels, "labels", "l", false, "show the labels associated with the image (default)")
	InspectCmd.Flags().SetAnnotation("labels", "envkey", []string{"LABELS"})

	InspectCmd.Flags().BoolVarP(&deffile, "deffile", "d", false, "show the Singularity recipe file that was used to generate the image")
	InspectCmd.Flags().SetAnnotation("deffile", "envkey", []string{"DEFFILE"})

	InspectCmd.Flags().BoolVarP(&runscript, "runscript", "r", false, "show the runscript for the image")
	InspectCmd.Flags().SetAnnotation("runscript", "envkey", []string{"RUNSCRIPT"})

	InspectCmd.Flags().BoolVarP(&testfile, "test", "t", false, "show the test script for the image")
	InspectCmd.Flags().SetAnnotation("test", "envkey", []string{"TEST"})

	InspectCmd.Flags().BoolVarP(&environment, "environment", "e", false, "show the environment settings for the image")
	InspectCmd.Flags().SetAnnotation("environment", "envkey", []string{"ENVIRONMENT"})

	InspectCmd.Flags().BoolVarP(&helpfile, "helpfile", "H", false, "inspect the runscript helpfile, if it exists")
	InspectCmd.Flags().SetAnnotation("helpfile", "envkey", []string{"HELPFILE"})

	InspectCmd.Flags().BoolVarP(&jsonfmt, "json", "j", false, "print structured json instead of sections")
	InspectCmd.Flags().SetAnnotation("json", "envkey", []string{"JSON"})

	SingularityCmd.AddCommand(InspectCmd)
}

func getLabelsFile(appName string) string {
	if appName == "" {
		return " cat /.singularity.d/labels.json;"
	}

	return fmt.Sprintf(" cat /scif/apps/%s/scif/labels.json;", appName)
}

func getRunscriptFile(appName string) string {
	if appName == "" {
		return " cat /.singularity.d/runscript;"
	}

	return fmt.Sprintf("/scif/apps/%s/scif/runscript", appName)
}

func getTestFile(appName string) string {
	if appName == "" {
		return " cat /.singularity.d/test;"
	}

	return fmt.Sprintf("/scif/apps/%s/scif/test", appName)
}

func getEnvFile(appName string) string {
	if appName == "" {
		return " find /.singularity.d/env -name 9*-environment.sh -exec echo -n == \\; -exec basename -z {} \\; -exec echo == \\; -exec cat {} \\; -exec echo \\;;"
	}

	return fmt.Sprintf(" find /scif/apps/%s/scif/env -name 9*-environment.sh -exec echo -n == \\; -exec basename -z {} \\; -exec echo == \\; -exec cat {} \\; -exec echo \\;;", appName)
}

func getHelpFile(appName string) string {
	if appName == "" {
		return " cat /.singularity.d/runscript.help;"
	}

	return fmt.Sprintf("/scif/apps/%s/scif/runscript.help", appName)
}

func getAppCheck(appName string) string {
	return fmt.Sprintf("if ! [ -d \"/scif/apps/%s\" ]; then echo \"App %s does not exist.\"; exit 2; fi;", appName, appName)
}

// InspectCmd represents the build command
var InspectCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(1),

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
		if err != nil {
			sylog.Fatalf("While determining absolute file path: %v", err)
		}
		name := filepath.Base(abspath)

		attributes := make(map[string]string)

		a := []string{"/bin/sh", "-c", ""}
		prefix := "@@@start"
		delimiter := "@@@end"

		// If AppName is given fail quickly (exit) if it doesn't exist
		if AppName != "" {
			sylog.Debugf("Inspection of App %s Selected.", AppName)
			a[2] += getAppCheck(AppName)
		}

		if helpfile {
			sylog.Debugf("Inspection of helpfile selected.")

			// append to a[2] to run commands in container
			a[2] += fmt.Sprintf(" echo '%v\nhelpfile';", prefix)
			a[2] += getHelpFile(AppName)
			a[2] += fmt.Sprintf(" echo '%v';", delimiter)
		}

		if deffile {
			sylog.Debugf("Inspection of deffile selected.")

			// append to a[2] to run commands in container
			a[2] += fmt.Sprintf(" echo '%v\ndeffile';", prefix)
			a[2] += " cat .singularity.d/Singularity;" // apps share common definition file
			a[2] += fmt.Sprintf(" echo '%v';", delimiter)
		}

		if runscript {
			sylog.Debugf("Inspection of runscript selected.")

			// append to a[2] to run commands in container
			a[2] += fmt.Sprintf(" echo '%v\nrunscript';", prefix)
			a[2] += getRunscriptFile(AppName)
			a[2] += fmt.Sprintf(" echo '%v';", delimiter)
		}

		if testfile {
			sylog.Debugf("Inspection of test selected.")

			// append to a[2] to run commands in container
			a[2] += fmt.Sprintf(" echo '%v\ntest';", prefix)
			a[2] += getTestFile(AppName)
			a[2] += fmt.Sprintf(" echo '%v';", delimiter)
		}

		if environment {
			sylog.Debugf("Inspection of environment selected.")

			// append to a[2] to run commands in container
			a[2] += fmt.Sprintf(" echo '%v\nenvironment';", prefix)
			a[2] += getEnvFile(AppName)
			a[2] += fmt.Sprintf(" echo '%v';", delimiter)
		}

		// default to labels if nothing was appended
		if labels || len(a[2]) == 0 {
			sylog.Debugf("Inspection of labels as default.")

			// append to a[2] to run commands in container
			a[2] += fmt.Sprintf(" echo '%v\nlabels';", prefix)
			a[2] += getLabelsFile(AppName)
			a[2] += fmt.Sprintf(" echo '%v';", delimiter)
		}

		fileContents, err := getFileContent(abspath, name, a)
		if err != nil {
			sylog.Fatalf("While getting helpfile: %v", err)
		}

		contentSlice := strings.Split(fileContents, delimiter)
		for _, s := range contentSlice {
			s = strings.TrimSpace(s)
			if strings.HasPrefix(s, prefix) {
				split := strings.SplitN(s, "\n", 3)
				if len(split) == 3 {
					attributes[split[1]] = split[2]
				} else if len(split) == 2 {
					sylog.Warningf("%v metadata was not found.", split[1])
				}
			}
		}

		// format that data based on --json flag
		if jsonfmt {
			// store this in a struct, then marshal the struct to json
			type result struct {
				Data map[string]string `json:"attributes"`
				T    string            `json:"type"`
			}

			d := result{
				Data: attributes,
				T:    "container",
			}

			b, err := json.MarshalIndent(d, "", "\t")
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println(string(b))
		} else {
			// iterate through sections of struct and print them
			for _, value := range attributes {
				fmt.Println("\n" + value + "\n")
			}
		}

	},
	TraverseChildren: true,
}

func getFileContent(abspath, name string, args []string) (string, error) {
	starter := buildcfg.LIBEXECDIR + "/singularity/bin/starter-suid"
	procname := "Singularity inspect"
	Env := []string{sylog.GetEnvVar()}

	engineConfig := singularityConfig.NewConfig()
	ociConfig := &oci.Config{}
	generator := generate.Generator{Config: &ociConfig.Spec}
	engineConfig.OciConfig = ociConfig

	generator.SetProcessArgs(args)
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

	//record from stdout and store as a string to return as the contents of the file?

	cmd, err := exec.PipeCommand(starter, []string{procname}, Env, configData)
	if err != nil {
		sylog.Fatalf("%s: %s", err, cmd)
	}

	b, err := cmd.Output()
	if err != nil {
		sylog.Fatalf("%s: %s", err, b)
	}

	return string(b), nil
}
