// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sylabs/sif/pkg/sif"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/cmdline"
	"github.com/sylabs/singularity/pkg/image"
)

const listAppsCommand = "echo apps:`ls \"$app/scif/apps\" | wc -c`; for app in ${SINGULARITY_MOUNTPOINT}/scif/apps/*; do\n    if [ -d \"$app/scif\" ]; then\n        APPNAME=`basename \"$app\"`\n        echo \"$APPNAME\"\n    fi\ndone\n"

var errNoSIFMetadata = errors.New("no SIF metadata found")
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

func setAttribute(obj *inspectFormat, label, app, value string) {
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
		if err := json.Unmarshal([]byte(value), &obj.Data.Attributes.Labels); err != nil {
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

func getSIFMetadata(img *image.Image, dataType uint32) ([]byte, error) {
	if img.Type != image.SIF {
		return nil, errNoSIF
	}

	for i, section := range img.Sections {
		if section.Type != dataType {
			continue
		}
		r, err := image.NewSectionReader(img, "", i)
		if err != nil {
			return nil, fmt.Errorf("while reading SIF section: %s", err)
		}
		b, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, fmt.Errorf("while reading metadata: %s", err)
		}
		return b, nil
	}

	sylog.Warningf("No SIF metadata partition, searching in container...")
	return nil, errNoSIFMetadata
}

func inspectLabelPartition(inspectData *inspectFormat, img *image.Image) error {
	data, err := getSIFMetadata(img, uint32(sif.DataLabels))
	if err != nil {
		return err
	}

	var hrOut map[string]json.RawMessage
	err = json.Unmarshal(data, &hrOut)
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

	return nil
}

func inspectDeffilePartition(inspectData *inspectFormat, img *image.Image) error {
	data, err := getSIFMetadata(img, uint32(sif.DataDeffile))
	if err != nil {
		return err
	}

	inspectData.Data.Attributes.Deffile = string(data)
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
		img, err := image.Init(args[0], false)
		if err != nil {
			sylog.Fatalf("Failed to open image %s: %s", args[0], err)
		}

		var inspectData inspectFormat
		inspectData.Type = containerType
		inspectData.Data.Attributes.Labels = make(map[string]string, 1)

		inspectShellCmd := []string{"/bin/sh", "-c", ""}

		// Try to inspect the label partition, if not, then exec/shell
		// the container to get the data.
		if labels || defaultToLabels() {
			if AppName == "" {
				err := inspectLabelPartition(&inspectData, img)
				if err == errNoSIFMetadata || err == errNoSIF {
					sylog.Debugf("Cant get label partition, looking in container...")
					inspectShellCmd[2] += getLabelsCommand(AppName)
				} else if err != nil {
					sylog.Fatalf("Unable to inspect container: %s", err)
				}
			} else {
				// If '--app' is specified, then we need to shell/exec the
				// container.
				sylog.Debugf("Inspection of labels selected.")
				inspectShellCmd[2] += getLabelsCommand(AppName)
			}
		}

		// Inspect the deffile.
		if deffile {
			err := inspectDeffilePartition(&inspectData, img)
			if err == errNoSIFMetadata || err == errNoSIF {
				sylog.Debugf("Inspection of deffile selected.")
				inspectShellCmd[2] += getDefinitionCommand()
			} else if err != nil {
				sylog.Fatalf("Unable to inspect deffile: %s", err)
			}
		}

		if listApps {
			sylog.Debugf("Listing all apps in container")
			inspectShellCmd[2] += listAppsCommand
		}

		if helpfile {
			sylog.Debugf("Inspection of helpfile selected.")
			inspectShellCmd[2] += getHelpCommand(AppName)
		}

		if runscript {
			sylog.Debugf("Inspection of runscript selected.")
			inspectShellCmd[2] += getRunscriptCommand(AppName)
		}

		if testfile {
			sylog.Debugf("Inspection of test selected.")
			inspectShellCmd[2] += getTestCommand(AppName)
		}

		if environment {
			sylog.Debugf("Inspection of environment selected.")
			inspectShellCmd[2] += getEnvironmentCommand(AppName)
		}

		if inspectShellCmd[2] != "" {
			// Execute the compound command string.
			fileContents, err := singularityExec(args[0], inspectShellCmd)
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
