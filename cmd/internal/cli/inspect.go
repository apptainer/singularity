// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sylabs/sif/pkg/sif"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/util/env"
	"github.com/sylabs/singularity/pkg/cmdline"
	"github.com/sylabs/singularity/pkg/image"
	"github.com/sylabs/singularity/pkg/inspect"
	"github.com/sylabs/singularity/pkg/sylog"
)

var errNoSIFMetadata = errors.New("no SIF metadata found")
var errNoSIF = errors.New("invalid SIF")

var (
	allData     bool
	runscript   bool
	startscript bool
	testfile    bool
	environment bool
	helpfile    bool
	listApps    bool
	labels      bool
	deffile     bool
	jsonfmt     bool
)

// -l|--labels
var inspectLabelsFlag = cmdline.Flag{
	ID:           "inspectLabelsFlag",
	Value:        &labels,
	DefaultValue: false,
	Name:         "labels",
	ShortHand:    "l",
	Usage:        "show the labels for the image (default)",
}

// -d|--deffile
var inspectDeffileFlag = cmdline.Flag{
	ID:           "inspectDeffileFlag",
	Value:        &deffile,
	DefaultValue: false,
	Name:         "deffile",
	ShortHand:    "d",
	Usage:        "show the Singularity recipe file that was used to generate the image",
}

// -j|--json
var inspectJSONFlag = cmdline.Flag{
	ID:           "inspectJSONFlag",
	Value:        &jsonfmt,
	DefaultValue: false,
	Name:         "json",
	ShortHand:    "j",
	Usage:        "print structured json instead of sections",
}

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
}

// -r|--runscript
var inspectRunscriptFlag = cmdline.Flag{
	ID:           "inspectRunscriptFlag",
	Value:        &runscript,
	DefaultValue: false,
	Name:         "runscript",
	ShortHand:    "r",
	Usage:        "show the runscript for the image",
}

// -s|--startscript
var inspectStartscriptFlag = cmdline.Flag{
	ID:           "inspectStartscriptFlag",
	Value:        &startscript,
	DefaultValue: false,
	Name:         "startscript",
	ShortHand:    "s",
	Usage:        "show the startscript for the image",
}

// -t|--test
var inspectTestFlag = cmdline.Flag{
	ID:           "inspectTestFlag",
	Value:        &testfile,
	DefaultValue: false,
	Name:         "test",
	ShortHand:    "t",
	Usage:        "show the test script for the image",
}

// -e|--environment
var inspectEnvironmentFlag = cmdline.Flag{
	ID:           "inspectEnvironmentFlag",
	Value:        &environment,
	DefaultValue: false,
	Name:         "environment",
	ShortHand:    "e",
	Usage:        "show the environment settings for the image",
}

// -H|--helpfile
var inspectHelpfileFlag = cmdline.Flag{
	ID:           "inspectHelpfileFlag",
	Value:        &helpfile,
	DefaultValue: false,
	Name:         "helpfile",
	ShortHand:    "H",
	Usage:        "inspect the runscript helpfile, if it exists",
}

// --all
var inspectAllFlag = cmdline.Flag{
	ID:           "inspectAllFlag",
	Value:        &allData,
	DefaultValue: false,
	Name:         "all",
	Usage:        "show all available data (imply --json option)",
}

func init() {
	addCmdInit(func(cmdManager *cmdline.CommandManager) {
		cmdManager.RegisterCmd(InspectCmd)

		cmdManager.RegisterFlagForCmd(&inspectAppNameFlag, InspectCmd)
		cmdManager.RegisterFlagForCmd(&inspectDeffileFlag, InspectCmd)
		cmdManager.RegisterFlagForCmd(&inspectEnvironmentFlag, InspectCmd)
		cmdManager.RegisterFlagForCmd(&inspectHelpfileFlag, InspectCmd)
		cmdManager.RegisterFlagForCmd(&inspectJSONFlag, InspectCmd)
		cmdManager.RegisterFlagForCmd(&inspectLabelsFlag, InspectCmd)
		cmdManager.RegisterFlagForCmd(&inspectRunscriptFlag, InspectCmd)
		cmdManager.RegisterFlagForCmd(&inspectStartscriptFlag, InspectCmd)
		cmdManager.RegisterFlagForCmd(&inspectTestFlag, InspectCmd)
		cmdManager.RegisterFlagForCmd(&inspectAppsListFlag, InspectCmd)
		cmdManager.RegisterFlagForCmd(&inspectAllFlag, InspectCmd)
	})
}

const sectionDelim = "~~##@@> "

type command struct {
	script   string
	metadata *inspect.Metadata
	img      *image.Image
}

func newCommand(allData bool, appName string, img *image.Image) *command {
	command := new(command)
	command.metadata = inspect.NewMetadata()
	command.img = img

	prefix := ""
	if img.Type == image.SANDBOX {
		prefix = img.Path
	}

	pathPrefix := filepath.Join(prefix, "/.singularity.d")
	if appName != "" && !allData {
		pathPrefix = fmt.Sprintf("%s/scif/apps/%s/scif", prefix, appName)
	}
	allVar := ""
	if allData {
		allVar = "ALL_DATA=1"
	}

	var snippet = `%s
	for app in %s/scif/apps/*; do
	if [ -d "$app/scif" ]; then
		echo "%s apps"
		echo "${app##*/}"
		if [ ! -z "${ALL_DATA}" ]; then
			if [ -z "${ALL_PATH}" ]; then
				ALL_PATH="$app/scif"
			else
				ALL_PATH="${ALL_PATH}:$app/scif"
			fi
		fi
	fi
	done

	ALL_PATH="%s:${ALL_PATH}"

	IFS=":"
	`

	command.script = fmt.Sprintf(snippet, allVar, prefix, sectionDelim, pathPrefix)
	return command
}

func (c *command) setAttribute(section, value, file string) error {
	sylog.Debugf("Section %s found", section)
	value = strings.TrimRight(value, "\n")

	app := ""
	if file != "" {
		fmt.Sscanf(file, "/scif/apps/%s", &app)
		if app != "" {
			app = strings.Split(app, "/")[0]
		}
	}

	switch section {
	case "apps":
		c.metadata.AddApp(value)
	case "deffile":
		if c.metadata.Data.Attributes.Deffile == "" {
			c.metadata.Data.Attributes.Deffile = value
		}
	case "test":
		if app != "" {
			c.metadata.Data.Attributes.Apps[app].Test = value
		} else {
			c.metadata.Data.Attributes.Test = value
		}
	case "helpfile":
		if app != "" {
			c.metadata.Data.Attributes.Apps[app].Helpfile = value
		} else {
			c.metadata.Data.Attributes.Helpfile = value
		}
	case "labels":
		labels := make(map[string]string)
		if err := json.Unmarshal([]byte(value), &labels); err != nil {
			sylog.Warningf("Unable to parse labels: %s", err)
		}
		if app != "" {
			c.metadata.Data.Attributes.Apps[app].Labels = labels
		} else {
			c.metadata.Data.Attributes.Labels = labels
		}
	case "runscript":
		if app != "" {
			c.metadata.Data.Attributes.Apps[app].Runscript = value
		} else {
			c.metadata.Data.Attributes.Runscript = value
		}
	case "startscript":
		c.metadata.Data.Attributes.Startscript = value
	case "environment":
		if app != "" {
			c.metadata.Data.Attributes.Apps[app].Environment[file] = value
		} else {
			c.metadata.Data.Attributes.Environment[file] = value
		}
	default:
		return fmt.Errorf("badly formatted content, unknown section %s", section)
	}
	return nil
}

func (c *command) getMetadata() (*inspect.Metadata, error) {
	args := []string{"/bin/sh", "-c", c.script}
	prefix := ""
	outBuf := new(bytes.Buffer)

	// Execute the compound script.
	if c.img.Type == image.SANDBOX {
		os.Setenv("PATH", env.DefaultPath)

		// look for sh
		shell, err := exec.LookPath("sh")
		if err != nil {
			return nil, fmt.Errorf("could not inspect container: sh command not found in %q", env.DefaultPath)
		}
		args[0] = shell

		// look for cat command
		_, err = exec.LookPath("cat")
		if err != nil {
			return nil, fmt.Errorf("could not inspect container: cat command not found in %q", env.DefaultPath)
		}

		cmd := exec.Command(args[0], args[1:]...)
		cmd.Env = []string{"PATH=" + env.DefaultPath}
		out, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("could not inspect container: %v", err)
		}
		outBuf.Write(out)
		prefix = c.img.Path
	} else {
		// single file image, run singularity exec with the compound script
		out, err := singularityExec(c.img.Path, args)
		if err != nil {
			return nil, fmt.Errorf("could not inspect container: %v", err)
		}
		outBuf.WriteString(out)
	}

	prevSection := ""
	prevFile := ""
	buf := new(bytes.Buffer)

	// Parse the command output string into sections.
	for {
		section, err := outBuf.ReadBytes('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("while reading formatted content: %s", err)
		}
		sectionStr := strings.TrimSpace(string(section))

		if strings.HasPrefix(sectionStr, sectionDelim) {
			sectionStr = strings.TrimSpace(strings.TrimPrefix(sectionStr, sectionDelim))
			parts := strings.SplitN(sectionStr, ":", 2)
			if len(parts) < 1 {
				return nil, fmt.Errorf("badly formatted content, can't recover: %v", parts)
			}
			if prevSection != "" {
				err := c.setAttribute(prevSection, buf.String(), strings.TrimPrefix(prevFile, prefix))
				if err != nil {
					return nil, err
				}
			}
			buf.Reset()
			prevSection = parts[0]
			prevFile = ""
			if len(parts) == 2 {
				prevFile = parts[1]
			}
		} else {
			buf.Write(section)
		}
	}

	// write the remaining section if any
	if prevSection != "" {
		err := c.setAttribute(prevSection, buf.String(), strings.TrimPrefix(prevFile, prefix))
		if err != nil {
			return nil, err
		}
	}

	return c.metadata, nil
}

func (c *command) addSingleFileCommand(file string, label string) {
	var snippet = `
	for prefix in ${ALL_PATH}; do
		file="$prefix/%s"
		if [ -f "$file" ]; then
			echo "%s %s:$file"
			cat $file
			echo ""
		fi
	done
	`
	c.script += fmt.Sprintf(snippet, file, sectionDelim, label)
}

func (c *command) addLabelsCommand() {
	c.addSingleFileCommand("labels.json", "labels")
}

func (c *command) addRunscriptCommand() {
	c.addSingleFileCommand("runscript", "runscript")
}

func (c *command) addStartscriptCommand() {
	c.addSingleFileCommand("startscript", "startscript")
}

func (c *command) addTestCommand() {
	c.addSingleFileCommand("test", "test")
}

func (c *command) addHelpCommand() {
	c.addSingleFileCommand("runscript.help", "helpfile")
}

func (c *command) addEnvironmentCommand() {
	var snippet = `
	for prefix in ${ALL_PATH}; do
		if [ "${prefix##*/}" = ".singularity.d" ]; then
			for env in $prefix/env/10-docker*.sh; do
				if [ -f "$env" ]; then
					echo "%[1]s environment:$env"
					cat $env
					echo ""
				fi
			done
		fi

		for env in $prefix/env/9*-environment.sh; do
			if [ -f "$env" ]; then
				echo "%[1]s environment:$env"
				cat $env
				echo ""
			fi
		done
	done
	`
	c.script += fmt.Sprintf(snippet, sectionDelim)
}

func (c *command) addDefinitionCommand() {
	var err error

	c.metadata.Attributes.Deffile, err = inspectDeffilePartition(c.img)
	if err == errNoSIFMetadata || err == errNoSIF {
		c.addSingleFileCommand("Singularity", "deffile")
	} else if err != nil {
		sylog.Warningf("Unable to inspect deffile: %s", err)
	}
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

func inspectDeffilePartition(img *image.Image) (string, error) {
	data, err := getSIFMetadata(img, uint32(sif.DataDeffile))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func printSortedApp(m map[string]*inspect.AppAttributes) {
	sorted := make([]string, 0, len(m))
	for k := range m {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)
	for _, k := range sorted {
		fmt.Printf("%s\n", k)
	}
}

func printSortedMap(m map[string]string, fn func(key string)) {
	sorted := make([]string, 0, len(m))
	for k := range m {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)
	for _, k := range sorted {
		fn(k)
	}
}

// returns true if flags for other forms of information are unset.
func defaultToLabels() bool {
	return !(helpfile || deffile || runscript || startscript || testfile || environment || listApps)
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

		if allData {
			// display all data in JSON format only
			jsonfmt = true
			AppName = ""
		}

		inspectCmd := newCommand(allData, AppName, img)

		// Try to inspect the label partition, if not, then exec/shell
		// the container to get the data.
		if labels || defaultToLabels() || allData {
			// If '--app' is specified, then we need to shell/exec the
			// container.
			sylog.Debugf("Inspection of labels selected.")
			inspectCmd.addLabelsCommand()
		}

		// Inspect the deffile.
		if deffile || allData {
			sylog.Debugf("Inspection of deffile selected.")
			inspectCmd.addDefinitionCommand()
		}

		if helpfile || allData {
			sylog.Debugf("Inspection of helpfile selected.")
			inspectCmd.addHelpCommand()
		}

		if runscript || allData {
			sylog.Debugf("Inspection of runscript selected.")
			inspectCmd.addRunscriptCommand()
		}

		if startscript || allData {
			if AppName == "" {
				sylog.Debugf("Inspection of startscript selected.")
				inspectCmd.addStartscriptCommand()
			}
		}

		if testfile || allData {
			sylog.Debugf("Inspection of test selected.")
			inspectCmd.addTestCommand()
		}

		if environment || allData {
			sylog.Debugf("Inspection of environment selected.")
			inspectCmd.addEnvironmentCommand()
		}

		if listApps || allData {
			sylog.Debugf("Listing all apps in container")
		}

		inspectData, err := inspectCmd.getMetadata()
		if err != nil {
			sylog.Fatalf("%s", err)
		}

		for app := range inspectData.Data.Attributes.Apps {
			if !listApps && !allData && AppName != app {
				delete(inspectData.Data.Attributes.Apps, app)
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
			appAttr := inspectData.Data.Attributes.Apps[AppName]

			if listApps {
				printSortedApp(inspectData.Data.Attributes.Apps)
			}

			if inspectData.Data.Attributes.Deffile != "" {
				fmt.Printf("%s\n", inspectData.Data.Attributes.Deffile)
			}
			if inspectData.Data.Attributes.Runscript != "" {
				fmt.Printf("%s\n", inspectData.Data.Attributes.Runscript)
			} else if appAttr != nil && appAttr.Runscript != "" {
				fmt.Printf("%s\n", appAttr.Runscript)
			}
			if inspectData.Data.Attributes.Startscript != "" {
				fmt.Printf("%s\n", inspectData.Data.Attributes.Startscript)
			}
			if inspectData.Data.Attributes.Test != "" {
				fmt.Printf("%s\n", inspectData.Data.Attributes.Test)
			} else if appAttr != nil && appAttr.Test != "" {
				fmt.Printf("%s\n", appAttr.Test)
			}
			if inspectData.Data.Attributes.Helpfile != "" {
				fmt.Printf("%s\n", inspectData.Data.Attributes.Helpfile)
			} else if appAttr != nil && appAttr.Helpfile != "" {
				fmt.Printf("%s\n", appAttr.Helpfile)
			}
			if len(inspectData.Data.Attributes.Environment) > 0 {
				printSortedMap(inspectData.Data.Attributes.Environment, func(k string) {
					fmt.Printf("=== %s ===\n%s\n\n", k, inspectData.Data.Attributes.Environment[k])
				})
			} else if appAttr != nil && len(appAttr.Environment) > 0 {
				printSortedMap(appAttr.Environment, func(k string) {
					fmt.Printf("=== %s ===\n%s\n\n", k, appAttr.Environment[k])
				})
			}
			if len(inspectData.Data.Attributes.Labels) > 0 {
				printSortedMap(inspectData.Data.Attributes.Labels, func(k string) {
					fmt.Printf("%s: %s\n", k, inspectData.Data.Attributes.Labels[k])
				})
			} else if appAttr != nil && len(appAttr.Labels) > 0 {
				printSortedMap(appAttr.Labels, func(k string) {
					fmt.Printf("%s: %s\n", k, appAttr.Labels[k])
				})
			}
		}
	},
	TraverseChildren: true,
}
