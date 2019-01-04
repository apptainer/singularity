// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package proc

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// HasFilesystem returns whether kernel support filesystem or not
func HasFilesystem(fs string) (bool, error) {
	p, err := os.Open("/proc/filesystems")
	if err != nil {
		return false, fmt.Errorf("can't open /proc/filesystems: %s", err)
	}
	defer p.Close()

	suffix := "\t" + fs
	scanner := bufio.NewScanner(p)
	for scanner.Scan() {
		if strings.HasSuffix(scanner.Text(), suffix) {
			return true, nil
		}
	}
	return false, nil
}

// ParseMountInfo parses mountinfo pointing to path and returns a map
// of parent mount points with associated child mount points
func ParseMountInfo(path string) (map[string][]string, error) {
	mp := make(map[string][]string)
	mountlist := make(map[string][]string)

	p, err := os.Open(path)
	if err != nil {
		return mp, fmt.Errorf("can't open %s: %s", path, err)
	}
	defer p.Close()

	scanner := bufio.NewScanner(p)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		mountlist[fields[0]] = fields
	}
	for k := range mountlist {
		if i, ok := mountlist[mountlist[k][1]]; ok {
			if mountlist[k][4] != i[4] {
				mp[i[4]] = append(mp[i[4]], mountlist[k][4])
			}
		}
	}
	return mp, nil
}

// ParentMount parses mountinfo and return the path of parent
// mount point for which the provided path is mounted in
func ParentMount(path string) (string, error) {
	var mountPoints []string
	parent := "/"

	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return parent, err
	}

	p, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return parent, fmt.Errorf("can't open /proc/self/mountinfo: %s", err)
	}
	defer p.Close()

	scanner := bufio.NewScanner(p)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		mountPoints = append(mountPoints, fields[4])
	}

	for resolved != "/" {
		for _, point := range mountPoints {
			if point == resolved {
				return point, nil
			}
		}
		resolved = filepath.Dir(resolved)
	}

	return parent, nil
}

// ExtractPid returns a pid extracted from path of type "/proc/1"
func ExtractPid(path string) (pid uint, err error) {
	n, err := fmt.Sscanf(path, "/proc/%d", &pid)
	if n != 1 {
		return 0, fmt.Errorf("can't extract PID from %s: %s", path, err)
	}
	return
}

// CountChilds returns the number of child processes for a given process id
func CountChilds(pid int) (int, error) {
	childs := 0

	parentProc := fmt.Sprintf("/proc/%d", pid)
	if _, err := os.Stat(parentProc); os.IsNotExist(err) {
		return 0, fmt.Errorf("pid %d doesn't exists", pid)
	}

	parentLine := fmt.Sprintf("PPid:\t%d", pid)
	pattern := filepath.Join("/proc", "[0-9]*")

	matches, _ := filepath.Glob(pattern)
	for _, path := range matches {
		r, err := os.Open(filepath.Join(path, "status"))
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			if scanner.Text() == parentLine {
				childs++
				break
			}
		}
		r.Close()
	}
	return childs, nil
}

// ReadIDMap reads uid_map or gid_map and returns both container ID
// and host ID
func ReadIDMap(path string) (uint32, uint32, error) {
	r, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer r.Close()

	scanner := bufio.NewScanner(r)
	scanner.Scan()
	fields := strings.Fields(scanner.Text())

	containerID, err := strconv.ParseUint(fields[0], 10, 32)
	if err != nil {
		return 0, 0, err
	}
	hostID, err := strconv.ParseUint(fields[1], 10, 32)
	if err != nil {
		return 0, 0, err
	}

	return uint32(containerID), uint32(hostID), nil
}

// SetOOMScoreAdj sets OOM score for process with pid
func SetOOMScoreAdj(pid int, score *int) error {
	if score != nil {
		path := fmt.Sprintf("/proc/%d/oom_score_adj", pid)

		f, err := os.OpenFile(path, os.O_WRONLY, 0)
		if err != nil {
			return fmt.Errorf("failed to open oom_score_adj: %s", err)
		}
		if _, err := fmt.Fprintf(f, "%d", *score); err != nil {
			return fmt.Errorf("failed to set oom_score_adj: %s", err)
		}

		f.Close()
	}
	return nil
}

// HasNamespace checks if host namespace and container namespace
// are different.
func HasNamespace(pid int, nstype string) (bool, error) {
	var st1 syscall.Stat_t
	var st2 syscall.Stat_t

	has := false

	processOne := fmt.Sprintf("/proc/%d/ns/%s", pid, nstype)
	processTwo := fmt.Sprintf("/proc/self/ns/%s", nstype)

	if err := syscall.Stat(processOne, &st1); err != nil {
		if os.IsNotExist(err) {
			return has, nil
		}
		return has, err
	}
	if err := syscall.Stat(processTwo, &st2); err != nil {
		if os.IsNotExist(err) {
			return has, nil
		}
		return has, err
	}

	if st1.Ino != st2.Ino {
		has = true
	}

	return has, nil
}
