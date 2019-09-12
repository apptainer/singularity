// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/opencontainers/runtime-tools/generate"
	"github.com/spf13/cobra"
	"github.com/sylabs/sif/pkg/sif"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/runtime/engine/config"
	"github.com/sylabs/singularity/internal/pkg/runtime/engine/config/oci"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/starter"
	"github.com/sylabs/singularity/pkg/cmdline"
	singularityConfig "github.com/sylabs/singularity/pkg/runtime/engines/singularity/config"
)

const listAppsCommand = "echo apps:`ls \"$app/scif/apps\" | wc -c`; for app in ${SINGULARITY_MOUNTPOINT}/scif/apps/*; do\n    if [ -d \"$app/scif\" ]; then\n        APPNAME=`basename \"$app\"`\n        echo \"$APPNAME\"\n    fi\ndone\n"

var errNoLabelPartition = errors.New("no label partition found")
var errNoSIF = errors.New("invalid SIF")

var (
	runscript   bool
	testfile    bool
	environment bool
	helpfile    bool
	listApps    bool
)

// --list-apps
var inspectAppsListFlag = cmdline.Flag{
	ID:           "inspectAppsListFlag",
	Value:        &listApps,
	DefaultValue: false,
	Name:         "list-apps",
	ShortHand:    "",
	Usage:        "list all apps in a container",
}

// --app
var inspectAppNameFlag = cmdline.Flag{
	ID:           "inspectAppNameFlag",
	Value:        &AppName,
	DefaultValue: "",
	Name:         "app",
	Usage:        "inspect a specific app",
	EnvKeys:      []string{"APP"},
}

// -r|--runscript
var inspectRunscriptFlag = cmdline.Flag{
	ID:           "inspectRunscriptFlag",
	Value:        &runscript,
	DefaultValue: false,
	Name:         "runscript",
	ShortHand:    "r",
	Usage:        "show the runscript for the image",
	EnvKeys:      []string{"RUNSCRIPT"},
}

// -t|--test
var inspectTestFlag = cmdline.Flag{
	ID:           "inspectTestFlag",
	Value:        &testfile,
	DefaultValue: false,
	Name:         "test",
	ShortHand:    "t",
	Usage:        "show the test script for the image",
	EnvKeys:      []string{"TEST"},
}

// -e|--environment
var inspectEnvironmentFlag = cmdline.Flag{
	ID:           "inspectEnvironmentFlag",
	Value:        &environment,
	DefaultValue: false,
	Name:         "environment",
	ShortHand:    "e",
	Usage:        "show the environment settings for the image",
	EnvKeys:      []string{"ENVIRONMENT"},
}

// -H|--helpfile
var inspectHelpfileFlag = cmdline.Flag{
	ID:           "inspectHelpfileFlag",
	Value:        &helpfile,
	DefaultValue: false,
	Name:         "helpfile",
	ShortHand:    "H",
	Usage:        "inspect the runscript helpfile, if it exists",
	EnvKeys:      []string{"HELPFILE"},
}

func init() {
	cmdManager.RegisterCmd(InspectCmd)

	cmdManager.RegisterFlagForCmd(&inspectAppNameFlag, InspectCmd)
	cmdManager.RegisterFlagForCmd(&inspectDeffileFlag, InspectCmd)
	cmdManager.RegisterFlagForCmd(&inspectEnvironmentFlag, InspectCmd)
	cmdManager.RegisterFlagForCmd(&inspectHelpfileFlag, InspectCmd)
	cmdManager.RegisterFlagForCmd(&inspectJSONFlag, InspectCmd)
	cmdManager.RegisterFlagForCmd(&inspectLabelsFlag, InspectCmd)
	cmdManager.RegisterFlagForCmd(&inspectRunscriptFlag, InspectCmd)
	cmdManager.RegisterFlagForCmd(&inspectTestFlag, InspectCmd)
	cmdManager.RegisterFlagForCmd(&inspectAppsListFlag, InspectCmd)
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

func getLabelsCommand() string {
	return getSingleFileCommand("labels.json", "labels", "")
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

func setAttribute(obj *inspectFormat, label, app string, value string) {
	if app == "" {
		app = "system-partition"
	}

	switch label {
	case "apps":
		obj.Data.Attributes.Apps = value
	case "deffile":
		obj.Data.Attributes.Deffile = value
	case "test":
		obj.Data.Attributes.Test = value
	case "helpfile":
		obj.Data.Attributes.Helpfile = value
	case "labels":
		b, _, _, err := jsonparser.Get([]byte(value), app)
		if err != nil {
			sylog.Warningf("Unable to find json from metadata: %s; skipping", err)
			b = []byte(value)
		}

		if err := json.Unmarshal(b, &obj.Data.Attributes.Labels); err != nil {
			sylog.Warningf("Unable to parse labels: %s", err)
		}
	case "runscript":
		obj.Data.Attributes.Runscript = value
	default:
		if strings.HasSuffix(label, "environment.sh") {
			obj.Data.Attributes.Environment = value
		} else {
			sylog.Warningf("Trying to set attribute for unknown label: %s", label)
		}
	}
}

// returns true if flags for other forms of information are unset.
func defaultToLabels() bool {
	return !(helpfile || deffile || runscript || testfile || environment || listApps)
}

func inspectLabelPartition(inspectData *inspectFormat, fimg *sif.FileImage) error {
	if fimg == nil {
		return errNoSIF
	}

	jsonName := AppName
	if jsonName == "" {
		jsonName = "system-partition"
	}

	labelDescriptor, _, err := fimg.GetLinkedDescrsByType(uint32(0), sif.DataLabels)
	if err != nil {
		sylog.Warningf("No metadata partition, searching in container...")
		return errNoLabelPartition
	}

	for _, v := range labelDescriptor {
		metaData := v.GetData(fimg)
		b, _, _, err := jsonparser.Get(metaData, jsonName)
		if err != nil {
			return fmt.Errorf("unable to find json from metadata: %s", err)
		}
		var hrOut map[string]json.RawMessage
		err = json.Unmarshal(b, &hrOut)
		if err != nil {
			return fmt.Errorf("unable to get json: %s", err)
		}

		for k, v := range hrOut {
			value := string(v)
			// Only remove the extra quotes if json output.
			if jsonfmt {
				var err error
				value, err = strconv.Unquote(value)
				if err != nil {
					return fmt.Errorf("unable to remove quotes from data: %s", err)
				}
			}
			inspectData.Data.Attributes.Labels[k] = value
		}
	}

	return nil
}

func inspectDeffilePartition(inspectData *inspectFormat, fimg *sif.FileImage) error {
	if fimg == nil {
		return errNoSIF
	}

	labelDescriptor, _, err := fimg.GetLinkedDescrsByType(uint32(0), sif.DataDeffile)
	if err != nil {
		sylog.Warningf("No metadata partition, searching in container...")
		return errNoLabelPartition
	}

	for _, v := range labelDescriptor {
		metaData := v.GetData(fimg)
		data := string(metaData)
		inspectData.Data.Attributes.Deffile = data
	}

	return nil
}

// InspectCmd represents the 'inspect' command.
// TODO: This should be in its own package, not cli.
var InspectCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(1),

	Use:     docs.InspectUse,
	Short:   docs.InspectShort,
	Long:    docs.InspectLong,
	Example: docs.InspectExample,

	Run: func(cmd *cobra.Command, args []string) {
		f, err := os.Stat(args[0])
		if os.IsNotExist(err) {
			sylog.Fatalf("Container not found: %s\n", err)
		} else if err != nil {
			sylog.Fatalf("Unable to stat file: %s", err)
		}
		sandboxImage := f.IsDir()

		var fimg sif.FileImage
		if !sandboxImage {
			var err error
			fimg, err = sif.LoadContainer(args[0], true)
			if err != nil {
				sylog.Fatalf("failed to load SIF container file: %s", err)
			}
			defer fimg.UnloadContainer()
		}

		var inspectData inspectFormat
		inspectData.Type = containerType
		inspectData.Data.Attributes.Labels = make(map[string]string, 1)

		inspectCMD := []string{"/bin/sh", "-c", ""}

		// Inspect Labels.
		if labels || defaultToLabels() {
			err := inspectLabelPartition(&inspectData, &fimg)
			if err == errNoLabelPartition || err == errNoSIF {
				sylog.Debugf("Inspection of labels selected.")
				inspectCMD[2] += getLabelsCommand()
			} else if err != nil {
				sylog.Fatalf("Unable to inspect container: %s", err)
			}
		}

		// Inspect Deffile.
		if deffile {
			err := inspectDeffilePartition(&inspectData, &fimg)
			if err == errNoLabelPartition || err == errNoSIF {
				sylog.Debugf("Inspection of deffile selected.")
				inspectCMD[2] += getDefinitionCommand()
			} else if err != nil {
				sylog.Fatalf("Unable to inspect deffile: %s", err)
			}
		}

		if listApps {
			sylog.Debugf("Listing all apps in container")
			inspectCMD[2] += listAppsCommand
		}

		if helpfile {
			sylog.Debugf("Inspection of helpfile selected.")
			inspectCMD[2] += getHelpCommand(AppName)
		}

		if runscript {
			sylog.Debugf("Inspection of runscript selected.")
			inspectCMD[2] += getRunscriptCommand(AppName)
		}

		if testfile {
			sylog.Debugf("Inspection of test selected.")
			inspectCMD[2] += getTestCommand(AppName)
		}

		if environment {
			sylog.Debugf("Inspection of environment selected.")
			inspectCMD[2] += getEnvironmentCommand(AppName)
		}

		if inspectCMD[2] != "" {
			abspath, err := filepath.Abs(args[0])
			if err != nil {
				sylog.Fatalf("While determining absolute file path: %v", err)
			}
			name := filepath.Base(abspath)

			// Execute the compound command string.
			fileContents, err := getFileContent(abspath, name, inspectCMD)
			if err != nil {
				sylog.Fatalf("Could not inspect container: %v", err)
			}

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
					setAttribute(&inspectData, label, AppName, string(data))
				} else {
					sylog.Fatalf("Badly formatted content, can't recover: %v", parts)
				}
			}
		}

		// Output the inspection results (use JSON if requested).
		if jsonfmt {
			jsonObj, err := json.MarshalIndent(inspectData, "", "\t")
			if err != nil {
				sylog.Fatalf("Could not format inspected data as JSON")
			}
			fmt.Printf("%s\n", string(jsonObj))
		} else {
			if inspectData.Data.Attributes.Apps != "" {
				fmt.Printf("%s\n", inspectData.Data.Attributes.Apps)
			}
			if inspectData.Data.Attributes.Helpfile != "" {
				fmt.Printf("%s\n", inspectData.Data.Attributes.Helpfile)
			}
			if inspectData.Data.Attributes.Deffile != "" {
				fmt.Printf("%s\n", inspectData.Data.Attributes.Deffile)
			}
			if inspectData.Data.Attributes.Runscript != "" {
				fmt.Printf("%s\n", inspectData.Data.Attributes.Runscript)
			}
			if inspectData.Data.Attributes.Test != "" {
				fmt.Printf("%s\n", inspectData.Data.Attributes.Test)
			}
			if len(inspectData.Data.Attributes.Environment) > 0 {
				fmt.Printf("%s\n", inspectData.Data.Attributes.Environment)
			}
			if len(inspectData.Data.Attributes.Labels) > 0 {
				// Sort the labels.
				var labelSort []string
				for k := range inspectData.Data.Attributes.Labels {
					labelSort = append(labelSort, k)
				}
				sort.Strings(labelSort)

				for _, k := range labelSort {
					fmt.Printf("%s: %s\n", k, inspectData.Data.Attributes.Labels[k])
				}
			}
		}
	},
	TraverseChildren: true,
}

func getFileContent(abspath, name string, args []string) (string, error) {
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

	// Record from stdout and store as a string to return as the contents of the file.
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := starter.Run(
		"Singularity inspect",
		cfg,
		starter.UseSuid(true),
		starter.WithStdout(&stdout),
		starter.WithStderr(&stderr),
	)
	if err != nil {
		sylog.Fatalf("Unable to process command: %s: %s", err, stderr.String())
	}

	return stdout.String(), nil
}
