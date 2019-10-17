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
	"strconv"

	"github.com/spf13/cobra"
	"github.com/sylabs/sif/pkg/sif"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

func init() {
	cmdManager.RegisterCmd(InspectCmd)

	cmdManager.RegisterFlagForCmd(&inspectDeffileFlag, InspectCmd)
	cmdManager.RegisterFlagForCmd(&inspectJSONFlag, InspectCmd)
	cmdManager.RegisterFlagForCmd(&inspectLabelsFlag, InspectCmd)
}

// InspectCmd represents the 'inspect' command.
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
			sylog.Fatalf("Cannot inspect a sandbox container")
		}

		fimg, err := sif.LoadContainer(args[0], true)
		if err != nil {
			sylog.Fatalf("Failed to load SIF container file: %s", err)
		}
		defer fimg.UnloadContainer()

		var inspectData inspectFormat
		inspectData.Type = containerType
		inspectData.Data.Attributes.Labels = make(map[string]string, 1)

		// Inspect Labels.
		if labels || !deffile {
			labelDescriptor, _, err := fimg.GetLinkedDescrsByType(uint32(0), sif.DataLabels)
			if err != nil {
				sylog.Fatalf("No metadata partition")
			}

			for _, v := range labelDescriptor {
				metaData := v.GetData(&fimg)
				var hrOut map[string]json.RawMessage
				err = json.Unmarshal(metaData, &hrOut)
				if err != nil {
					sylog.Fatalf("Unable to get json: %s", err)
				}

				for k, v := range hrOut {
					value := string(v)
					// Only remove the extra quotes if json output.
					if jsonfmt {
						var err error
						value, err = strconv.Unquote(value)
						if err != nil {
							sylog.Fatalf("Unable to remove quotes from data: %s\n", err)
						}
					}
					inspectData.Data.Attributes.Labels[k] = value
				}
			}
		}

		// Inspect Deffile.
		if deffile {
			labelDescriptor, _, err := fimg.GetLinkedDescrsByType(uint32(0), sif.DataDeffile)
			if err != nil {
				sylog.Fatalf("No metadata partition")
			}

			for _, v := range labelDescriptor {
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
