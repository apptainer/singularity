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
	"regexp"
	"strings"

	"github.com/singularityware/singularity/src/pkg/sylog"
)

func nvidiaContainerCli() ([]string, []string, error) {
	var strArray []string
	var bindArray []string
	var soID = regexp.MustCompile(".so")

	// use nvidia-container-cli (if present)
	command, err := exec.LookPath("nvidia-container-cli")
	if err != nil {
		return nil, nil, fmt.Errorf("no nvidia-container-cli present: %v\n", err)
	}
	cmd := exec.Command(command, "list", "--binaries", "--ipcs", "--libraries")
	out, err := cmd.Output()
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to execute nvidia-container-cli: %v\n", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		val := soID.FindString(line) // this will disallow binaries (non .so files)
		if val != "" {               // contains .so
			if line != "" {
				// extract the filename from the path
				fileNames := strings.SplitAfter(line, "/")
				fileName := fileNames[len(fileNames)-1]

				testString := line + ":/.singularity.d/libs/" + fileName
				bindArray = append(bindArray, testString)
				strArray = append(strArray, fileName)
			}
		} else { // binary executable
			bindArray = append(bindArray, line)
			strArray = append(strArray, line)
		}
	}
	return bindArray, strArray, nil
}

// GetNvidiaBindPath returns a string array consisting of filepaths of nvidia
// related files to be added to the BindPaths
func GetNvidiaBindPath(abspath string) ([]string, error) {
	var strArray []string
	var bindArray []string
	var searchArray []string
	var commentID = regexp.MustCompile(`^#`)

	bindArray, strArray, err := nvidiaContainerCli()
	if err != nil {
		sylog.Warningf("nvidia-container-cli returned: %v", err)
	}
	cliEntries := strings.Join(strArray, " ") // save away for later comparison check (disallow duplicates)

	// grab the entries in nvliblist.conf file
	// use ldconfig to pattern match from ld.so.cache
	file, err := os.Open(abspath + "/nvliblist.conf")
	if err != nil {
		sylog.Warningf("failed to open file %v:", err)
	} else {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			val := commentID.FindString(line)
			if val == "" && line != "" {
				searchArray = append(searchArray, line)
			}
		}

	}

	// walk thru the ldconfig output and add entries which contain the filenames located in
	// the nvliblist.conf file (ldconfig filenames are full filepaths)
	var searchFileName string
	command, err := exec.LookPath("ldconfig")
	if err != nil {
		sylog.Warningf("ldconfig not found: %v", err)
		return bindArray, nil
	}
	cmd := exec.Command(command, "-p")
	out, err := cmd.Output()
	if err != nil {
		sylog.Warningf("ldconfig execution error: %v", err)
		return bindArray, nil
	}
	lastadd := ""
	for _, ldconfigOutputline := range strings.Split(strings.TrimSuffix(string(out), "\n"), "\n") {
		if ldconfigOutputline != "" {
			for _, nvidiaConfFileline := range searchArray {

				// sample ldconfig -p output (ldconfigOutputline)
				// 	libnvidia-ml.so.1 (libc6,x86-64) => /usr/lib64/nvidia/libnvidia-ml.so.1
				//	libnvidia-ml.so (libc6,x86-64) => /usr/lib64/nvidia/libnvidia-ml.so

				ldconfigOutputSplitline := strings.SplitN(ldconfigOutputline, "=> ", 2)
				if len(ldconfigOutputSplitline) > 1 {

					// ldconfigOutputSplitline[0] is the "libnvidia-ml.so[.1] (libc6,x86-64)"" (from the above example)
					// ldconfigOutputSplitline[1] is the "/usr/lib64/nvidia/libnvidia-ml.so[.1]" (from the above example)

					if !strings.Contains(cliEntries, ldconfigOutputSplitline[1]) { // skip if nvidia-container-cli found it

						// these 2 lines extract the "libnvdia-ml.so[.1]" (from the example above) - fileName
						ldconfigFileNames := strings.Split(ldconfigOutputSplitline[0], " ")
						ldconfigFileName := strings.TrimSpace(string(ldconfigFileNames[0]))

						// this code block adds in foo.so.1 if there is a foo.so found in the config file
						if strings.HasSuffix(ldconfigFileName, ".1") {
							// remove the .1 from the search param (but will bind the actual name)
							searchFileName = strings.TrimSuffix(ldconfigFileName, ".1")
						} else {
							searchFileName = ldconfigFileName
						}

						if searchFileName == nvidiaConfFileline {
							if ldconfigFileName != lastadd { // add if not duplicate
								// this is binding the actual name found above...
								bindString := ldconfigOutputSplitline[1] + ":/.singularity.d/libs/" + ldconfigFileName
								bindArray = append(bindArray, bindString)
								lastadd = ldconfigFileName
							}
						}
					}
				}
			}
		}
	}

	return bindArray, nil
}
