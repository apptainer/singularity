// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package imgbuild

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// PrestartProcess _
func (e *EngineOperations) PrestartProcess() error {
	return nil
}

// StartProcess runs the %post script
func (e *EngineOperations) StartProcess() error {

	// Run %post scripts here
	runScriptSections("post", e.EngineConfig.Recipe.BuildData.Post)

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
