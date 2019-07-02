// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package namespaces

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

// IsInsideUserNamespace checks if a process is already running in a
// user namespace and also returns if the process has permissions to use
// setgroups in this user namespace.
func IsInsideUserNamespace(pid int) (bool, bool) {
	// default values returned in case of error
	insideUserNs := false
	setgroupsAllowed := false

	// can fail if the kernel doesn't support user namespace
	r, err := os.Open(fmt.Sprintf("/proc/%d/uid_map", pid))
	if err != nil {
		return insideUserNs, setgroupsAllowed
	}
	defer r.Close()

	scanner := bufio.NewScanner(r)
	// we are interested only by the first line of
	// uid_map which would give us the answer quickly
	// based on the value of size field
	if scanner.Scan() {
		fields := strings.Fields(scanner.Text())

		// trust values returned by procfs
		size, _ := strconv.ParseUint(fields[2], 10, 32)

		// a size of 4294967295 means the process is running
		// in the host user namespace
		if uint32(size) == ^uint32(0) {
			return insideUserNs, setgroupsAllowed
		}

		// process is running inside user namespace
		insideUserNs = true

		// should not fail if open call passed
		d, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/setgroups", pid))
		if err != nil {
			return insideUserNs, setgroupsAllowed
		}
		setgroupsAllowed = string(d) == "allow\n"
	}

	return insideUserNs, setgroupsAllowed
}
