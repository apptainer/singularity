// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/opencontainers/runtime-tools/generate"
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/src/pkg/buildcfg"
	"github.com/sylabs/singularity/src/pkg/instance"
	"github.com/sylabs/singularity/src/pkg/sylog"
	"github.com/sylabs/singularity/src/pkg/util/exec"
	"github.com/sylabs/singularity/src/pkg/util/signal"
	"github.com/sylabs/singularity/src/runtime/engines/config"
	"github.com/sylabs/singularity/src/runtime/engines/oci"
)

var bundlePath = ""

func init() {
	SingularityCmd.AddCommand(OciCmd)

	OciCreateCmd.Flags().SetInterspersed(false)
	OciCreateCmd.Flags().StringVarP(&bundlePath, "bundle", "b", "", "specify the OCI bundle path")
	OciCreateCmd.Flags().SetAnnotation("bundle", "argtag", []string{"<path>"})

	OciStartCmd.Flags().SetInterspersed(false)
	OciDeleteCmd.Flags().SetInterspersed(false)
	OciStateCmd.Flags().SetInterspersed(false)

	OciKillCmd.Flags().SetInterspersed(false)
	OciKillCmd.Flags().StringVarP(&stopSignal, "signal", "s", "", "signal sent to the container (default SIGTERM)")

	OciRunCmd.Flags().SetInterspersed(false)
	OciRunCmd.Flags().StringVarP(&bundlePath, "bundle", "b", "", "specify the OCI bundle path")
	OciRunCmd.Flags().SetAnnotation("bundle", "argtag", []string{"<path>"})

	OciCmd.AddCommand(OciStartCmd)
	OciCmd.AddCommand(OciCreateCmd)
	OciCmd.AddCommand(OciRunCmd)
	OciCmd.AddCommand(OciDeleteCmd)
	OciCmd.AddCommand(OciKillCmd)
	OciCmd.AddCommand(OciStateCmd)
}

// OciCreateCmd represents oci create command
var OciCreateCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if err := ociCreate(args[0]); err != nil {
			sylog.Fatalf("%s", err)
		}
	},
	Use:     "create",
	Short:   "oci create",
	Long:    "oci create",
	Example: "oci create",
}

// OciRunCmd allow to create/start in row
var OciRunCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if err := ociRun(args[0]); err != nil {
			sylog.Fatalf("%s", err)
		}
	},
	Use:     "run",
	Short:   "oci run",
	Long:    "oci run",
	Example: "oci run",
}

// OciStartCmd represents oci start command
var OciStartCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if err := ociStart(args[0]); err != nil {
			sylog.Fatalf("%s", err)
		}
	},
	Use:     "start",
	Short:   "oci start",
	Long:    "oci start",
	Example: "oci start",
}

// OciDeleteCmd represents oci start command
var OciDeleteCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if err := ociDelete(args[0]); err != nil {
			sylog.Fatalf("%s", err)
		}
	},
	Use:     "delete",
	Short:   "oci delete",
	Long:    "oci delete",
	Example: "oci delete",
}

// OciKillCmd represents oci start command
var OciKillCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if err := ociKill(args[0]); err != nil {
			sylog.Fatalf("%s", err)
		}
	},
	Use:     "kill",
	Short:   "oci kill",
	Long:    "oci kill",
	Example: "oci kill",
}

// OciStateCmd represents oci start command
var OciStateCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if err := ociState(args[0]); err != nil {
			sylog.Fatalf("%s", err)
		}
	},
	Use:     "state",
	Short:   "oci state",
	Long:    "oci state",
	Example: "oci state",
}

// OciCmd singularity oci runtime
var OciCmd = &cobra.Command{
	Run:                   nil,
	DisableFlagsInUseLine: true,

	Use:     "oci",
	Short:   "oci",
	Long:    "oci",
	Example: "oci",
}

func getState(containerID string) (*specs.State, error) {
	commonConfig := config.Common{
		EngineConfig: &oci.EngineConfig{},
	}

	file, err := instance.Get(containerID)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(file.Config, &commonConfig); err != nil {
		return nil, err
	}

	engineConfig := commonConfig.EngineConfig.(*oci.EngineConfig)

	return &engineConfig.State, nil
}

func ociRun(containerID string) error {
	if err := ociCreate(containerID); err != nil {
		return err
	}
	return ociStart(containerID)
}

func ociStart(containerID string) error {
	// send SIGCONT signal to the instance
	state, err := getState(containerID)
	if err != nil {
		return err
	}
	if err := syscall.Kill(state.Pid, syscall.SIGCONT); err != nil {
		return err
	}
	return nil
}

func ociKill(containerID string) error {
	// send signal to the instance
	state, err := getState(containerID)
	if err != nil {
		return err
	}

	sig := syscall.SIGTERM

	if stopSignal != "" {
		sig, err = signal.Convert(stopSignal)
		if err != nil {
			return err
		}
	}

	return syscall.Kill(state.Pid, sig)
}

func ociDelete(containerID string) error {
	// remove instance files
	file, err := instance.Get(containerID)
	if err != nil {
		return err
	}
	return file.Delete()
}

func ociState(containerID string) error {
	// query instance files and returns state
	state, err := getState(containerID)
	if err != nil {
		return err
	}
	c, err := json.MarshalIndent(state, "", "\t")
	if err != nil {
		return err
	}
	fmt.Println(string(c))
	return nil
}

func ociCreate(containerID string) error {
	starter := buildcfg.LIBEXECDIR + "/singularity/bin/starter"

	os.Clearenv()

	engineConfig := oci.NewConfig()
	generator := generate.Generator{Config: &engineConfig.OciConfig.Spec}
	engineConfig.SetBundlePath(bundlePath)

	// load config.json from bundle path
	configJSON := filepath.Join(bundlePath, "config.json")
	fb, err := os.Open(configJSON)
	if err != nil {
		return fmt.Errorf("failed to open %s: %s", configJSON, err)
	}

	data, err := ioutil.ReadAll(fb)
	if err != nil {
		return fmt.Errorf("failed to read %s: %s", configJSON, err)
	}

	fb.Close()

	if err := json.Unmarshal(data, generator.Config); err != nil {
		return fmt.Errorf("failed to parse %s: %s", configJSON, err)
	}

	Env := []string{sylog.GetEnvVar(), "SRUNTIME=oci"}

	commonConfig := &config.Common{
		ContainerID:  containerID,
		EngineName:   "oci",
		EngineConfig: engineConfig,
	}

	configData, err := json.Marshal(commonConfig)
	if err != nil {
		sylog.Fatalf("%s", err)
	}

	cmd, err := exec.PipeCommand(starter, []string{"OCI"}, Env, configData)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}
