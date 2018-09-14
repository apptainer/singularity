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
			fmt.Printf("nvidia-container-cli raw Array = %v\n", string(out))
			for _, line := range strings.Split(string(out), "\n") {
				val := soID.FindString(line) // this will disallow binaries (non .so files)
				if val != "" {               // contains .so
					if line != "" {
						//if strings.HasSuffix(line, ".so") {	// this kill files like foo.so.396.44
						strArray = append(strArray, line)
						//}
					}
				} else {
					strArray = append(strArray, line)
				}
			}
		}
	}
	// fmt.Printf("nvidia-container-cli strArray = %v\n", strArray)

	// grab the entries in nvliblist.conf file
	// use ldconfig to pattern match from ld.so.cache
	newpath, err := filepath.Glob(abspath + "/singularity/nvliblist.conf")
	if err == nil {
		for _, filename := range newpath {
			// open the file, strip comments, etc.
			// fmt.Println(filename)
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
	//sylog.Debugf("search array = %v\n", searchArray)
	// fmt.Printf("nvliblist.conf searchrArray = %v\n", searchArray)

	// walk thru the ldconfig output and add entries which contain the filenames located in
	// the nvliblist.conf file (ldconfig filenames are full filepaths)
	// NOTE: this is how it was implemented in 2.6
	command, err = exec.LookPath("ldconfig")
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
								//		//if strings.HasSuffix(line2[1], ".so") {
								fmt.Printf("adding filepath %v (contains conf file entry %v) to bind list \n", line2[1], fileline)
								strArray = append(strArray, line2[1])
								//		//}
							}
						}
						//if strings.Contains(line, fileline) {
						//	//if strings.HasSuffix(line2[1], ".so") {
						//	fmt.Printf("adding filepath %v (contains conf file entry %v) to bind list \n", line, fileline)
						//	strArray = append(strArray, fileline)
						//	//}
						//}
					}
				}
			}
		}
	}

	//fmt.Printf("searchArray = %v\n", searchArray)
	fmt.Printf("final strArray = %v\n", strArray)

	return strArray
	// return strings.Join(strArray, " ")
}
