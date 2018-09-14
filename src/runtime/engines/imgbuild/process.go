// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package imgbuild

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/singularityware/singularity/src/pkg/sylog"
)

// StartProcess runs the %post script
func (e *EngineOperations) StartProcess(masterConn net.Conn) error {
	// Run %post script here

	post := exec.Command("/bin/sh", "-c", e.EngineConfig.Recipe.BuildData.Post)
	post.Env = e.EngineConfig.OciConfig.Process.Env
	post.Stdout = os.Stdout
	post.Stderr = os.Stderr

	sylog.Infof("Running %%post script\n")
	if err := post.Start(); err != nil {
		sylog.Fatalf("failed to start %%post proc: %v\n", err)
	}
	if err := post.Wait(); err != nil {
		sylog.Fatalf("post proc: %v\n", err)
	}
	sylog.Infof("Finished running %%post script. exit status 0\n")

	// Run %test script here if its defined
	// this also needs to consider the --notest flag from the CLI eventually
	if e.EngineConfig.Recipe.BuildData.Test != "" {
		test := exec.Command("/bin/sh", "-c", e.EngineConfig.Recipe.BuildData.Test)
		test.Stdout = os.Stdout
		test.Stderr = os.Stderr

		sylog.Infof("Running %%test script\n")
		if err := test.Start(); err != nil {
			sylog.Fatalf("failed to start %%test proc: %v\n", err)
		}
		if err := test.Wait(); err != nil {
			sylog.Fatalf("test proc: %v\n", err)
		}
		sylog.Infof("Finished running %%test script. exit status 0\n")
	}

	os.Exit(0)
	return nil
}

// MonitorContainer is responsible for waiting on container process
func (e *EngineOperations) MonitorContainer(pid int) (syscall.WaitStatus, error) {
	var status syscall.WaitStatus

	signals := make(chan os.Signal, 1)
	signal.Notify(signals)

	for {
		s := <-signals
		switch s {
		case syscall.SIGCHLD:
			if wpid, err := syscall.Wait4(pid, &status, syscall.WNOHANG, nil); err != nil {
				return status, fmt.Errorf("error while waiting child: %s", err)
			} else if wpid != pid {
				continue
			}
			return status, nil
		default:
			return status, fmt.Errorf("interrupted by signal %s", s.String())
		}
	}
}

// CleanupContainer _
func (e *EngineOperations) CleanupContainer() error {
	return nil
}

// PostStartProcess actually does nothing for build engine
func (e *EngineOperations) PostStartProcess(pid int) error {
	return nil
}
