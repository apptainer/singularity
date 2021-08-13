// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
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

	"golang.org/x/sys/unix"
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

// GetMountPointMap parses mountinfo pointing to path and returns
// a map of parent mount points with associated child mount points.
func GetMountPointMap(path string) (map[string][]string, error) {
	mp := make(map[string][]string)
	entries := make(map[string]MountInfoEntry)

	p, err := os.Open(path)
	if err != nil {
		return mp, fmt.Errorf("can't open %s: %s", path, err)
	}
	defer p.Close()

	scanner := bufio.NewScanner(p)
	for scanner.Scan() {
		entry := parseMountInfoLine(scanner.Text())
		entries[entry.ID] = entry
	}

	for e := range entries {
		parentID := entries[e].ParentID
		if i, ok := entries[parentID]; ok {
			point := i.Point
			if entries[e].Point != point {
				mp[point] = append(mp[point], entries[e].Point)
			}
		}
	}
	return mp, nil
}

// MountInfoEntry contains parsed fields of a mountinfo line.
type MountInfoEntry struct {
	ID           string
	ParentID     string
	Dev          string
	Root         string
	Point        string
	Options      []string
	Fields       string
	FSType       string
	Source       string
	SuperOptions []string
}

// parseMountInfoLine parses a mountinfo line and returns
// a MountInfoEntry containing parsed fields associated
// to the line.
func parseMountInfoLine(line string) MountInfoEntry {
	fields := strings.Split(line, " ")
	entry := MountInfoEntry{}

	// ID field
	entry.ID = fields[0]
	// convert Parent ID field
	entry.ParentID = fields[1]
	// convert major:minor field
	entry.Dev = fields[2]
	// root field
	entry.Root = fields[3]
	// mount point field
	entry.Point = fields[4]
	// mount options field
	entry.Options = strings.Split(fields[5], ",")
	// optional fields field
	index := 6
	for ; fields[index] != "-"; index++ {
		entry.Fields += " " + fields[index]
	}
	entry.Fields = strings.TrimSpace(entry.Fields)

	// filesystem type field
	entry.FSType = fields[index+1]
	// mount source field
	entry.Source = fields[index+2]
	// super block options field
	entry.SuperOptions = strings.Split(fields[index+3], ",")

	// major/minor number reported in mountinfo may be wrong
	// for btrfs/overlay filesystems as it uses virtual
	// device numbers, st_dev from stat will return numbers
	// different from those shown in mountinfo, to fix that
	// we need to get major/minor directly from a stat call
	// on the corresponding mount point
	if entry.FSType == "btrfs" || entry.FSType == "overlay" || entry.FSType == "ceph" {
		fi, err := os.Stat(entry.Point)
		if err == nil {
			st := fi.Sys().(*syscall.Stat_t)
			// cast to uint64 as st.Dev is uint32 on MIPS
			entry.Dev = fmt.Sprintf("%d:%d", unix.Major(uint64(st.Dev)), unix.Minor(uint64(st.Dev)))
		}
	}

	return entry
}

// GetMountInfoEntry parses a mountinfo file and returns all
// parsed entries as an array of MountInfoEntry.
func GetMountInfoEntry(path string) ([]MountInfoEntry, error) {
	p, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("can't open %s: %s", path, err)
	}
	defer p.Close()

	entries := make([]MountInfoEntry, 0)
	scanner := bufio.NewScanner(p)
	for scanner.Scan() {
		entry := parseMountInfoLine(scanner.Text())
		entries = append(entries, entry)
	}

	return entries, nil
}

// FindParentMountEntry finds the parent mount point entry associated
// to the provided path among the entry list provided in argument.
func FindParentMountEntry(path string, entries []MountInfoEntry) (*MountInfoEntry, error) {
	p, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, fmt.Errorf("while resolving path %s: %s", path, err)
	}

	fi, err := os.Stat(p)
	if err != nil {
		return nil, fmt.Errorf("while getting stat for %s: %s", path, err)
	}
	st := fi.Sys().(*syscall.Stat_t)
	// cast to uint64 as st.Dev is uint32 on MIPS
	dev := fmt.Sprintf("%d:%d", unix.Major(uint64(st.Dev)), unix.Minor(uint64(st.Dev)))

	var entry *MountInfoEntry
	matchLen := 0

	for i, e := range entries {
		// find the longest mount point for the provided path
		if e.Dev == dev && strings.HasPrefix(p, e.Point) {
			l := len(e.Point)
			if l > matchLen {
				matchLen = l
				entry = &entries[i]
			}
		}
	}

	if entry == nil {
		return nil, fmt.Errorf("no parent mount point found")
	}

	return entry, nil
}

// ParentMount parses mountinfo and returns the path of parent
// mount point for which the provided path is mounted in.
func ParentMount(path string) (string, error) {
	entries, err := GetMountInfoEntry("/proc/self/mountinfo")
	if err != nil {
		return "", fmt.Errorf("while parsing %s: %s", path, err)
	}

	entry, err := FindParentMountEntry(path, entries)
	if err != nil {
		return "", err
	}
	return entry.Point, nil
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
	if !scanner.Scan() {
		return 0, 0, fmt.Errorf("scanner error: %s", scanner.Err())
	}
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

// Getppid returns the parent process ID for the corresponding
// process ID passed in parameter.
func Getppid(pid int) (int, error) {
	status := fmt.Sprintf("/proc/%d/status", pid)
	p, err := os.Open(status)
	if err != nil {
		return -1, fmt.Errorf("could not open %s: %s", status, err)
	}
	defer p.Close()

	scanner := bufio.NewScanner(p)
	for scanner.Scan() {
		ppid := -1
		n, _ := fmt.Sscanf(scanner.Text(), "PPid:\t%d", &ppid)
		if n == 1 && ppid > 0 {
			return ppid, nil
		}
	}

	return -1, fmt.Errorf("no parent process ID found")
}
