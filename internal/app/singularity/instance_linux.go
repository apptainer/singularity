// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"encoding/json"
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
	"github.com/sylabs/singularity/pkg/util/fs/proc"
)

// instance list/stop options
var username string

// instance list options
var jsonFormat bool

// instance stop options
var stopSignal string
var stopAll bool
var forceStop bool
var stopTimeout int

func init() {
	SingularityCmd.AddCommand(InstanceCmd)
	InstanceCmd.AddCommand(InstanceStartCmd)
	InstanceCmd.AddCommand(InstanceStopCmd)
	InstanceCmd.AddCommand(InstanceListCmd)
}

// InstanceCmd singularity instance
var InstanceCmd = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("Invalid command")
	},
	DisableFlagsInUseLine: true,

	Use:           docs.InstanceUse,
	Short:         docs.InstanceShort,
	Long:          docs.InstanceLong,
	Example:       docs.InstanceExample,
	SilenceErrors: true,
}

func listInstance() {
	uid := os.Getuid()
	if username != "" && uid != 0 {
		sylog.Fatalf("only root user can list user's instances")
	}
	files, err := instance.List(username, "*")
	if err != nil {
		sylog.Fatalf("failed to retrieve instance list: %s", err)
	}
	if !jsonFormat {
		fmt.Printf("%-16s %-8s %s\n", "INSTANCE NAME", "PID", "IMAGE")
		for _, file := range files {
			fmt.Printf("%-16s %-8d %s\n", file.Name, file.Pid, file.Image)
		}
	} else {
		output := make(map[string][]jsonList)
		output["instances"] = make([]jsonList, len(files))

		for i := range output["instances"] {
			output["instances"][i].Image = files[i].Image
			output["instances"][i].Pid = files[i].Pid
			output["instances"][i].Instance = files[i].Name
		}

		c, err := json.MarshalIndent(output, "", "\t")
		if err != nil {
			sylog.Fatalf("error while printing structured JSON: %s", err)
		}
		fmt.Println(string(c))
	}
}

func killInstance(file *instance.File, sig syscall.Signal, fileChan chan *instance.File) {
	syscall.Kill(file.Pid, sig)

	for {
		if err := syscall.Kill(file.Pid, 0); err == syscall.ESRCH {
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

func stopInstance(name string) {
	sig := syscall.SIGINT
	uid := os.Getuid()
	fileChan := make(chan *instance.File, 1)
	stopped := make([]int, 0)

	if username != "" && uid != 0 {
		sylog.Fatalf("only root user can list user's instances")
	}
	if stopSignal != "" {
		var err error

		sig, err = signal.Convert(stopSignal)
		if err != nil {
			sylog.Fatalf("%s", err)
		}
	}
	if forceStop {
		sig = syscall.SIGKILL
	}
	files, err := instance.List(username, name)
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
		case <-time.After(time.Duration(stopTimeout) * time.Second):
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
