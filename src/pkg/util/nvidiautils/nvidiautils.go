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

// generate bind list using nvidia-container-cli
func nvidiaContainerCli() ([]string, []string, error) {
	var strArray []string
	var bindArray []string

	// use nvidia-container-cli (if present)
	command, err := exec.LookPath("nvidia-container-cli")
	if err != nil {
		return nil, nil, fmt.Errorf("no nvidia-container-cli present: %v\n", err)
	}

	// process the binaries first
	cmd := exec.Command(command, "list", "--binaries")
	out, err := cmd.Output()
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to execute nvidia-container-cli: %v\n", err)
	}

	for _, line := range strings.Split(string(out), "\n") {
		if line != "" {
			// add this in to the bind list here so we don't need to special case it
			// when the libraries are processed later in GetNvidiaBindPath
			// (i.e. thiese will never show up in ldconfig output, hence they would not be
			// added in later without adding a lot of special case code)
			bindString := line + ":" + line
			bindArray = append(bindArray, bindString)
		}
	}

	cmd = exec.Command(command, "list", "--ipcs", "--libraries")
	out, err = cmd.Output()
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to execute nvidia-container-cli: %v\n", err)
	}

	for _, line := range strings.Split(string(out), "\n") {
		if line != "" {

			// strip off the filepath info as we are using
			// the filepath returned by "ldconfig -p"
			filePaths := strings.Split(line, "/")
			fileName := ""
			if len(filePaths) > 1 {
				fileName = filePaths[len(filePaths)-1]
			}

			if strings.Contains(fileName, ".so") {
				strArray = append(strArray, fileName) // add entry to list to be bound
				// strip off .xxx.xx prefix and add so and so.1 entries as well
				newentry := strings.SplitAfter(fileName, ".so")
				strArray = append(strArray, newentry[0])      // add prefix (filepath.so)
				strArray = append(strArray, newentry[0]+".1") // add prefix +".1" (filepath.so.1)
			}
		}
	}
	return strArray, bindArray, nil
}

// generate bind list using contents of nvliblist.conf
func nvidiaLiblist(abspath string) ([]string, error) {
	var strArray []string
	var commentID = regexp.MustCompile(`^#`)

	// grab the entries in nvliblist.conf file
	file, err := os.Open(abspath + "/nvliblist.conf")
	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}

	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		val := commentID.FindString(line)
		if val == "" && line != "" {
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
	strArray, bindArray, err := nvidiaContainerCli()
	if err != nil {
		sylog.Warningf("nvidiaContainercli returned: %v", err)

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
	for _, ldconfigOutputline := range strings.Split(strings.TrimSuffix(string(out), "\n"), "\n") {
		if ldconfigOutputline != "" {
			for _, nvidiaFileName := range strArray {

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

					if ldconfigFileName == nvidiaFileName && ldconfigFileName != lastadd { // add if not duplicate
						// this is binding the actual name found above...
						bindString := ldconfigOutputSplitline[1] + ":/.singularity.d/libs/" + ldconfigFileName
						bindArray = append(bindArray, bindString)
						lastadd = ldconfigFileName
					}
				}
			}
		}
	}

	return bindArray, nil
}
