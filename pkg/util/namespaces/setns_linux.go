// Copyright (c) 2018-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

//go:build go1.10
// +build go1.10

package namespaces

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
)

var setnsSysNo = map[string]uintptr{
	"386":     346,
	"arm64":   268,
	"amd64":   308,
	"arm":     375,
	"ppc":     350,
	"ppc64":   350,
	"ppc64le": 350,
	"s390x":   339,
}

var nsMap = map[string]uintptr{
	"ipc": syscall.CLONE_NEWIPC,
	"net": syscall.CLONE_NEWNET,
	"mnt": syscall.CLONE_NEWNS,
	"uts": syscall.CLONE_NEWUTS,
}

// Enter enters in provided process namespace.
func Enter(pid int, namespace string) error {
	flag, ok := nsMap[namespace]
	if !ok {
		return fmt.Errorf("namespace %s not supported", namespace)
	}

	path := fmt.Sprintf("/proc/%d/ns/%s", pid, namespace)
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("can't open namespace path %s: %s", path, err)
	}
	defer f.Close()

	ns, ok := setnsSysNo[runtime.GOARCH]
	if !ok {
		return fmt.Errorf("unsupported platform %s", runtime.GOARCH)
	}

	_, _, errSys := syscall.RawSyscall(ns, f.Fd(), flag, 0)
	if errSys != 0 {
		return errSys
	}

	return nil
}
