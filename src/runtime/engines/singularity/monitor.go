// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/singularityware/singularity/src/pkg/sylog"
)

// MonitorContainer monitors a container
func (engine *EngineOperations) MonitorContainer() error {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGCHLD)

	s := <-signals
	switch s {
	case syscall.SIGCHLD:
		var status syscall.WaitStatus
		syscall.Wait4(-1, &status, syscall.WNOHANG, nil)
		sylog.Debugf("received from monitor")
	}
	return nil
}
