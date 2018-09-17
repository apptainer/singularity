// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package nvidiautils

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// GetNvidiaBindPath returns a string array consisting of filepaths of nvidia
// related files to be added to the BindPaths
func GetNvidiaBindPath(abspath string) []string {
	var strArray []string
	var searchArray []string
	var commentID = regexp.MustCompile(`#`)
	var soID = regexp.MustCompile(".so")

	// use nvidia-container-cli (if present)
	command, err := exec.LookPath("nvidia-container-cli")
	if err == nil {
		cmd := exec.Command(command, "list", "--binaries", "--ipcs", "--libraries")
		out, err := cmd.Output()
		if err == nil {

			for _, line := range strings.Split(string(out), "\n") {
				val := soID.FindString(line) // this will disallow binaries (non .so files)
				if val != "" {               // contains .so
					if line != "" {
						strArray = append(strArray, line)
					}
				} else {
					strArray = append(strArray, line) // binary
				}
			}
		}
	}

	cliEntries := strings.Join(strArray, " ") // save away for later comparison check (disallow duplicates)

	// grab the entries in nvliblist.conf file
	// use ldconfig to pattern match from ld.so.cache
	newpath, err := filepath.Glob(abspath + "/singularity/nvliblist.conf")
	if err == nil {
		for _, filename := range newpath {

			file, err := os.Open(filename)
			if err == nil {
				scanner := bufio.NewScanner(file)
				for scanner.Scan() {
					line := scanner.Text()
					val := commentID.FindString(line)
					if val == "" && line != "" {
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
	command, err = exec.LookPath("ldconfig")
	if err == nil {
		cmd := exec.Command(command, "-p")
		out, err := cmd.Output()
		if err == nil {
			lastadd := ""
			for _, line := range strings.Split(strings.TrimSuffix(string(out), "\n"), "\n") {
				if line != "" {

					for _, fileline := range strings.Split(strings.TrimSuffix(searchString, "\n"), "\n") {
						if fileline != "" {

							line2 := strings.SplitN(line, "=> ", 2)
							if len(line2) > 1 {

								if !strings.Contains(cliEntries, line2[1]) { // skip if nvidia-container-cli found it

									if strings.Contains(line2[1], fileline) && fileline != lastadd { // add if not duplicate
										strArray = append(strArray, line2[1])
										lastadd = fileline
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return strArray
}
