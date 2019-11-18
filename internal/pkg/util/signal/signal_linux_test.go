// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package signal

import (
	"testing"

	"golang.org/x/sys/unix"
)

var signalOK = []struct {
	tests  []string
	signal unix.Signal
}{
	{[]string{"SIGHUP", "HUP", "1"}, unix.SIGHUP},
	{[]string{"SIGINT", "INT", "2"}, unix.SIGINT},
	{[]string{"SIGQUIT", "QUIT", "3"}, unix.SIGQUIT},
	{[]string{"SIGILL", "ILL", "4"}, unix.SIGILL},
	{[]string{"SIGTRAP", "TRAP", "5"}, unix.SIGTRAP},
	{[]string{"SIGABRT", "ABRT", "6"}, unix.SIGABRT},
	{[]string{"SIGIOT", "IOT", "6"}, unix.SIGIOT},
	{[]string{"SIGBUS", "BUS", "7"}, unix.SIGBUS},
	{[]string{"SIGFPE", "FPE", "8"}, unix.SIGFPE},
	{[]string{"SIGKILL", "KILL", "9"}, unix.SIGKILL},
	{[]string{"SIGUSR1", "USR1", "10"}, unix.SIGUSR1},
	{[]string{"SIGSEGV", "SEGV", "11"}, unix.SIGSEGV},
	{[]string{"SIGUSR2", "USR2", "12"}, unix.SIGUSR2},
	{[]string{"SIGPIPE", "PIPE", "13"}, unix.SIGPIPE},
	{[]string{"SIGALRM", "ALRM", "14"}, unix.SIGALRM},
	{[]string{"SIGTERM", "TERM", "15"}, unix.SIGTERM},
	{[]string{"SIGCHLD", "CHLD", "17"}, unix.SIGCHLD},
	{[]string{"SIGCLD", "CLD", "17"}, unix.SIGCLD},
	{[]string{"SIGCONT", "CONT", "18"}, unix.SIGCONT},
	{[]string{"SIGSTOP", "STOP", "19"}, unix.SIGSTOP},
	{[]string{"SIGTSTP", "TSTP", "20"}, unix.SIGTSTP},
	{[]string{"SIGTTIN", "TTIN", "21"}, unix.SIGTTIN},
	{[]string{"SIGTTOU", "TTOU", "22"}, unix.SIGTTOU},
	{[]string{"SIGURG", "URG", "23"}, unix.SIGURG},
	{[]string{"SIGXCPU", "XCPU", "24"}, unix.SIGXCPU},
	{[]string{"SIGXFSZ", "XFSZ", "25"}, unix.SIGXFSZ},
	{[]string{"SIGVTALRM", "VTALRM", "26"}, unix.SIGVTALRM},
	{[]string{"SIGPROF", "PROF", "27"}, unix.SIGPROF},
	{[]string{"SIGWINCH", "WINCH", "28"}, unix.SIGWINCH},
	{[]string{"SIGIO", "IO", "29"}, unix.SIGIO},
	{[]string{"SIGPOLL", "POLL", "29"}, unix.SIGPOLL},
	{[]string{"SIGPWR", "PWR", "30"}, unix.SIGPWR},
	{[]string{"SIGSYS", "SYS", "31"}, unix.SIGSYS},
}

var signalKO = []struct {
	tests  []string
	signal unix.Signal
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
