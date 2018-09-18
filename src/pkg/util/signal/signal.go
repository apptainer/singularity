// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package signal

import (
	"fmt"
	"strconv"
	"strings"
	"syscall"
)

var signalMap = map[string]syscall.Signal{
	"SIGHUP":    syscall.SIGHUP,
	"SIGINT":    syscall.SIGINT,
	"SIGQUIT":   syscall.SIGQUIT,
	"SIGILL":    syscall.SIGILL,
	"SIGTRAP":   syscall.SIGTRAP,
	"SIGABRT":   syscall.SIGABRT,
	"SIGBUS":    syscall.SIGBUS,
	"SIGFPE":    syscall.SIGFPE,
	"SIGKILL":   syscall.SIGKILL,
	"SIGUSR1":   syscall.SIGUSR1,
	"SIGSEGV":   syscall.SIGSEGV,
	"SIGUSR2":   syscall.SIGUSR2,
	"SIGPIPE":   syscall.SIGPIPE,
	"SIGALRM":   syscall.SIGALRM,
	"SIGTERM":   syscall.SIGTERM,
	"SIGSTKFLT": syscall.SIGSTKFLT,
	"SIGCHLD":   syscall.SIGCHLD,
	"SIGCONT":   syscall.SIGCONT,
	"SIGSTOP":   syscall.SIGSTOP,
	"SIGTSTP":   syscall.SIGTSTP,
	"SIGTTIN":   syscall.SIGTTIN,
	"SIGTTOU":   syscall.SIGTTOU,
	"SIGURG":    syscall.SIGURG,
	"SIGXCPU":   syscall.SIGXCPU,
	"SIGXFSZ":   syscall.SIGXFSZ,
	"SIGVTALRM": syscall.SIGVTALRM,
	"SIGPROF":   syscall.SIGPROF,
	"SIGWINCH":  syscall.SIGWINCH,
	"SIGIO":     syscall.SIGIO,
	"SIGPWR":    syscall.SIGPWR,
	"SIGSYS":    syscall.SIGSYS,
}

const signalMax = syscall.SIGSYS

// Convert converts a signal string to corresponding signal number
func Convert(sig string) (syscall.Signal, error) {
	var sigNum syscall.Signal

	if strings.HasPrefix(sig, "SIG") {
		if sigNum, ok := signalMap[sig]; ok {
			return sigNum, nil
		}
	}

	if sigNum, ok := signalMap["SIG"+sig]; ok {
		return sigNum, nil
	}

	sigConv, err := strconv.ParseInt(sig, 10, 32)
	if err == nil {
		if sigConv <= int64(signalMax) && sigConv > 0 {
			return syscall.Signal(sigConv), nil
		}
	}

	return sigNum, fmt.Errorf("can't convert %s to signal number", sig)
}
