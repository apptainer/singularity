// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build seccomp

package seccomp

import (
	"syscall"
	"testing"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity/internal/pkg/test"
)

func defaultProfile() *specs.LinuxSeccomp {
	syscalls := []specs.LinuxSyscall{
		{
			Names: []string{
				"brk",
				"chdir",
				"chmod",
				"chown",
				"chown32",
				"clock_getres",
				"clock_gettime",
				"clock_nanosleep",
				"close",
				"dup",
				"dup2",
				"dup3",
				"epoll_create",
				"epoll_create1",
				"epoll_ctl",
				"epoll_ctl_old",
				"epoll_pwait",
				"epoll_wait",
				"epoll_wait_old",
				"eventfd",
				"eventfd2",
				"exit",
				"exit_group",
				"fchdir",
				"fchmod",
				"fchown",
				"fchown32",
				"flock",
				"futex",
				"getcpu",
				"getcwd",
				"getitimer",
				"getpgid",
				"getpgrp",
				"getpid",
				"getppid",
				"getpriority",
				"getrandom",
				"getrlimit",
				"getrusage",
				"getsid",
				"getsockname",
				"getsockopt",
				"get_thread_area",
				"gettid",
				"gettimeofday",
				"io_cancel",
				"ioctl",
				"io_destroy",
				"io_getevents",
				"ioprio_get",
				"ioprio_set",
				"io_setup",
				"io_submit",
				"ipc",
				"kill",
				"lchown",
				"lchown32",
				"link",
				"listen",
				"lseek",
				"lstat",
				"lstat64",
				"madvise",
				"mincore",
				"mkdir",
				"mmap",
				"mmap2",
				"mprotect",
				"mremap",
				"msync",
				"munlock",
				"munlockall",
				"munmap",
				"nanosleep",
				"_newselect",
				"open",
				"pause",
				"pipe",
				"pipe2",
				"poll",
				"ppoll",
				"prctl",
				"pread64",
				"preadv",
				"prlimit64",
				"pselect6",
				"pwrite64",
				"pwritev",
				"read",
				"readahead",
				"readlink",
				"readv",
				"recv",
				"recvfrom",
				"recvmmsg",
				"recvmsg",
				"remap_file_pages",
				"rename",
				"restart_syscall",
				"rmdir",
				"rt_sigaction",
				"rt_sigpending",
				"rt_sigprocmask",
				"rt_sigqueueinfo",
				"rt_sigreturn",
				"rt_sigsuspend",
				"rt_sigtimedwait",
				"rt_tgsigqueueinfo",
				"select",
				"send",
				"sendfile",
				"sendfile64",
				"sendmmsg",
				"sendmsg",
				"sendto",
				"setitimer",
				"setpgid",
				"setpriority",
				"setrlimit",
				"set_robust_list",
				"setsid",
				"setsockopt",
				"set_thread_area",
				"set_tid_address",
				"shutdown",
				"sigaltstack",
				"sigreturn",
				"splice",
				"stat",
				"stat64",
				"sync",
				"sync_file_range",
				"syncfs",
				"sysinfo",
				"syslog",
				"tee",
				"tgkill",
				"time",
				"times",
				"tkill",
				"umask",
				"uname",
				"unlink",
				"utime",
				"utimes",
				"vmsplice",
				"wait4",
				"waitid",
				"waitpid",
				"write",
				"writev",
			},
			Action: specs.ActAllow,
			Args:   []specs.LinuxSeccompArg{},
		},
		{
			Names:  []string{"mount"},
			Action: specs.ActAllow,
			Args: []specs.LinuxSeccompArg{
				{
					Index:    3,
					Value:    syscall.MS_NOSUID | syscall.MS_NODEV,
					ValueTwo: syscall.MS_NOSUID | syscall.MS_NODEV,
					Op:       specs.OpMaskedEqual,
				},
			},
		},
	}
	return &specs.LinuxSeccomp{
		DefaultAction: specs.ActErrno,
		Syscalls:      syscalls,
	}
}

func TestLoadSeccompConfig(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	if err := LoadSeccompConfig(nil); err == nil {
		t.Errorf("shoud have failed with an empty config")
	}
	if err := LoadSeccompConfig(defaultProfile()); err != nil {
		t.Errorf("%s", err)
	}
	if hasConditionSupport() {
		// with default action as ActErrno mount don't return error
		if err := syscall.Mount("/etc", "/mnt", "", syscall.MS_BIND, ""); err != nil {
			t.Errorf("mount syscall allowed: %s", err)
		}
		// without MS_NODEV, mount don't return error here too
		if err := syscall.Mount("/etc", "/mnt", "", syscall.MS_BIND|syscall.MS_NOSUID, ""); err != nil {
			t.Errorf("mount syscall allowed: %s", err)
		}
		// by passing MS_NOSUID and MS_NODEV, mount is allowed by the filter and returns permission denied
		if err := syscall.Mount("/etc", "/mnt", "", syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_NODEV, ""); err == nil {
			t.Errorf("mount syscall filter failed")
		}
	}
}
