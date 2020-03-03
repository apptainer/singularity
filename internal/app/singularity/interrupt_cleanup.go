// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"os"
	"os/signal"
	"syscall"
)

// interruptCleanup will watch for a interrupt signal, if there's
// one detected, then it will remove all the specified file(s)
func interruptCleanup(f func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	f()
	os.Exit(1)
}
