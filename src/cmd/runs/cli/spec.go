// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/singularityware/singularity/src/docs"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/sylabs/sif/pkg/sif"

	"github.com/opencontainers/runtime-tools/generate"
	"github.com/spf13/cobra"
)

const (
	ConfigSpec = "config.json"
)

func init() {
	SpecCmd.Flags().SetInterspersed(false)

	cwd, err := os.Getwd()
	if err != nil {
		sylog.Fatalf("%v", err)
	}

	SpecGenCmd.Flags().StringVarP(&bundlePath, "bundle", "b", cwd, "path to singularity image file (SIF), default to current directory")
	ExecRunsCmd.AddCommand(SpecCmd)
	SpecCmd.AddCommand(SpecGenCmd)
	SpecCmd.AddCommand(SpecAddCmd)
	SpecCmd.AddCommand(SpecInspectCmd)
}

// SpecCmd runs spec cmd
var SpecCmd = &cobra.Command{
	Args: cobra.MinimumNArgs(1),
	Run:  nil,
	DisableFlagsInUseLine: true,

	Use:   docs.RunsSpecUse,
	Short: docs.RunsSpecShort,
	Long:  docs.RunsSpecLong,
}

// SpecInspectCmd prints the OCi runtime spec stored in the SIF bundle
var SpecInspectCmd = &cobra.Command{
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sifPath := args[0]

		// load the SIF (singularity image file)
		fimg, err := sif.LoadContainer(sifPath, false)
		if err != nil {
			sylog.Fatalf("Error loading SIF %s:\t%s", sifPath, err)
		}

		// lookup of a descriptor of type DataGenericJSON
		descr := sif.Descriptor{
			Datatype: sif.DataGenericJSON,
		}
		d, match, _ := fimg.GetFromDescr(descr)
		if err != nil {
			sylog.Fatalf("%s", err)
		}
		if match != 1 && d.GetName() != ConfigSpec {
			sylog.Infof("SIF bundle doesn't contains a OCI runtime spec")
			return
		}

		// if found, print OCI runtime spec into stdout
		// TODO: format flag, or to file
		if _, err := fimg.Fp.Seek(d.Fileoff, 0); err != nil {
			sylog.Errorf("while seeking to data object: %s", err)
			return
		}
		if _, err := io.CopyN(os.Stdout, fimg.Fp, d.Filelen); err != nil {
			sylog.Errorf("while copying data object to stdout: %s", err)
			return
		}

		// unload the SIF container
		if err = fimg.UnloadContainer(); err != nil {
			sylog.Fatalf("UnloadContainer(fimg):\t%s", err)
		}

	},
	DisableFlagsInUseLine: true,

	Use:   docs.RunsSpecInspectUse,
	Short: docs.RunsSpecInspectShort,
	Long:  docs.RunsSpecInspectLong,
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

		// lookup of a descriptor of type DataGenericJSON
		descr := sif.Descriptor{
			Datatype: sif.DataGenericJSON,
		}
		d, match, _ := fimg.GetFromDescr(descr)
		if err != nil {
			sylog.Fatalf("%s", err)
		}
		if match == 1 && d.GetName() == ConfigSpec {
			sylog.Fatalf("SIF bundle already contains a config.json")
		}

		// add new data object 'configInput' to SIF file
		if err = fimg.AddObject(configInput); err != nil {
			sylog.Fatalf("fimg.AddObject():\t%s", err)
		}

		// unload the SIF container
		if err = fimg.UnloadContainer(); err != nil {
			sylog.Fatalf("UnloadContainer(fimg):\t%s", err)
		}

	},
	DisableFlagsInUseLine: true,

	Use:   docs.RunsSpecAddUse,
	Short: docs.RunsSpecAddShort,
	Long:  docs.RunsSpecAddLong,
}

// SpecGenCmd creates a config.json in the cwd, with Linux as the default OS
var SpecGenCmd = &cobra.Command{
	Run: func(cmd *cobra.Command, args []string) {
		// Check for existing config.json on CWD
		overwrite, err := configFileExist(ConfigSpec)
		if err != nil {
			sylog.Fatalf("%s", err)
		}
		if !overwrite {
			return
		} else {
			sylog.Infof("Overwriting existing config.json...")
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

		err = ioutil.WriteFile(ConfigSpec, config, 0644)
		if err != nil {
			sylog.Fatalf("couldn't write config file: %s", err)
		}
	},
	DisableFlagsInUseLine: true,

	Use:   docs.RunsSpecGenUse,
	Short: docs.RunsSpecGenShort,
	Long:  docs.RunsSpecGenLong,
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
		} else {
			return false, nil
		}

	}

	if !os.IsNotExist(err) {
		return false, err
	}

	return true, nil
}
