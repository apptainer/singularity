// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package proc

import (
	"bufio"
	"fmt"
	"os"
	"strings"
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
		splitted := strings.Split(scanner.Text(), " ")
		mountlist[splitted[0]] = splitted
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
