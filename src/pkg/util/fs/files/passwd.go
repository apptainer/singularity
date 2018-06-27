// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package files

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/fs"
	"github.com/singularityware/singularity/src/pkg/util/user"
)

// Passwd creates a passwd template based on content of file provided in path,
// updates content with current user information and returns content
func Passwd(path string, home string) (content []byte, err error) {
	sylog.Verbosef("Checking for template passwd file: %s\n", path)
	if fs.IsFile(path) == false {
		return content, fmt.Errorf("passwd file doesn't exist in container, not updating")
	}

	sylog.Verbosef("Creating passwd content\n")
	passwdFile, err := os.Open(path)
	if err != nil {
		return content, fmt.Errorf("failed to open passwd file in container: %s", err)
	}
	defer passwdFile.Close()

	content, err = ioutil.ReadAll(passwdFile)
	if err != nil {
		return content, fmt.Errorf("failed to read passwd file content in container: %s", err)
	}

	pwInfo, err := user.GetPwUID(uint32(os.Getuid()))
	if err != nil {
		return content, err
	}

	homeDir := pwInfo.Dir
	if home != "" {
		homeDir = home
	}
	userInfo := fmt.Sprintf("%s:x:%d:%d:%s:%s:%s\n", pwInfo.Name, pwInfo.UID, pwInfo.GID, pwInfo.Gecos, homeDir, pwInfo.Shell)

	if content[len(content)-1] != '\n' {
		content = append(content, byte('\n'))
	}

	sylog.Verbosef("Creating template passwd file and appending user data: %s\n", path)
	content = append(content, []byte(userInfo)...)

	return content, nil
}
