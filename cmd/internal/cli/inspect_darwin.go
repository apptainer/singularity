// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/buger/jsonparser"
	"github.com/spf13/cobra"
	"github.com/sylabs/sif/pkg/sif"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/build/metadata"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/cmdline"
)

var (
	labels  bool
	deffile bool
	jsonfmt bool
)

type inspectMetadata struct {
	// TODO: Its possible to get this info through the label partition
	Apps      string `json:"apps,omitempty"`
	AppLabels string `json:"apps-labels,omitempty"`

	Labels  map[string]string `json:"labels,omitempty"`
	Deffile string            `json:"deffile,omitempty"`
}

type inspectAttributesData struct {
	Attributes inspectMetadata `json:"attributes"`
}

type inspectFormat struct {
	Data inspectAttributesData `json:"data"`
	Type string                `json:"type"`
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

	cmdManager.RegisterFlagForCmd(&inspectDeffileFlag, InspectCmd)
	cmdManager.RegisterFlagForCmd(&inspectJSONFlag, InspectCmd)
	cmdManager.RegisterFlagForCmd(&inspectLabelsFlag, InspectCmd)
}

// InspectCmd represents the 'inspect' command
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
		if f.IsDir() {
			sylog.Fatalf("not yet...")
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
		if labels || !deffile {
			jsonName := ""
			if AppName == "" {
				jsonName = "system-partition"
			} else {
				jsonName = AppName
			}

			sifData, err := metadata.GetSIFData(&fimg, sif.DataLabels)
			if err == metadata.ErrNoMetaData {
				sylog.Fatalf("No metadata partition")
			} else if err != nil {
				sylog.Fatalf("Unable to get label metadata: %s", err)
			}

			for _, v := range sifData {
				metaData := v.GetData(&fimg)
				newbytes, _, _, err := jsonparser.Get(metaData, jsonName)
				if err != nil {
					sylog.Fatalf("Unable to find json from metadata: %s", err)
				}
				var hrOut map[string]*json.RawMessage
				err = json.Unmarshal(newbytes, &hrOut)
				if err != nil {
					sylog.Fatalf("Unable to get json: %s", err)
				}

				for k, v := range hrOut {
					inspectData.Data.Attributes.Labels[k] = string(*v)
				}
			}
		}

		// Inspect Deffile
		if deffile {
			sifData, err := metadata.GetSIFData(&fimg, sif.DataDeffile)
			if err == metadata.ErrNoMetaData {
				sylog.Fatalf("No metadata partition")
			} else if err != nil {
				sylog.Fatalf("Unable to get metadata: %s", err)
			}

			for _, v := range sifData {
				metaData := v.GetData(&fimg)
				data := string(metaData)
				inspectData.Data.Attributes.Deffile = data
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
			if inspectData.Data.Attributes.Deffile != "" {
				fmt.Printf("%s\n", inspectData.Data.Attributes.Deffile)
			}
			if len(inspectData.Data.Attributes.Labels) > 0 {
				// Sort the labels
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
