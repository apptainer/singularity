// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"os"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/plugin"
	singularitycallback "github.com/sylabs/singularity/pkg/plugin/callback/runtime/engine/singularity"
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
// not need them for wait4 and kill syscalls.
func (e *EngineOperations) MonitorContainer(pid int, signals chan os.Signal) (syscall.WaitStatus, error) {
	var status syscall.WaitStatus

	callbackType := (singularitycallback.MonitorContainer)(nil)
	callbacks, err := plugin.LoadCallbacks(callbackType)
	if err != nil {
		return status, fmt.Errorf("while loading plugins callbacks '%T': %s", callbackType, err)
	}
	if len(callbacks) > 1 {
		return status, fmt.Errorf("multiple plugins have registered callback for '%T'", callbackType)
	} else if len(callbacks) == 1 {
		return callbacks[0].(singularitycallback.MonitorContainer)(e.CommonConfig, pid, signals)
	}

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
			if e.EngineConfig.GetSignalPropagation() {
				if err := syscall.Kill(pid, s.(syscall.Signal)); err != nil {
					return status, fmt.Errorf("interrupted by signal %s", s.String())
				}
			}
			// Handle CTRL-Z and send ourself a SIGSTOP to implicitly send SIGCHLD
			// signal to parent process as this process is the direct child
			if s == syscall.SIGTSTP {
				if err := syscall.Kill(os.Getpid(), syscall.SIGSTOP); err != nil {
					return status, fmt.Errorf("received SIGTSTP but was not able to stop")
				}
			}
		}
	}
}
