// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build linux

package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	osignal "os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/sylabs/singularity/internal/pkg/cgroups"

	"github.com/kr/pty"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/opencontainers/runtime-tools/generate"
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/instance"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/oci"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/exec"
	"github.com/sylabs/singularity/internal/pkg/util/signal"
	"github.com/sylabs/singularity/pkg/ociruntime"
	"github.com/sylabs/singularity/pkg/util/unix"
	"golang.org/x/crypto/ssh/terminal"
)

var bundlePath string
var logPath string
var logFormat string
var syncSocketPath string
var emptyProcess bool
var pidFile string
var fromFile string
var killSignal string

func init() {
	SingularityCmd.AddCommand(OciCmd)

	OciCreateCmd.Flags().SetInterspersed(false)
	OciCreateCmd.Flags().StringVarP(&bundlePath, "bundle", "b", "", "specify the OCI bundle path")
	OciCreateCmd.Flags().SetAnnotation("bundle", "argtag", []string{"<path>"})
	OciCreateCmd.Flags().StringVarP(&syncSocketPath, "sync-socket", "s", "", "specify the path to unix socket for state synchronization (internal)")
	OciCreateCmd.Flags().SetAnnotation("sync-socket", "argtag", []string{"<path>"})
	OciCreateCmd.Flags().BoolVar(&emptyProcess, "empty-process", false, "run container without executing container process (eg: for POD container)")
	OciCreateCmd.Flags().StringVarP(&logPath, "log-path", "l", "", "specify the log file path")
	OciCreateCmd.Flags().SetAnnotation("log-path", "argtag", []string{"<path>"})
	OciCreateCmd.Flags().StringVar(&logFormat, "log-format", "kubernetes", "specify the log file format")
	OciCreateCmd.Flags().SetAnnotation("log-format", "argtag", []string{"<format>"})
	OciCreateCmd.Flags().StringVar(&pidFile, "pid-file", "", "specify the pid file")
	OciCreateCmd.Flags().SetAnnotation("pid-file", "argtag", []string{"<path>"})

	OciStartCmd.Flags().SetInterspersed(false)
	OciDeleteCmd.Flags().SetInterspersed(false)
	OciAttachCmd.Flags().SetInterspersed(false)
	OciExecCmd.Flags().SetInterspersed(false)
	OciPauseCmd.Flags().SetInterspersed(false)
	OciResumeCmd.Flags().SetInterspersed(false)

	OciStateCmd.Flags().SetInterspersed(false)
	OciStateCmd.Flags().StringVarP(&syncSocketPath, "sync-socket", "s", "", "specify the path to unix socket for state synchronization (internal)")
	OciStateCmd.Flags().SetAnnotation("sync-socket", "argtag", []string{"<path>"})

	OciKillCmd.Flags().SetInterspersed(false)
	OciKillCmd.Flags().StringVarP(&killSignal, "signal", "s", "", "signal sent to the container (default SIGTERM)")

	OciRunCmd.Flags().SetInterspersed(false)
	OciRunCmd.Flags().StringVarP(&bundlePath, "bundle", "b", "", "specify the OCI bundle path")
	OciRunCmd.Flags().SetAnnotation("bundle", "argtag", []string{"<path>"})
	OciRunCmd.Flags().StringVarP(&logPath, "log-path", "l", "", "specify the log file path")
	OciRunCmd.Flags().SetAnnotation("log-path", "argtag", []string{"<path>"})
	OciRunCmd.Flags().StringVar(&logFormat, "log-format", "kubernetes", "specify the log file format")
	OciRunCmd.Flags().SetAnnotation("log-format", "argtag", []string{"<format>"})
	OciRunCmd.Flags().StringVar(&pidFile, "pid-file", "", "specify the pid file")
	OciRunCmd.Flags().SetAnnotation("pid-file", "argtag", []string{"<path>"})

	OciUpdateCmd.Flags().SetInterspersed(false)
	OciUpdateCmd.Flags().StringVarP(&fromFile, "from-file", "f", "", "specify path to OCI JSON cgroups resource file ('-' to read from STDIN)")

	OciCmd.AddCommand(OciStartCmd)
	OciCmd.AddCommand(OciCreateCmd)
	OciCmd.AddCommand(OciRunCmd)
	OciCmd.AddCommand(OciDeleteCmd)
	OciCmd.AddCommand(OciKillCmd)
	OciCmd.AddCommand(OciStateCmd)
	OciCmd.AddCommand(OciAttachCmd)
	OciCmd.AddCommand(OciExecCmd)
	OciCmd.AddCommand(OciUpdateCmd)
	OciCmd.AddCommand(OciPauseCmd)
	OciCmd.AddCommand(OciResumeCmd)
}

// OciCreateCmd represents oci create command.
var OciCreateCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if err := ociCreate(args[0]); err != nil {
			sylog.Fatalf("%s", err)
		}
	},
	Use:     docs.OciCreateUse,
	Short:   docs.OciCreateShort,
	Long:    docs.OciCreateLong,
	Example: docs.OciCreateExample,
}

// OciRunCmd allow to create/start in row.
var OciRunCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if err := ociRun(args[0]); err != nil {
			sylog.Fatalf("%s", err)
		}
	},
	Use:     docs.OciRunUse,
	Short:   docs.OciRunShort,
	Long:    docs.OciRunLong,
	Example: docs.OciRunExample,
}

// OciStartCmd represents oci start command.
var OciStartCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if err := ociStart(args[0]); err != nil {
			sylog.Fatalf("%s", err)
		}
	},
	Use:     docs.OciStartUse,
	Short:   docs.OciStartShort,
	Long:    docs.OciStartLong,
	Example: docs.OciStartExample,
}

// OciDeleteCmd represents oci delete command.
var OciDeleteCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if err := ociDelete(args[0]); err != nil {
			sylog.Fatalf("%s", err)
		}
	},
	Use:     docs.OciDeleteUse,
	Short:   docs.OciDeleteShort,
	Long:    docs.OciDeleteLong,
	Example: docs.OciDeleteExample,
}

// OciKillCmd represents oci kill command.
var OciKillCmd = &cobra.Command{
	Args:                  cobra.MinimumNArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 1 && args[1] != "" {
			killSignal = args[1]
		}
		if err := ociKill(args[0], 0); err != nil {
			sylog.Fatalf("%s", err)
		}
	},
	Use:     docs.OciKillUse,
	Short:   docs.OciKillShort,
	Long:    docs.OciKillLong,
	Example: docs.OciKillExample,
}

// OciStateCmd represents oci state command.
var OciStateCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if err := ociState(args[0]); err != nil {
			sylog.Fatalf("%s", err)
		}
	},
	Use:     docs.OciStateUse,
	Short:   docs.OciStateShort,
	Long:    docs.OciStateLong,
	Example: docs.OciStateExample,
}

// OciAttachCmd represents oci attach command.
var OciAttachCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if err := ociAttach(args[0]); err != nil {
			sylog.Fatalf("%s", err)
		}
	},
	Use:     docs.OciAttachUse,
	Short:   docs.OciAttachShort,
	Long:    docs.OciAttachLong,
	Example: docs.OciAttachExample,
}

// OciExecCmd represents oci exec command.
var OciExecCmd = &cobra.Command{
	Args:                  cobra.MinimumNArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if err := ociExec(args[0], args[1:]); err != nil {
			sylog.Fatalf("%s", err)
		}
	},
	Use:     docs.OciExecUse,
	Short:   docs.OciExecShort,
	Long:    docs.OciExecLong,
	Example: docs.OciExecExample,
}

// OciUpdateCmd represents oci update command.
var OciUpdateCmd = &cobra.Command{
	Args:                  cobra.MinimumNArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if err := ociUpdate(args[0]); err != nil {
			sylog.Fatalf("%s", err)
		}
	},
	Use:     docs.OciUpdateUse,
	Short:   docs.OciUpdateShort,
	Long:    docs.OciUpdateLong,
	Example: docs.OciUpdateExample,
}

// OciPauseCmd represents oci pause command.
var OciPauseCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if err := ociPauseResume(args[0], true); err != nil {
			sylog.Fatalf("%s", err)
		}
	},
	Use:     docs.OciPauseUse,
	Short:   docs.OciPauseShort,
	Long:    docs.OciPauseLong,
	Example: docs.OciPauseExample,
}

// OciResumeCmd represents oci resume command.
var OciResumeCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if err := ociPauseResume(args[0], false); err != nil {
			sylog.Fatalf("%s", err)
		}
	},
	Use:     docs.OciResumeUse,
	Short:   docs.OciResumeShort,
	Long:    docs.OciResumeLong,
	Example: docs.OciResumeExample,
}

// OciCmd singularity oci runtime.
var OciCmd = &cobra.Command{
	Run:                   nil,
	DisableFlagsInUseLine: true,

	Use:     docs.OciUse,
	Short:   docs.OciShort,
	Long:    docs.OciLong,
	Example: docs.OciExample,
}

func getCommonConfig(containerID string) (*config.Common, error) {
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

	return &commonConfig, nil
}

func getEngineConfig(containerID string) (*oci.EngineConfig, error) {
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

	return commonConfig.EngineConfig.(*oci.EngineConfig), nil
}

func getState(containerID string) (*ociruntime.State, error) {
	engineConfig, err := getEngineConfig(containerID)
	if err != nil {
		return nil, err
	}
	return &engineConfig.State, nil
}

func resize(controlSocket string, oversized bool) {
	ctrl := &ociruntime.Control{}
	ctrl.ConsoleSize = &specs.Box{}

	c, err := unix.Dial(controlSocket)
	if err != nil {
		sylog.Errorf("failed to connect to control socket")
		return
	}
	defer c.Close()

	rows, cols, err := pty.Getsize(os.Stdin)
	if err != nil {
		sylog.Errorf("terminal resize error: %s", err)
		return
	}

	ctrl.ConsoleSize.Height = uint(rows)
	ctrl.ConsoleSize.Width = uint(cols)

	if oversized {
		ctrl.ConsoleSize.Height++
		ctrl.ConsoleSize.Width++
	}

	enc := json.NewEncoder(c)
	if err != nil {
		sylog.Errorf("%s", err)
		return
	}

	if err := enc.Encode(ctrl); err != nil {
		sylog.Errorf("%s", err)
		return
	}
}

func attach(engineConfig *oci.EngineConfig, run bool) error {
	var ostate *terminal.State
	var conn net.Conn
	var wg sync.WaitGroup

	state := &engineConfig.State

	if state.AttachSocket == "" {
		return fmt.Errorf("attach socket not available, container state: %s", state.Status)
	}
	if state.ControlSocket == "" {
		return fmt.Errorf("control socket not available, container state: %s", state.Status)
	}

	hasTerminal := engineConfig.OciConfig.Process.Terminal && terminal.IsTerminal(0)

	var err error
	conn, err = unix.Dial(state.AttachSocket)
	if err != nil {
		return err
	}
	defer conn.Close()

	if hasTerminal {
		ostate, _ = terminal.MakeRaw(0)
		resize(state.ControlSocket, true)
		resize(state.ControlSocket, false)
	}

	wg.Add(1)

	go func() {
		// catch SIGWINCH signal for terminal resize
		signals := make(chan os.Signal, 1)
		pid := state.Pid
		osignal.Notify(signals)

		for {
			s := <-signals
			switch s {
			case syscall.SIGWINCH:
				if hasTerminal {
					resize(state.ControlSocket, false)
				}
			default:
				syscall.Kill(pid, s.(syscall.Signal))
			}
		}
	}()

	// Pipe session to bash and visa-versa
	go func() {
		if !run {
			io.Copy(os.Stdout, conn)
		} else {
			io.Copy(ioutil.Discard, conn)
		}
		wg.Done()
	}()

	go func() {
		io.Copy(conn, os.Stdin)
	}()

	wg.Wait()

	if hasTerminal {
		fmt.Printf("\r")
		return terminal.Restore(0, ostate)
	}

	return nil
}

func exitContainer(containerID string, delete bool) {
	state, err := getState(containerID)
	if err != nil {
		if !delete {
			sylog.Errorf("%s", err)
			os.Exit(1)
		}
		return
	}

	if state.ExitCode != nil {
		defer os.Exit(*state.ExitCode)
	}

	if delete {
		if err := ociDelete(containerID); err != nil {
			sylog.Errorf("%s", err)
		}
	}
}

func ociRun(containerID string) error {
	dir, err := instance.GetDirPrivileged(containerID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	syncSocketPath = filepath.Join(dir, "run.sock")

	l, err := unix.CreateSocket(syncSocketPath)
	if err != nil {
		os.Remove(syncSocketPath)
		return err
	}

	defer l.Close()

	status := make(chan string, 1)

	if err := ociCreate(containerID); err != nil {
		defer os.Remove(syncSocketPath)
		if _, err1 := getState(containerID); err1 != nil {
			return err
		}
		if err := ociDelete(containerID); err != nil {
			sylog.Warningf("can't delete container %s", containerID)
		}
		return err
	}

	defer exitContainer(containerID, true)
	defer os.Remove(syncSocketPath)

	go func() {
		var state specs.State

		for {
			c, err := l.Accept()
			if err != nil {
				status <- err.Error()
				return
			}

			dec := json.NewDecoder(c)
			if err := dec.Decode(&state); err != nil {
				status <- err.Error()
				return
			}

			c.Close()

			switch state.Status {
			case ociruntime.Created:
				// ignore error there and wait for stopped status
				ociStart(containerID)
			case ociruntime.Running:
				status <- state.Status
			case ociruntime.Stopped:
				status <- state.Status
			}
		}
	}()

	// wait running status
	s := <-status
	if s != ociruntime.Running {
		return fmt.Errorf("%s", s)
	}

	engineConfig, err := getEngineConfig(containerID)
	if err != nil {
		return err
	}

	if err := attach(engineConfig, true); err != nil {
		return err
	}

	// wait stopped status
	s = <-status
	if s != ociruntime.Stopped {
		return fmt.Errorf("%s", s)
	}

	return nil
}

func ociAttach(containerID string) error {
	engineConfig, err := getEngineConfig(containerID)
	if err != nil {
		return err
	}

	defer exitContainer(containerID, false)

	return attach(engineConfig, false)
}

func ociStart(containerID string) error {
	state, err := getState(containerID)
	if err != nil {
		return err
	}

	if state.Status != ociruntime.Created {
		return fmt.Errorf("container %s is not created", containerID)
	}

	if state.ControlSocket == "" {
		return fmt.Errorf("can't find control socket")
	}

	ctrl := &ociruntime.Control{}
	ctrl.StartContainer = true

	c, err := unix.Dial(state.ControlSocket)
	if err != nil {
		return fmt.Errorf("failed to connect to control socket")
	}
	defer c.Close()

	enc := json.NewEncoder(c)
	if err != nil {
		return err
	}

	if err := enc.Encode(ctrl); err != nil {
		return err
	}

	// wait runtime close socket connection for ACK
	d := make([]byte, 1)
	if _, err := c.Read(d); err != io.EOF {
		return err
	}

	return nil
}

func ociKill(containerID string, killTimeout int) error {
	// send signal to the instance
	state, err := getState(containerID)
	if err != nil {
		return err
	}

	if state.Status != ociruntime.Created && state.Status != ociruntime.Running {
		return fmt.Errorf("container %s is nor created nor running", containerID)
	}

	sig := syscall.SIGTERM

	if killSignal != "" {
		sig, err = signal.Convert(killSignal)
		if err != nil {
			return err
		}
	}

	if killTimeout > 0 {
		c, err := unix.Dial(state.ControlSocket)
		if err != nil {
			return fmt.Errorf("failed to connect to control socket")
		}
		defer c.Close()

		killed := make(chan bool, 1)

		go func() {
			// wait runtime close socket connection for ACK
			d := make([]byte, 1)
			if _, err := c.Read(d); err == io.EOF {
				killed <- true
			}
		}()

		if err := syscall.Kill(state.Pid, sig); err != nil {
			return err
		}

		select {
		case <-killed:
		case <-time.After(time.Duration(killTimeout) * time.Second):
			return syscall.Kill(state.Pid, syscall.SIGKILL)
		}
	} else {
		return syscall.Kill(state.Pid, sig)
	}

	return nil
}

func ociDelete(containerID string) error {
	engineConfig, err := getEngineConfig(containerID)
	if err != nil {
		return err
	}

	switch engineConfig.State.Status {
	case ociruntime.Running:
		return fmt.Errorf("container is not stopped: running")
	case ociruntime.Stopped:
	case ociruntime.Created:
		if err := ociKill(containerID, 2); err != nil {
			return err
		}
		engineConfig, err = getEngineConfig(containerID)
		if err != nil {
			return err
		}
	}

	hooks := engineConfig.OciConfig.Hooks
	if hooks != nil {
		for _, h := range hooks.Poststop {
			if err := exec.Hook(&h, &engineConfig.State.State); err != nil {
				sylog.Warningf("%s", err)
			}
		}
	}

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

	absBundle, err := filepath.Abs(bundlePath)
	if err != nil {
		return fmt.Errorf("failed to determine bundle absolute path: %s", err)
	}

	if err := os.Chdir(absBundle); err != nil {
		return fmt.Errorf("failed to change directory to %s: %s", absBundle, err)
	}

	engineConfig := oci.NewConfig()
	generator := generate.Generator{Config: &engineConfig.OciConfig.Spec}
	engineConfig.SetBundlePath(absBundle)
	engineConfig.SetLogPath(logPath)
	engineConfig.SetLogFormat(logFormat)
	engineConfig.SetPidFile(pidFile)

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

	Env := []string{sylog.GetEnvVar()}

	engineConfig.EmptyProcess = emptyProcess
	engineConfig.SyncSocket = syncSocketPath

	commonConfig := &config.Common{
		ContainerID:  containerID,
		EngineName:   oci.Name,
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
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

func ociExec(containerID string, cmdArgs []string) error {
	starter := buildcfg.LIBEXECDIR + "/singularity/bin/starter"

	commonConfig, err := getCommonConfig(containerID)
	if err != nil {
		return fmt.Errorf("%s doesn't exist", containerID)
	}

	engineConfig := commonConfig.EngineConfig.(*oci.EngineConfig)

	engineConfig.Exec = true
	engineConfig.OciConfig.SetProcessArgs(cmdArgs)

	os.Clearenv()

	configData, err := json.Marshal(commonConfig)
	if err != nil {
		sylog.Fatalf("%s", err)
	}

	Env := []string{sylog.GetEnvVar()}

	procName := fmt.Sprintf("Singularity OCI %s", containerID)
	return exec.Pipe(starter, []string{procName}, Env, configData)
}

func ociUpdate(containerID string) error {
	var reader io.Reader

	state, err := getState(containerID)
	if err != nil {
		return err
	}

	if state.State.Status != ociruntime.Running && state.State.Status != ociruntime.Created {
		return fmt.Errorf("container %s is neither running nor created", containerID)
	}

	if fromFile == "" {
		return fmt.Errorf("you must specify --from-file")
	}

	resources := &specs.LinuxResources{}
	manager := &cgroups.Manager{Pid: state.State.Pid}

	if fromFile == "-" {
		reader = os.Stdin
	} else {
		f, err := os.Open(fromFile)
		if err != nil {
			return err
		}
		reader = f
	}

	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read cgroups config file: %s", err)
	}

	if err := json.Unmarshal(data, resources); err != nil {
		return err
	}

	return manager.UpdateFromSpec(resources)
}

func ociPauseResume(containerID string, pause bool) error {
	state, err := getState(containerID)
	if err != nil {
		return err
	}

	if state.Status != ociruntime.Running {
		return fmt.Errorf("container %s is not running", containerID)
	}

	manager := &cgroups.Manager{Pid: state.State.Pid}

	if !pause {
		return manager.Resume()
	}

	return manager.Pause()
}
