// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// MonitorContainer monitors a container
func (engine *EngineOperations) MonitorContainer(pid int) (syscall.WaitStatus, error) {
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
