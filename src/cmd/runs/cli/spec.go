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
