// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package files

import (
	"fmt"
	"io/ioutil"

	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/internal/pkg/util/user"
	"github.com/sylabs/singularity/pkg/sylog"
)

// Passwd creates a passwd template based on content of file provided in path,
// updates content with current user information and returns content.
func Passwd(path string, home string, uid int) (content []byte, err error) {
	sylog.Verbosef("Checking for template passwd file: %s", path)
	if !fs.IsFile(path) {
		return content, fmt.Errorf("passwd file doesn't exist in container, not updating")
	}

	sylog.Verbosef("Creating passwd content")
	content, err = ioutil.ReadFile(path)
	if err != nil {
		return content, fmt.Errorf("failed to read passwd file content in container: %s", err)
	}

	pwInfo, err := user.GetPwUID(uint32(uid))
	if err != nil {
		return content, err
	}

	homeDir := pwInfo.Dir
	if home != "" {
		homeDir = home
	}
	userInfo := fmt.Sprintf("%s:x:%d:%d:%s:%s:%s\n", pwInfo.Name, pwInfo.UID, pwInfo.GID, pwInfo.Gecos, homeDir, pwInfo.Shell)

	if len(content) > 0 && content[len(content)-1] != '\n' {
		content = append(content, '\n')
	}

	sylog.Verbosef("Creating template passwd file and appending user data: %s", path)
	content = append(content, []byte(userInfo)...)

	return content, nil
}
