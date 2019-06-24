// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/sylog"
)

func SignalHandlerInterrupt(image ...string) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	for _, f := range image {
		sylog.Debugf("Removing file: %q because of receiving termination signal", f)
		err := os.Remove(f)
		if err != nil {
			sylog.Debugf("ERROR: unable to remove: %s: %v", f, err)
		}
	}
	os.Exit(1)
}
