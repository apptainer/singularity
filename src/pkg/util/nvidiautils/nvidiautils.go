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
	"strings"

	"github.com/sylabs/singularity/src/pkg/sylog"
)

// generate bind list using nvidia-container-cli
func nvidiaContainerCli() ([]string, error) {
	var strArray []string

	// use nvidia-container-cli (if present)
	command, err := exec.LookPath("nvidia-container-cli")
	if err != nil {
		return nil, fmt.Errorf("no nvidia-container-cli present: %v", err)
	}

	cmd := exec.Command(command, "list", "--binaries", "--ipcs", "--libraries")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("Unable to execute nvidia-container-cli: %v", err)
	}

	for _, line := range strings.Split(string(out), "\n") {
		if line != "" {

			// if this is a library, then add a .so entry as well
			if strings.Contains(line, ".so") {
				fileName := filepath.Base(line)
				strArray = append(strArray, fileName) // add entry to list to be bound

				// strip off .xxx.xx prefix and add so entry as well
				newentry := strings.SplitAfter(fileName, ".so")
				strArray = append(strArray, newentry[0]) // add prefix (filepath.so)
			} else {
				// Assume we're a binary and need the full path
				strArray = append(strArray, line)
			}
		}
	}
	return strArray, nil
}

// generate bind list using contents of nvliblist.conf
func nvidiaLiblist(abspath string) ([]string, error) {
	var strArray []string

	// grab the entries in nvliblist.conf file
	file, err := os.Open(abspath + "/nvliblist.conf")
	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}

	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "#") && line != "" {
			strArray = append(strArray, line)
		}
	}
	return strArray, nil
}

// GetNvidiaBindPath returns a string array consisting of filepaths of nvidia
// related files to be added to the BindPaths
func GetNvidiaBindPath(abspath string) ([]string, error) {
	var strArray []string
	var bindArray []string

	// use nvidia-container-cli if presenet
	strArray, err := nvidiaContainerCli()
	if err != nil {
		sylog.Verbosef("nvidiaContainercli returned: %v", err)
		sylog.Verbosef("Falling back to nvliblist.conf")

		// nvidia-container-cli not present or errored out
		// fallback is to use nvliblist.conf
		strArray, err = nvidiaLiblist(abspath)
		if err != nil {
			sylog.Warningf("nvidiaLiblist returned: %v", err)
			return nil, err
		}
	}

	// walk thru the ldconfig output and add entries which contain the filenames
	// returned by nvidia-container-cli OR the nvliblist.conf file contents
	command, err := exec.LookPath("ldconfig")
	if err != nil {
		sylog.Warningf("ldconfig not found: %v", err)
		return nil, nil
	}

	cmd := exec.Command(command, "-p")
	out, err := cmd.Output()
	if err != nil {
		sylog.Warningf("ldconfig execution error: %v", err)
		return nil, nil
	}

	lastadd := ""
	for _, nvidiaFileName := range strArray {

		// if the file contins a ".so", treat it as a library
		if strings.Contains(nvidiaFileName, ".so") {

			for _, ldconfigOutputline := range strings.Split(strings.TrimSuffix(string(out), "\n"), "\n") {
				if ldconfigOutputline != "" {

					// sample ldconfig -p output (ldconfigOutputline)
					// 	libnvidia-ml.so.1 (libc6,x86-64) => /usr/lib64/nvidia/libnvidia-ml.so.1
					//	libnvidia-ml.so (libc6,x86-64) => /usr/lib64/nvidia/libnvidia-ml.so

					ldconfigOutputSplitline := strings.SplitN(ldconfigOutputline, "=> ", 2)
					if len(ldconfigOutputSplitline) > 1 {

						// ldconfigOutputSplitline[0] is the "libnvidia-ml.so[.1] (libc6,x86-64)"" (from the above example)
						// ldconfigOutputSplitline[1] is the "/usr/lib64/nvidia/libnvidia-ml.so[.1]" (from the above example)
						// these next 2 lines extract the "libnvdia-ml.so[.1]" (from the above example)

						// ldconfigFileName is "libnvidia-ml.so[.1]" (from the above example)
						ldconfigFileNames := strings.Split(ldconfigOutputSplitline[0], " ")
						ldconfigFileName := strings.TrimSpace(string(ldconfigFileNames[0]))

						if strings.HasPrefix(ldconfigFileName, nvidiaFileName) && ldconfigFileName != lastadd { // add if not duplicate
							// this is binding the actual name found above...
							bindString := ldconfigOutputSplitline[1] + ":/.singularity.d/libs/" + ldconfigFileName
							bindArray = append(bindArray, bindString)
							lastadd = ldconfigFileName
						}
					}

				}
			}
		} else {
			// treat the file as a binary file - add it to the bind list
			// no need to check the ldconfig output
			bindString := nvidiaFileName + ":" + nvidiaFileName
			bindArray = append(bindArray, bindString)
		}
	}

	return bindArray, nil
}
