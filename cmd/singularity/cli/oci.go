// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/opencontainers/runtime-tools/generate"
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/instance"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/oci"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/exec"
	"github.com/sylabs/singularity/internal/pkg/util/signal"
	"github.com/sylabs/singularity/internal/pkg/util/unix"
	"golang.org/x/crypto/ssh/terminal"
)

var bundlePath string
var syncSocketPath string
var emptyProcess bool

func init() {
	SingularityCmd.AddCommand(OciCmd)

	OciCreateCmd.Flags().SetInterspersed(false)
	OciCreateCmd.Flags().StringVarP(&bundlePath, "bundle", "b", "", "specify the OCI bundle path")
	OciCreateCmd.Flags().SetAnnotation("bundle", "argtag", []string{"<path>"})
	OciCreateCmd.Flags().StringVarP(&syncSocketPath, "sync-socket", "s", "", "specify the path to unix socket for state synchronization (internal)")
	OciCreateCmd.Flags().SetAnnotation("sync-socket", "argtag", []string{"<path>"})
	OciCreateCmd.Flags().BoolVar(&emptyProcess, "empty-process", false, "run container without executing container process (eg: for POD container)")

	OciStartCmd.Flags().SetInterspersed(false)
	OciDeleteCmd.Flags().SetInterspersed(false)
	OciAttachCmd.Flags().SetInterspersed(false)

	OciStateCmd.Flags().SetInterspersed(false)
	OciStateCmd.Flags().StringVarP(&syncSocketPath, "sync-socket", "s", "", "specify the path to unix socket for state synchronization (internal)")
	OciStateCmd.Flags().SetAnnotation("sync-socket", "argtag", []string{"<path>"})

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
	OciCmd.AddCommand(OciAttachCmd)
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

// OciAttachCmd represents oci start command
var OciAttachCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if err := ociAttach(args[0]); err != nil {
			sylog.Fatalf("%s", err)
		}
	},
	Use:     "attach",
	Short:   "oci attach",
	Long:    "oci attach",
	Example: "oci attach",
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

func attach(socket string) error {
	channel := os.Stdout

	f, err := net.Dial("unix", socket)
	if err != nil {
		return err
	}
	defer f.Close()

	ostate, _ := terminal.MakeRaw(0)

	var once sync.Once
	var wg sync.WaitGroup

	wg.Add(1)

	close := func() {
		terminal.Restore(0, ostate)
		wg.Done()
	}

	// Pipe session to bash and visa-versa
	go func() {
		io.Copy(channel, f)
		once.Do(close)
	}()

	go func() {
		io.Copy(f, channel)
	}()

	wg.Wait()

	return nil
}

func exitContainer(containerID string, syncSocketPath string) {
	state, err := getState(containerID)
	if err != nil {
		sylog.Errorf("%s", err)
		os.Exit(1)
	}

	if _, ok := state.Annotations["io.sylabs.runtime.oci.exit-code"]; ok {
		code := state.Annotations["io.sylabs.runtime.oci.exit-code"]
		exitCode, err := strconv.Atoi(code)
		if err != nil {
			sylog.Errorf("%s", err)
			defer os.Exit(1)
		} else {
			defer os.Exit(exitCode)
		}
	}

	if syncSocketPath != "" {
		if err := ociDelete(containerID); err != nil {
			sylog.Errorf("%s", err)
		}
		os.Remove(syncSocketPath)
	}
}

func ociRun(containerID string) error {
	syncSocketPath = filepath.Join("/tmp", containerID+".sock")

	defer os.Remove(syncSocketPath)

	l, err := net.Listen("unix", syncSocketPath)
	if err != nil {
		return err
	}
	defer l.Close()

	defer exitContainer(containerID, syncSocketPath)

	if err := ociCreate(containerID); err != nil {
		return err
	}

	start := make(chan string, 1)

	go func() {
		var state specs.State

		for {
			c, err := l.Accept()
			if err != nil {
				return
			}

			dec := json.NewDecoder(c)
			if err := dec.Decode(&state); err != nil {
				return
			}

			c.Close()

			switch state.Status {
			case "created":
				if err := ociStart(containerID); err != nil {
					return
				}
			case "running":
				start <- state.Annotations["io.sylabs.runtime.oci.attach-socket"]
			case "stopped":
				return
			}
		}
	}()

	socket := <-start

	return attach(socket)
}

func ociAttach(containerID string) error {
	state, err := getState(containerID)
	if err != nil {
		return err
	}

	socket, ok := state.Annotations["io.sylabs.runtime.oci.attach-socket"]
	if !ok {
		return fmt.Errorf("attach socket not available, container state: %s", state.Status)
	}

	defer exitContainer(containerID, "")

	return attach(socket)
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
	if syncSocketPath != "" {
		data, err := json.Marshal(state)
		if err != nil {
			return fmt.Errorf("failed to marshal state data: %s", err)
		} else if err := unix.WriteSocket(syncSocketPath, data); err != nil {
			return err
		}
	} else {
		c, err := json.MarshalIndent(state, "", "\t")
		if err != nil {
			return err
		}
		fmt.Println(string(c))
	}
	return nil
}

func ociCreate(containerID string) error {
	starter := buildcfg.LIBEXECDIR + "/singularity/bin/starter"

	_, err := getState(containerID)
	if err == nil {
		return fmt.Errorf("%s already exists", containerID)
	}

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

	engineConfig.EmptyProcess = emptyProcess
	engineConfig.SyncSocket = syncSocketPath

	commonConfig := &config.Common{
		ContainerID:  containerID,
		EngineName:   "oci",
		EngineConfig: engineConfig,
	}

	configData, err := json.Marshal(commonConfig)
	if err != nil {
		sylog.Fatalf("%s", err)
	}

	procName := fmt.Sprintf("Singularity OCI %s", containerID)
	cmd, err := exec.PipeCommand(starter, []string{procName}, Env, configData)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
