// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package signal

/*
#include <signal.h>

void raiseSignal(int sig) {
	signal(sig, SIG_DFL);
	raise(sig);
}
*/
import "C"

import (
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// similarSignals maps similar signals not handled
// by unix package.
var similarSignals = map[string]string{
	"SIGIOT":  "SIGABRT",
	"SIGCLD":  "SIGCHLD",
	"SIGPOLL": "SIGIO",
}

// Convert converts a signal string to corresponding signal number
func Convert(sig string) (unix.Signal, error) {
	sigStr := strings.ToUpper(sig)

	if !strings.HasPrefix(sigStr, "SIG") {
		sigStr = "SIG" + sigStr
	}
	if s, ok := similarSignals[sigStr]; ok {
		sigStr = s
	}

	sigNum := unix.SignalNum(sigStr)
	if sigNum != 0 {
		return sigNum, nil
	}

	sigConv, err := strconv.ParseInt(sig, 10, 32)
	if err != nil {
		return sigNum, fmt.Errorf("%s is not a number", sig)
	}

	sigName := unix.SignalName(unix.Signal(sigConv))
	sigNum = unix.SignalNum(sigName)
	if sigNum == 0 {
		return sigNum, fmt.Errorf("can't convert %s to signal number", sig)
	}

	return sigNum, nil
}

// Raise sends a signal to the current process and ensure the
// current signal handler is set to its default handler for the
// corresponding signal. It allows to send signals like SIGABRT
// without triggering Go core dump handler.
func Raise(sig unix.Signal) {
	// signal handlers like the one for SIGABRT can't be reset
	// easily from Go without using rt_sigaction syscall implying
	// to defines sigaction structure for various architecture so
	// let's proceed with C
	C.raiseSignal(C.int(int(sig)))
}
