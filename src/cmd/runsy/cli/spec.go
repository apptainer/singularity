// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/singularityware/singularity/src/docs"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/oci"
	"github.com/sylabs/sif/pkg/sif"

	"github.com/opencontainers/runtime-tools/generate"
	"github.com/spf13/cobra"
)

func init() {
	SpecCmd.Flags().SetInterspersed(false)

	cwd, err := os.Getwd()
	if err != nil {
		sylog.Fatalf("%v", err)
	}

	SpecGenCmd.Flags().StringVarP(&bundlePath, "bundle", "b", cwd, "path to singularity image file (SIF), default to current directory")
	RunsyCmd.AddCommand(SpecCmd)
	SpecCmd.AddCommand(SpecGenCmd)
	SpecCmd.AddCommand(SpecAddCmd)
	SpecCmd.AddCommand(SpecInspectCmd)
}

// SpecCmd runs spec cmd
var SpecCmd = &cobra.Command{
	Args: cobra.MinimumNArgs(1),
	Run:  nil,
	DisableFlagsInUseLine: true,

	Use:   docs.RunsySpecUse,
	Short: docs.RunsySpecShort,
	Long:  docs.RunsySpecLong,
}

// SpecInspectCmd check if the target SIF has a OCI runtime spec
// if found will print it into stdout
// TODO: flag for print to file
var SpecInspectCmd = &cobra.Command{
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sifPath := args[0]

		spec, err := oci.LoadConfigSpec(sifPath)
		if err != nil {
			sylog.Fatalf("%v", err)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.Encode(spec)
	},
	DisableFlagsInUseLine: true,

	Use:   docs.RunsySpecInspectUse,
	Short: docs.RunsySpecInspectShort,
	Long:  docs.RunsySpecInspectLong,
}

// SpecAddCmd adds a given config.json to a  target SIF
var SpecAddCmd = &cobra.Command{
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		specPath := args[0]
		sifPath := args[1]

		// create sif descriptor input for DataGenericJSON
		configInput := sif.DescriptorInput{
			Datatype: sif.DataGenericJSON,
			Groupid:  sif.DescrDefaultGroup,
			Link:     sif.DescrUnusedLink,
			Fname:    specPath,
		}

		// open up the JSON data object file for this descriptor
		if configInput.Fp, err = os.Open(configInput.Fname); err != nil {
			sylog.Fatalf("read data object file:\t%s", err)
		}
		defer configInput.Fp.Close()

		fi, err := configInput.Fp.Stat()
		if err != nil {
			sylog.Fatalf("can't stat partition file:\t%s", err)
		}
		configInput.Size = fi.Size()

		// load the SIF (singularity image file)
		fimg, err := sif.LoadContainer(sifPath, false)
		if err != nil {
			sylog.Fatalf("Error loading SIF %s:\t%s", sifPath, err)
		}
		defer fimg.UnloadContainer()

		// lookup of a descriptor of type DataGenericJSON
		descr := sif.Descriptor{
			Datatype: sif.DataGenericJSON,
		}
		d, match, _ := fimg.GetFromDescr(descr)
		if err != nil {
			sylog.Fatalf("%s", err)
		}
		if match == 1 && d.GetName() == oci.ConfigSpec {
			sylog.Fatalf("SIF bundle already contains a config.json")
		}

		// add new data object 'configInput' to SIF file
		if err = fimg.AddObject(configInput); err != nil {
			sylog.Fatalf("fimg.AddObject():\t%s", err)
		}

	},
	DisableFlagsInUseLine: true,

	Use:   docs.RunsySpecAddUse,
	Short: docs.RunsySpecAddShort,
	Long:  docs.RunsySpecAddLong,
}

// SpecGenCmd creates a config.json in the cwd, with Linux as the default OS
var SpecGenCmd = &cobra.Command{
	Run: func(cmd *cobra.Command, args []string) {
		// Check for existing config.json on CWD
		overwrite, err := configFileExist(oci.ConfigSpec)
		if err != nil {
			sylog.Fatalf("%s", err)
		}
		if !overwrite {
			return
		}

		specgen, err := generate.New("linux")
		if err != nil {
			sylog.Fatalf("%s", err)
		}

		spec := specgen.Spec()
		config, err := json.Marshal(spec)
		if err != nil {
			sylog.Fatalf("unable to marshal oci runtime spec: %s", err)
		}

		err = ioutil.WriteFile(oci.ConfigSpec, config, 0644)
		if err != nil {
			sylog.Fatalf("couldn't write config file: %s", err)
		}
	},
	DisableFlagsInUseLine: true,

	Use:   docs.RunsySpecGenUse,
	Short: docs.RunsySpecGenShort,
	Long:  docs.RunsySpecGenLong,
}

func configFileExist(name string) (bool, error) {
	cwd, err := os.Getwd()
	if err != nil {
		sylog.Fatalf("%s", err)
	}

	_, err = os.Stat(fmt.Sprintf("%s/%s", cwd, name))
	if err == nil {
		fmt.Printf("File %s exists: do you want to overwrite it? (y/n)\n", name)
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		resp := scanner.Text()
		if err := scanner.Err(); err != nil {
			sylog.Errorf("error while reading response from user: %s\n", err)
			return false, err
		}

		if resp == "y" {
			return true, nil
		}
		return false, nil

	}

	if !os.IsNotExist(err) {
		return false, err
	}

	return true, nil
}
