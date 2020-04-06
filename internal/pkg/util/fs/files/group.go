// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package files

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/internal/pkg/util/user"
	"github.com/sylabs/singularity/pkg/sylog"
)

// Group creates a group template based on content of file provided in path,
// updates content with current user information and returns content
func Group(path string, uid int, gids []int) (content []byte, err error) {
	duplicate := false
	var groups []int

	sylog.Verbosef("Checking for template group file: %s\n", path)
	if !fs.IsFile(path) {
		return content, fmt.Errorf("group file doesn't exist in container, not updating")
	}

	sylog.Verbosef("Creating group content\n")
	groupFile, err := os.Open(path)
	if err != nil {
		return content, fmt.Errorf("failed to open group file in container: %s", err)
	}
	defer groupFile.Close()

	pwInfo, err := user.GetPwUID(uint32(uid))
	if err != nil || pwInfo == nil {
		return content, err
	}
	if len(gids) == 0 {
		grInfo, err := user.GetGrGID(pwInfo.GID)
		if err != nil || grInfo == nil {
			return content, err
		}
		groups, err = os.Getgroups()
		if err != nil {
			return content, err
		}
	} else {
		groups = gids
	}
	for _, gid := range groups {
		if gid == int(pwInfo.GID) {
			duplicate = true
			break
		}
	}
	if !duplicate {
		if len(gids) == 0 {
			groups = append(groups, int(pwInfo.GID))
		}
	}
	content, err = ioutil.ReadAll(groupFile)
	if err != nil {
		return content, fmt.Errorf("failed to read group file content in container: %s", err)
	}

	if len(content) > 0 && content[len(content)-1] != '\n' {
		content = append(content, '\n')
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
