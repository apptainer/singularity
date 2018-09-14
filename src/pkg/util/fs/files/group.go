// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
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

// Group creates a group template based on content of file provided in path,
// updates content with current user information and returns content
func Group(path string) (content []byte, err error) {
	duplicate := false

	sylog.Verbosef("Checking for template group file: %s\n", path)
	if fs.IsFile(path) == false {
		return content, fmt.Errorf("group file doesn't exist in container, not updating")
	}

	sylog.Verbosef("Creating group content\n")
	groupFile, err := os.Open(path)
	if err != nil {
		return content, fmt.Errorf("failed to open group file in container: %s", err)
	}
	defer groupFile.Close()

	pwInfo, err := user.GetPwUID(uint32(os.Getuid()))
	if err != nil || pwInfo == nil {
		return content, err
	}
	grInfo, err := user.GetGrGID(pwInfo.GID)
	if err != nil || grInfo == nil {
		return content, err
	}
	groups, err := os.Getgroups()
	if err != nil {
		return content, err
	}
	for _, gid := range groups {
		if gid == int(pwInfo.GID) {
			duplicate = true
			break
		}
	}
	if duplicate == false {
		groups = append(groups, int(pwInfo.GID))
	}
	content, err = ioutil.ReadAll(groupFile)
	if err != nil {
		return content, fmt.Errorf("failed to read group file content in container: %s", err)
	}

	if content[len(content)-1] != '\n' {
		content = append(content, byte('\n'))
	}

	for _, gid := range groups {
		grInfo, err := user.GetGrGID(uint32(gid))
		if err != nil || grInfo == nil {
			sylog.Verbosef("Skipping GID %d as group entry doesn't exist.\n", gid)
			continue
		}
		groupLine := fmt.Sprintf("%s:x:%d:%s\n", grInfo.Name, grInfo.GID, pwInfo.Name)
		content = append(content, []byte(groupLine)...)
	}
	return content, nil
}
