// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/opencontainers/runtime-tools/generate"
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/exec"

	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config/oci"
	singularityConfig "github.com/sylabs/singularity/pkg/runtime/engines/singularity/config"
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

type inspectAttributes struct {
	Labels      map[string]string `json:"labels,omitempty"`
	Deffile     string            `json:"deffile,omitempty"`
	Runscript   string            `json:"runscript,omitempty"`
	Test        string            `json:"test,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	Helpfile    string            `json:"helpfile,omitempty"`
}

type inspectFormat struct {
	Attributes inspectAttributes `json:"attributes"`
	Type       string            `json:"type"`
}

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

func getPathPrefix(appName string) string {
	if appName == "" {
		return "/.singularity.d"
	}
	return fmt.Sprintf("/scif/apps/%s/scif", appName)
}

func getSingleFileCommand(file string, label string, appName string) string {
	var str strings.Builder
	str.WriteString(fmt.Sprintf(" if [ -f %s/%s ]; then", getPathPrefix(appName), file))
	str.WriteString(fmt.Sprintf("     echo %s:`wc -c < %s/%s`;", label, getPathPrefix(appName), file))
	str.WriteString(fmt.Sprintf("     cat %s/%s;", getPathPrefix(appName), file))
	str.WriteString(" fi;")
	return str.String()
}

func getLabelsCommand(appName string) string {
	return getSingleFileCommand("labels.json", "labels", appName)
}

func getDefinitionCommand() string {
	return getSingleFileCommand("Singularity", "deffile", "")
}

func getRunscriptCommand(appName string) string {
	return getSingleFileCommand("runscript", "runscript", appName)
}

func getTestCommand(appName string) string {
	return getSingleFileCommand("test", "test", appName)
}

func getEnvironmentCommand(appName string) string {
	var str strings.Builder
	str.WriteString(" for env in %s/env/9*-environment.sh; do")
	str.WriteString("     echo ${env##*/}:`wc -c < $env`;")
	str.WriteString("     cat $env;")
	str.WriteString(" done;")
	return fmt.Sprintf(str.String(), getPathPrefix(appName))
}

func getHelpCommand(appName string) string {
	return getSingleFileCommand("runscript.help", "helpfile", appName)
}

func setAttribute(obj *inspectFormat, label string, value string) {
	switch label {
	case "deffile":
		obj.Attributes.Deffile = value
	case "test":
		obj.Attributes.Test = value
	case "helpfile":
		obj.Attributes.Helpfile = value
	case "labels":
		if err := json.Unmarshal([]byte(value), &obj.Attributes.Labels); err != nil {
			sylog.Warningf("Unable to parse labels: %s", value)
		}
	case "runscript":
		obj.Attributes.Runscript = value
	default:
		if strings.HasSuffix(label, "environment.sh") {
			obj.Attributes.Environment[label] = value
		} else {
			sylog.Warningf("Trying to set attribute for unknown label: %s", label)
		}
	}
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

		a := []string{"/bin/sh", "-c", ""}

		// If AppName is given fail quickly (exit) if it doesn't exist
		if AppName != "" {
			sylog.Debugf("Inspection of App %s Selected.", AppName)
			a[2] += getAppCheck(AppName)
		}

		if helpfile {
			sylog.Debugf("Inspection of helpfile selected.")
			a[2] += getHelpCommand(AppName)
		}

		if deffile {
			sylog.Debugf("Inspection of deffile selected.")
			a[2] += getDefinitionCommand()
		}

		if runscript {
			sylog.Debugf("Inspection of runscript selected.")
			a[2] += getRunscriptCommand(AppName)
		}

		if testfile {
			sylog.Debugf("Inspection of test selected.")
			a[2] += getTestCommand(AppName)
		}

		if environment {
			sylog.Debugf("Inspection of environment selected.")
			a[2] += getEnvironmentCommand(AppName)
		}

		// Default to labels if nothing was appended
		if labels || len(a[2]) == 0 {
			sylog.Debugf("Inspection of labels selected.")
			a[2] += getLabelsCommand(AppName)
		}

		// Execute the compound command string.
		fileContents, err := getFileContent(abspath, name, a)
		if err != nil {
			sylog.Fatalf("Could not inspect container: %v", err)
		}

		inspectObj := inspectFormat{}
		inspectObj.Type = "container"
		inspectObj.Attributes.Labels = make(map[string]string)
		inspectObj.Attributes.Environment = make(map[string]string)

		// Parse the command output string into sections.
		reader := bufio.NewReader(strings.NewReader(fileContents))
		for {
			section, err := reader.ReadBytes('\n')
			if err != nil {
				break
			}
			parts := strings.SplitN(strings.TrimSpace(string(section)), ":", 3)
			if len(parts) == 2 {
				label := parts[0]
				sizeData, errConv := strconv.Atoi(parts[1])
				if errConv != nil {
					sylog.Fatalf("Badly formatted content, can't recover: %v", parts)
				}
				sylog.Debugf("Section %s found with %d bytes of data.", label, sizeData)
				data := make([]byte, sizeData)
				n, err := io.ReadFull(reader, data)
				if n != len(data) && err != nil {
					sylog.Fatalf("Unable to read %d bytes.", sizeData)
				}
				setAttribute(&inspectObj, label, string(data))
			} else {
				sylog.Fatalf("Badly formatted content, can't recover: %v", parts)
			}
		}

		// Output the inspection results (use JSON if requested).
		if jsonfmt {
			jsonObj, err := json.MarshalIndent(inspectObj, "", "\t")
			if err != nil {
				sylog.Fatalf("Could not format inspected data as JSON.")
			}
			fmt.Println(string(jsonObj))
		} else {
			if inspectObj.Attributes.Helpfile != "" {
				fmt.Println("==helpfile==\n" + inspectObj.Attributes.Helpfile)
			}
			if inspectObj.Attributes.Deffile != "" {
				fmt.Println("==deffile==\n" + inspectObj.Attributes.Deffile)
			}
			if inspectObj.Attributes.Runscript != "" {
				fmt.Println("==runscript==\n" + inspectObj.Attributes.Runscript)
			}
			if inspectObj.Attributes.Test != "" {
				fmt.Println("==test==\n" + inspectObj.Attributes.Test)
			}
			if len(inspectObj.Attributes.Environment) > 0 {
				fmt.Println("==environment==")
				for envLabel, envValue := range inspectObj.Attributes.Environment {
					fmt.Println("==environment:" + envLabel + "==\n" + envValue)
				}
			}
			if len(inspectObj.Attributes.Labels) > 0 {
				fmt.Println("==labels==")
				for labLabel, labValue := range inspectObj.Attributes.Labels {
					fmt.Println(labLabel + ": " + labValue)
				}
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
		sylog.Fatalf("%s: %s", err, cmd.Args)
	}

	b, err := cmd.Output()
	if err != nil {
		sylog.Fatalf("%s: %s", err, b)
	}

	return string(b), nil
}
