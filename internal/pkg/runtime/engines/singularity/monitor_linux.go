// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"os"
	"syscall"
)

// MonitorContainer monitors a container
func (e *EngineOperations) MonitorContainer(pid int, signals chan os.Signal) (syscall.WaitStatus, error) {
	var status syscall.WaitStatus

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
			if e.EngineConfig.GetSignalPropagation() {
				if err := syscall.Kill(pid, s.(syscall.Signal)); err != nil {
					return status, fmt.Errorf("interrupted by signal %s", s.String())
				}
			}
		}
	}
}
