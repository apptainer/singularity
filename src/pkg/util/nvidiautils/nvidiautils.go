// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package nvidiautils

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// GetNvidiaBindPath returns a string array consisting of filepaths of nvidia
// related files to be added to the BindPaths
func GetNvidiaBindPath(abspath string) string {
	var strArray []string
	var searchArray []string
	var commentID = regexp.MustCompile(`#`)

	// grab the entries in nvliblist.conf file
	// use ldconfig to pattern match from ld.so.cache
	newpath, err := filepath.Glob(abspath + "/singularity/nvliblist.conf")
	if err == nil {
		for _, filename := range newpath {
			// open the file, strip comments, etc.
			fmt.Println(filename)
			file, err := os.Open(filename)
			if err == nil {
				scanner := bufio.NewScanner(file)
				for scanner.Scan() {
					line := scanner.Text()
					val := commentID.FindString(line)
					if val == "" {
						searchArray = append(searchArray, line)
					}
				}
				file.Close()
			}
		}
	}
	searchString := strings.Join(searchArray, "\n")

	// walk thru the ldconfig output and add entries which contain the filenames located in
	// the nvliblist.conf file (ldconfig filenames are full filepaths)
	// NOTE: this is how it was implemented in 2.6
	command, err := exec.LookPath("ldconfig")
	if err == nil {
		cmd := exec.Command(command, "-p")
		out, err := cmd.Output()
		if err == nil {
			for _, line := range strings.Split(strings.TrimSuffix(string(out), "\n"), "\n") {
				for _, fileline := range strings.Split(strings.TrimSuffix(searchString, "\n"), "\n") {
					if fileline != "" {
						line2 := strings.SplitN(line, "=> ", 2)
						if len(line2) > 1 {
							if strings.Contains(line2[1], fileline) {
								strArray = append(strArray, line2[1])
							}
						}
					}
				}
			}
		}
	}

	// use nvidia-container-cli (if present)
	command, err = exec.LookPath("nvidia-container-cli")
	if err == nil {
		cmd := exec.Command(command, "list", "--binaries", "--ipcs", "--libraries")
		out, err := cmd.Output()
		if err == nil {
			for _, line := range strings.Split(strings.TrimSuffix(string(out), "\n"), "\n") {
				//val := soID.FindString(line)	// this will disallow binaries (non .so files)
				//if val != "" {
				if line != "" {
					strArray = append(strArray, line)
				}
				//}
			}
		}
	}

	return strings.Join(strArray, " ")
}
