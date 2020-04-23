// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package oci

import (
	"fmt"
	"os"
	"syscall"
)

// MonitorContainer is called from master once the container has
// been spawned. It will block until the container exists.
//
// Additional privileges may be gained when running
// in suid flow. However, when a user namespace is requested and it is not
// a hybrid workflow (e.g. fakeroot), then there is no privileged saved uid
// and thus no additional privileges can be gained.
//
// Particularly here no additional privileges are gained as monitor does
// not need them for wait4 and kill syscalls. However, most likely this
// still will be executed as root since `singularity oci` command set requires
// privileged execution.
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
		case syscall.SIGURG:
			// Ignore SIGURG, which is used for non-cooperative goroutine
			// preemption starting with Go 1.14. For more information, see
			// https://github.com/golang/go/issues/24543.
			break
		default:
			if err := syscall.Kill(pid, s.(syscall.Signal)); err != nil {
				return status, fmt.Errorf("interrupted by signal %s", s.String())
			}
		}
	}
}
