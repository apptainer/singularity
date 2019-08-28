// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/instance"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/signal"
	"github.com/sylabs/singularity/pkg/cmdline"
	"github.com/sylabs/singularity/pkg/util/fs/proc"
)

func init() {
	cmdManager.RegisterFlagForCmd(&instanceStopUserFlag, instanceStopCmd)
	cmdManager.RegisterFlagForCmd(&instanceStopAllFlag, instanceStopCmd)
	cmdManager.RegisterFlagForCmd(&instanceStopForceFlag, instanceStopCmd)
	cmdManager.RegisterFlagForCmd(&instanceStopSignalFlag, instanceStopCmd)
	cmdManager.RegisterFlagForCmd(&instanceStopTimeoutFlag, instanceStopCmd)
}

// -u|--user
var instanceStopUser string
var instanceStopUserFlag = cmdline.Flag{
	ID:           "instanceStopUserFlag",
	Value:        &instanceStopUser,
	DefaultValue: "",
	Name:         "user",
	ShortHand:    "u",
	Usage:        "If running as root, stop instances belonging to user",
	Tag:          "<username>",
	EnvKeys:      []string{"USER"},
}

// -a|--all
var instanceStopAll bool
var instanceStopAllFlag = cmdline.Flag{
	ID:           "instanceStopAllFlag",
	Value:        &instanceStopAll,
	DefaultValue: false,
	Name:         "all",
	ShortHand:    "a",
	Usage:        "stop all user's instances",
	EnvKeys:      []string{"ALL"},
}

// -f|--force
var instanceStopForce bool
var instanceStopForceFlag = cmdline.Flag{
	ID:           "instanceStopForceFlag",
	Value:        &instanceStopForce,
	DefaultValue: false,
	Name:         "force",
	ShortHand:    "F",
	Usage:        "force kill instance",
	EnvKeys:      []string{"FORCE"},
}

// -s|--signal
var instanceStopSignal string
var instanceStopSignalFlag = cmdline.Flag{
	ID:           "instanceStopSignalFlag",
	Value:        &instanceStopSignal,
	DefaultValue: "",
	Name:         "signal",
	ShortHand:    "s",
	Usage:        "signal sent to the instance",
	Tag:          "<signal>",
	EnvKeys:      []string{"SIGNAL"},
}

// -t|--timeout
var instanceStopTimeout int
var instanceStopTimeoutFlag = cmdline.Flag{
	ID:           "instanceStopTimeoutFlag",
	Value:        &instanceStopTimeout,
	DefaultValue: 10,
	Name:         "timeout",
	ShortHand:    "t",
	Usage:        "force kill non stopped instances after X seconds",
}

// singularity instance stop
var instanceStopCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 && !instanceStopAll {
			stopInstance(args[0])
			return nil
		}
		if instanceStopAll {
			stopInstance("*")
			return nil
		}
		return errors.New("invalid command")
	},

	Use:     docs.InstanceStopUse,
	Short:   docs.InstanceStopShort,
	Long:    docs.InstanceStopLong,
	Example: docs.InstanceStopExample,
}

func stopInstance(name string) {
	sig := syscall.SIGINT
	uid := os.Getuid()
	fileChan := make(chan *instance.File, 1)
	stopped := make([]int, 0)

	if instanceStopUser != "" && uid != 0 {
		sylog.Fatalf("only root user can list user's instances")
	}
	if instanceStopSignal != "" {
		var err error

		sig, err = signal.Convert(instanceStopSignal)
		if err != nil {
			sylog.Fatalf("%s", err)
		}
	}
	if instanceStopForce {
		sig = syscall.SIGKILL
	}
	files, err := instance.List(instanceStopUser, name, instance.SingSubDir)
	if err != nil {
		sylog.Fatalf("failed to retrieve instance list: %s", err)
	}
	if len(files) == 0 {
		sylog.Fatalf("no instance found")
	}

	for _, file := range files {
		go killInstance(file, sig, fileChan)
	}

	for {
		select {
		case f := <-fileChan:
			fmt.Printf("Stopping %s instance of %s (PID=%d)\n", f.Name, f.Image, f.Pid)
			stopped = append(stopped, f.Pid)
			if len(stopped) == len(files) {
				os.Exit(0)
			}
		case <-time.After(time.Duration(instanceStopTimeout) * time.Second):
			for _, file := range files {
				kill := true
				for _, pid := range stopped {
					if pid == file.Pid {
						kill = false
						break
					}
				}
				if !kill {
					continue
				}
				syscall.Kill(file.Pid, syscall.SIGKILL)
				fmt.Printf("Killing %s instance of %s (PID=%d) (Timeout)\n", file.Name, file.Image, file.Pid)
			}
			os.Exit(0)
		}
	}
}

func killInstance(file *instance.File, sig syscall.Signal, fileChan chan *instance.File) {
	syscall.Kill(file.Pid, sig)

	for {
		if err := syscall.Kill(file.PPid, 0); err == syscall.ESRCH {
			fileChan <- file
			break
		} else if childs, err := proc.CountChilds(file.Pid); childs == 0 {
			if err == nil {
				syscall.Kill(file.Pid, syscall.SIGKILL)
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}
