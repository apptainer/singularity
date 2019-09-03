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
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/opencontainers/runtime-tools/generate"
	"github.com/spf13/cobra"
	"github.com/sylabs/sif/pkg/sif"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config/oci"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/exec"
	"github.com/sylabs/singularity/pkg/build/types/parser"
	"github.com/sylabs/singularity/pkg/cmdline"
	singularityConfig "github.com/sylabs/singularity/pkg/runtime/engines/singularity/config"
)

// TODO: this should not be in a cli package

var ErrNoMetaData = errors.New("no metadata found for system partition")

const listAppsCommand = "echo apps:`ls \"$app/scif/apps\" | wc -c`; for app in ${SINGULARITY_MOUNTPOINT}/scif/apps/*; do\n    if [ -d \"$app/scif\" ]; then\n        APPNAME=`basename \"$app\"`\n        echo \"$APPNAME\"\n    fi\ndone\n"

var (
	labels      bool
	deffile     bool
	runscript   bool
	testfile    bool
	environment bool
	helpfile    bool
	jsonfmt     bool
	listApps    bool
)

type inspectMetadata struct {
	Apps        string            `json:"apps,omitempty"`
	AppLabels   string            `json:"apps-labels,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Deffile     string            `json:"deffile,omitempty"`
	Runscript   string            `json:"runscript,omitempty"`
	Test        string            `json:"test,omitempty"`
	Environment string            `json:"environment,omitempty"`
	//Environment map[string]string `json:"environment,omitempty"`
	Helpfile string `json:"helpfile,omitempty"`
}

type inspectData struct {
	Attributes inspectMetadata `json:"attributes"`
}

type inspectFormat struct {
	Data inspectData `json:"data"`
	Type string      `json:"type"`
}

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

// -l|--labels
var inspectLabelsFlag = cmdline.Flag{
	ID:           "inspectLabelsFlag",
	Value:        &labels,
	DefaultValue: false,
	Name:         "labels",
	ShortHand:    "l",
	Usage:        "show the labels associated with the image (default)",
	EnvKeys:      []string{"LABELS"},
}

// -d|--deffile
var inspectDeffileFlag = cmdline.Flag{
	ID:           "inspectDeffileFlag",
	Value:        &deffile,
	DefaultValue: false,
	Name:         "deffile",
	ShortHand:    "d",
	Usage:        "show the Singularity recipe file that was used to generate the image",
	EnvKeys:      []string{"DEFFILE"},
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

// -j|--json
var inspectJSONFlag = cmdline.Flag{
	ID:           "inspectJSONFlag",
	Value:        &jsonfmt,
	DefaultValue: false,
	Name:         "json",
	ShortHand:    "j",
	Usage:        "print structured json instead of sections",
	EnvKeys:      []string{"JSON"},
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

func setAttribute(obj *inspectFormat, label string, value string) {
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
			sylog.Warningf("Unable to parse labels: %s", value)
		}
	case "runscript":
		obj.Data.Attributes.Runscript = value
	default:
		if strings.HasSuffix(label, "environment.sh") {
			//obj.Data.Attributes.Environment[label] = value
			obj.Data.Attributes.Environment = value
		} else {
			sylog.Warningf("Trying to set attribute for unknown label: %s", label)
		}
	}
}

func getAppCheck(appName string) string {
	return fmt.Sprintf("if ! [ -d \"/scif/apps/%s\" ]; then echo \"App %s does not exist.\"; exit 2; fi;", appName, appName)
}

// returns true if flags for other forms of information are unset
func defaultToLabels() bool {
	return !(helpfile || deffile || runscript || testfile || environment || listApps)
}

// getMetaData will return a dataType
func getMetaData(fimg *sif.FileImage, dataType sif.Datatype) (sigs []*sif.Descriptor, descr []*sif.Descriptor, err error) {
	descr = make([]*sif.Descriptor, 1)

	descr[0], _, err = fimg.GetPartPrimSys()
	if err != nil {
		return nil, nil, fmt.Errorf("no primary partition found")
	}

	// GetFromDescrID
	sigs, _, err = fimg.GetLinkedDescrsByType(uint32(0), dataType)
	if err != nil {
		return nil, nil, ErrNoMetaData
	}

	return
}

// parseDeffile will take a deffile content, appname and a section, and
// return the specified section. Returns "" if the section does not exist.
func parseDeffile(content, app, section string) string {
	ret := ""
	headerFound := false
	end := false
	searchSection := ""

	if app != "" {
		switch section {
		case "help":
			searchSection = "%apphelp " + app
		case "runscript":
			searchSection = "%apprun " + app
		case "test":
			searchSection = "%apptest " + app
		case "environment":
			searchSection = "%appenv " + app
		case "labels":
			searchSection = "%applabels " + app
		}
	} else {
		searchSection = "%" + section
	}

	sylog.Debugf("Searching for section: %s\n", searchSection)

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		if headerFound {
			nextWord := strings.Fields(scanner.Text())
			for s := range parser.ValidSections {
				if len(nextWord) != 0 && nextWord[0] == "%"+s {
					end = true
					break
				}
			}
			for s := range parser.AppSections {
				if len(nextWord) != 0 && nextWord[0] == "%"+s {
					end = true
					break
				}
			}
			if !end {
				ret += scanner.Text()
				ret += "\n"
			}
		}
		if scanner.Text() == searchSection {
			headerFound = true
		}
		if end {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		sylog.Fatalf("error from scanner: %s\n", err)
	}

	return ret
}

// InspectCmd represents the 'inspect' command
// TODO: This should be in its own package, not cli
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

		if !runscript && !helpfile && !deffile && !testfile && !environment && !listApps {
			labels = true
		}

		f, err := os.Stat(args[0])
		if err != nil {
			sylog.Fatalf("Unable to stat file: %s", err)
		}
		if f.IsDir() {
			inspectInContainer(args[0])
			return
		}

		fimg, err := sif.LoadContainer(args[0], true)
		if err != nil {
			sylog.Fatalf("failed to load SIF container file: %s", err)
		}
		defer fimg.UnloadContainer()

		var inspectData inspectFormat
		inspectData.Type = "container"
		inspectData.Data.Attributes.Labels = make(map[string]string, 1)

		// Inspect Labels
		if labels {
			if AppName != "" {
				sifData, _, err := getMetaData(&fimg, sif.DataDeffile)
				if err != nil {
					sylog.Fatalf("Unable to get metadata: %s", err)
				}

				for _, v := range sifData {
					metaData := v.GetData(&fimg)
					data := parseDeffile(string(metaData), AppName, "labels")
					inspectData.Data.Attributes.AppLabels = data
				}
			} else {
				sifData, _, err := getMetaData(&fimg, sif.DataLabels)
				if err == ErrNoMetaData {
					sylog.Warningf("No metadata partition, searching in container...")
					inspectInContainer(args[0])
				} else if err != nil {
					sylog.Fatalf("Unable to get label metadata: %s", err)
				}

				for _, v := range sifData {
					metaData := v.GetData(&fimg)
					var hrOut map[string]*json.RawMessage
					err := json.Unmarshal(metaData, &hrOut)
					if err != nil {
						sylog.Fatalf("Unable to get json: %s", err)
					}

					// Sort the labels
					var labelSort []string
					for k := range hrOut {
						labelSort = append(labelSort, k)
					}
					sort.Strings(labelSort)

					for _, k := range labelSort {
						inspectData.Data.Attributes.Labels[k] = string(*hrOut[k])
					}
				}
			}
		}

		// Inspect Deffile
		if deffile {
			sifData, _, err := getMetaData(&fimg, sif.DataDeffile)
			if err != nil {
				sylog.Fatalf("Unable to get metadata: %s", err)
			}
			for _, v := range sifData {
				metaData := v.GetData(&fimg)
				data := string(metaData)
				inspectData.Data.Attributes.Deffile = data
			}
		}

		// For %help
		if helpfile {
			sifData, _, err := getMetaData(&fimg, sif.DataDeffile)
			if err != nil {
				sylog.Fatalf("Unable to get metadata: %s", err)
			}

			for _, v := range sifData {
				metaData := v.GetData(&fimg)
				data := parseDeffile(string(metaData), AppName, "help")
				inspectData.Data.Attributes.Helpfile = data
			}
		}

		// For %runscript
		if runscript {
			sifData, _, err := getMetaData(&fimg, sif.DataDeffile)
			if err != nil {
				sylog.Fatalf("Unable to get metadata: %s", err)
			}
			for _, v := range sifData {
				metaData := v.GetData(&fimg)
				data := parseDeffile(string(metaData), AppName, "runscript")
				inspectData.Data.Attributes.Runscript = data
			}
		}

		// For %test
		if testfile {
			sifData, _, err := getMetaData(&fimg, sif.DataDeffile)
			if err != nil {
				sylog.Fatalf("Unable to get metadata: %s", err)
			}
			for _, v := range sifData {
				metaData := v.GetData(&fimg)
				data := parseDeffile(string(metaData), AppName, "test")
				inspectData.Data.Attributes.Test = data
			}
		}

		// For %environment
		if environment {
			sifData, _, err := getMetaData(&fimg, sif.DataDeffile)
			if err != nil {
				sylog.Fatalf("Unable to get metadata: %s", err)
			}
			for _, v := range sifData {
				metaData := v.GetData(&fimg)
				data := parseDeffile(string(metaData), AppName, "environment")
				inspectData.Data.Attributes.Environment = data
			}
		}

		if jsonfmt {
			text, err := json.MarshalIndent(inspectData, "", "\t")
			if err != nil {
				sylog.Fatalf("Unable to marshal json: %s", err)
			}
			fmt.Printf("%s\n", string(text))
		} else {
			if inspectData.Data.Attributes.Apps != "" {
				fmt.Printf("== apps ==\n%s\n", inspectData.Data.Attributes.Apps)
			}
			// TODO: fix the app labels (maybe in the main label partition)
			if inspectData.Data.Attributes.AppLabels != "" {
				fmt.Printf("== app labels ==\n%s\n", inspectData.Data.Attributes.AppLabels)
			}
			if inspectData.Data.Attributes.Helpfile != "" {
				fmt.Printf("== helpfile ==\n%s\n", inspectData.Data.Attributes.Helpfile)
			}
			if inspectData.Data.Attributes.Deffile != "" {
				fmt.Printf("== deffile ==\n%s\n", inspectData.Data.Attributes.Deffile)
			}
			if inspectData.Data.Attributes.Runscript != "" {
				fmt.Printf("== runscript ==\n%s\n", inspectData.Data.Attributes.Runscript)
			}
			if inspectData.Data.Attributes.Test != "" {
				fmt.Printf("== test ==\n%s\n", inspectData.Data.Attributes.Test)
			}
			if inspectData.Data.Attributes.Environment != "" {
				fmt.Printf("== environment ==\n%s", inspectData.Data.Attributes.Environment)
			}
			if len(inspectData.Data.Attributes.Labels) > 0 {
				fmt.Printf("== labels ==\n")
				for labLabel, labValue := range inspectData.Data.Attributes.Labels {
					fmt.Printf("%s: %s\n", labLabel, labValue)
				}
			}
		}
	},
	TraverseChildren: true,
}

func inspectInContainer(path string) {
	abspath, err := filepath.Abs(path)
	if err != nil {
		sylog.Fatalf("While determining absolute file path: %v", err)
	}
	name := filepath.Base(abspath)

	a := []string{"/bin/sh", "-c", ""}

	if listApps {
		sylog.Debugf("Listing all apps in container")
		a[2] += listAppsCommand
	}

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

	// Default to labels if other flags are unset, excludes --app
	if labels || defaultToLabels() {
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
	inspectObj.Data.Attributes.Labels = make(map[string]string)
	//inspectObj.Data.Attributes.Environment = make(map[string]string)

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
		if inspectObj.Data.Attributes.Apps != "" {
			fmt.Printf("==apps==\n")
			fmt.Printf("%s\n", inspectObj.Data.Attributes.Apps)
		}
		if inspectObj.Data.Attributes.Helpfile != "" {
			fmt.Println("==helpfile==\n" + inspectObj.Data.Attributes.Helpfile)
		}
		if inspectObj.Data.Attributes.Deffile != "" {
			fmt.Println("==deffile==\n" + inspectObj.Data.Attributes.Deffile)
		}
		if inspectObj.Data.Attributes.Runscript != "" {
			fmt.Println("==runscript==\n" + inspectObj.Data.Attributes.Runscript)
		}
		if inspectObj.Data.Attributes.Test != "" {
			fmt.Println("==test==\n" + inspectObj.Data.Attributes.Test)
		}
		if len(inspectObj.Data.Attributes.Environment) > 0 {
			fmt.Printf("==environment==\n%s", inspectObj.Data.Attributes.Environment)
			//			for envLabel, envValue := range inspectObj.Data.Attributes.Environment {
			//				fmt.Println("==environment:" + envLabel + "==\n" + envValue)
			//			}
		}
		if len(inspectObj.Data.Attributes.Labels) > 0 {
			fmt.Println("==labels==")
			for labLabel, labValue := range inspectObj.Data.Attributes.Labels {
				fmt.Println(labLabel + ": " + labValue)
			}
		}
	}

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

	// Record from stdout and store as a string to return as the contents of the file

	cmd, err := exec.PipeCommand(starter, []string{procname}, Env, configData)
	if err != nil {
		sylog.Fatalf("Unable to exec command: %s: %s", err, cmd.Args)
	}

	b, err := cmd.Output()
	if err != nil {
		sylog.Fatalf("Unable to process command: %s: %s", err, b)
	}

	return string(b), nil
}
