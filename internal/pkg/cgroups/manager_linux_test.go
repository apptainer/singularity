// Copyright (c) 2018-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cgroups

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

// ensureIntInFile asserts that the content of path is the inteeger wantInt
func ensureIntInFile(t *testing.T, path string, wantInt int64) {
	file, err := os.Open(path)
	if err != nil {
		t.Errorf("while opening %q: %v", path, err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	hasData := scanner.Scan()
	if !hasData {
		t.Errorf("no data found in %q", path)
	}

	val, err := strconv.ParseInt(scanner.Text(), 10, 64)
	if err != nil {
		t.Errorf("could not parse %q: %v", path, err)
	}

	if val != wantInt {
		t.Errorf("found %d in %q, expected %d", val, path, wantInt)
	}
}

// ensureState asserts that a process pid has the required state
func ensureState(t *testing.T, pid int, wantStates string) {
	file, err := os.Open(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		t.Error(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	procState := ""

	for scanner.Scan() {
		// State:	R (running)
		if strings.HasPrefix(scanner.Text(), "State:\t") {
			f := strings.Fields(scanner.Text())
			if len(f) < 2 {
				t.Errorf("Could not check process state - not enough fields: %s", scanner.Text())
			}
			procState = f[1]
		}
	}

	if !strings.ContainsAny(procState, wantStates) {
		t.Errorf("Process %d had state %q, expected state %q", pid, procState, wantStates)
	}
}
