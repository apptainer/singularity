// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cgroups

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

func readIntFromFile(path string) (int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		return strconv.ParseInt(scanner.Text(), 10, 64)
	}

	return 0, fmt.Errorf("no data found")
}

func TestCgroups(t *testing.T) {
	test.EnsurePrivilege(t)

	cmd := exec.Command("/bin/cat")
	pipe, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	pid := cmd.Process.Pid
	strPid := strconv.Itoa(pid)
	path := filepath.Join("/singularity", strPid)

	manager := &Manager{Pid: pid, Path: path}
	if err := manager.ApplyFromFile("example/cgroups.toml"); err != nil {
		t.Errorf("%s", err)
	}

	defer manager.Remove()

	rootPath := manager.GetCgroupRootPath()
	if rootPath == "" {
		t.Fatalf("can't determine cgroups root path, is cgroups enabled ?")
	}

	cpuShares := filepath.Join(rootPath, "cpu", path, "cpu.shares")

	i, err := readIntFromFile(cpuShares)
	if err != nil {
		t.Errorf("failed to read %s: %s", cpuShares, err)
	}
	if i != 1024 {
		t.Errorf("cpu shares should be equal to 1024")
	}

	content := []byte("[cpu]\nshares = 512")
	tmpfile, err := ioutil.TempFile("", "cgroups")
	if err != nil {
		t.Fatal(err)
	}

	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// test update/load from PID
	manager = &Manager{Pid: pid}

	if err := manager.UpdateFromFile(tmpfile.Name()); err != nil {
		t.Fatal(err)
	}
	i, err = readIntFromFile(cpuShares)
	if err != nil {
		t.Errorf("failed to read %s: %s", cpuShares, err)
	}
	if i != 512 {
		t.Errorf("cpu shares should be equal to 512")
	}

	pipe.Close()

	cmd.Wait()
}

func TestPauseResume(t *testing.T) {
	test.EnsurePrivilege(t)

	manager := &Manager{}
	if err := manager.Pause(); err == nil {
		t.Errorf("unexpected success with PID 0")
	}
	if err := manager.Resume(); err == nil {
		t.Errorf("unexpected success with PID 0")
	}

	cmd := exec.Command("/bin/cat")
	pipe, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	manager.Pid = cmd.Process.Pid
	manager.Path = filepath.Join("/singularity", strconv.Itoa(manager.Pid))

	if err := manager.ApplyFromFile("example/cgroups.toml"); err != nil {
		t.Errorf("%s", err)
	}

	manager.Pause()

	file, err := os.Open(fmt.Sprintf("/proc/%d/status", manager.Pid))
	if err != nil {
		t.Error(err)
	}

	scanner := bufio.NewScanner(file)
	stateOk := false

	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "State:\tD") {
			stateOk = true
			break
		}
	}

	if !stateOk {
		t.Errorf("failed to pause process %d", manager.Pid)
	}

	file.Close()

	manager.Resume()

	file, err = os.Open(fmt.Sprintf("/proc/%d/status", manager.Pid))
	if err != nil {
		t.Error(err)
	}

	scanner = bufio.NewScanner(file)
	stateOk = false

	for scanner.Scan() {
		text := scanner.Text()
		if strings.HasPrefix(text, "State:\tS") || strings.HasPrefix(text, "State:\tR") {
			stateOk = true
			break
		}
	}

	if !stateOk {
		t.Errorf("failed to resume process %d", manager.Pid)
	}

	file.Close()

	defer manager.Remove()

	pipe.Close()

	cmd.Wait()
}
