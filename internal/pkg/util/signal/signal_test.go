// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package signal

import (
	"syscall"
	"testing"
)

var signalOK = []struct {
	tests  []string
	signal syscall.Signal
}{
	{[]string{"SIGHUP", "HUP", "1"}, syscall.SIGHUP},
	{[]string{"SIGINT", "INT", "2"}, syscall.SIGINT},
	{[]string{"SIGQUIT", "QUIT", "3"}, syscall.SIGQUIT},
	{[]string{"SIGILL", "ILL", "4"}, syscall.SIGILL},
	{[]string{"SIGTRAP", "TRAP", "5"}, syscall.SIGTRAP},
	{[]string{"SIGABRT", "ABRT", "6"}, syscall.SIGABRT},
	{[]string{"SIGBUS", "BUS", "7"}, syscall.SIGBUS},
	{[]string{"SIGFPE", "FPE", "8"}, syscall.SIGFPE},
	{[]string{"SIGKILL", "KILL", "9"}, syscall.SIGKILL},
	{[]string{"SIGUSR1", "USR1", "10"}, syscall.SIGUSR1},
	{[]string{"SIGSEGV", "SEGV", "11"}, syscall.SIGSEGV},
	{[]string{"SIGUSR2", "USR2", "12"}, syscall.SIGUSR2},
	{[]string{"SIGPIPE", "PIPE", "13"}, syscall.SIGPIPE},
	{[]string{"SIGALRM", "ALRM", "14"}, syscall.SIGALRM},
	{[]string{"SIGTERM", "TERM", "15"}, syscall.SIGTERM},
	{[]string{"SIGSTKFLT", "STKFLT", "16"}, syscall.SIGSTKFLT},
	{[]string{"SIGCHLD", "CHLD", "17"}, syscall.SIGCHLD},
	{[]string{"SIGCONT", "CONT", "18"}, syscall.SIGCONT},
	{[]string{"SIGSTOP", "STOP", "19"}, syscall.SIGSTOP},
	{[]string{"SIGTSTP", "TSTP", "20"}, syscall.SIGTSTP},
	{[]string{"SIGTTIN", "TTIN", "21"}, syscall.SIGTTIN},
	{[]string{"SIGTTOU", "TTOU", "22"}, syscall.SIGTTOU},
	{[]string{"SIGURG", "URG", "23"}, syscall.SIGURG},
	{[]string{"SIGXCPU", "XCPU", "24"}, syscall.SIGXCPU},
	{[]string{"SIGXFSZ", "XFSZ", "25"}, syscall.SIGXFSZ},
	{[]string{"SIGVTALRM", "VTALRM", "26"}, syscall.SIGVTALRM},
	{[]string{"SIGPROF", "PROF", "27"}, syscall.SIGPROF},
	{[]string{"SIGWINCH", "WINCH", "28"}, syscall.SIGWINCH},
	{[]string{"SIGIO", "IO", "29"}, syscall.SIGIO},
	{[]string{"SIGPWR", "PWR", "30"}, syscall.SIGPWR},
	{[]string{"SIGSYS", "SYS", "31"}, syscall.SIGSYS},
}

var signalKO = []struct {
	tests  []string
	signal syscall.Signal
}{
	{[]string{"SIGNULL", "NULL", "0"}, 0},
}

func TestConvert(t *testing.T) {
	for _, test := range signalOK {
		for _, sig := range test.tests {
			if s, err := Convert(sig); s != test.signal && err != nil {
				t.Error(err)
			}
		}
	}
	for _, test := range signalKO {
		for _, sig := range test.tests {
			if s, err := Convert(sig); s != test.signal && err == nil {
				t.Error(err)
			}
		}
	}
}
