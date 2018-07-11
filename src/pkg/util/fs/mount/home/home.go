// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package home

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/fs"
	"github.com/singularityware/singularity/src/pkg/util/fs/mount"
	"github.com/singularityware/singularity/src/pkg/util/user"
)

// EnvVar returns the environment variable that is to be set inside
// the container referring to the home directory. Return is in form
// []string{"HOME", $PATH}.
func EnvVar(spec string) ([]string, error) {
	split, err := specSplit(spec)
	if err != nil {
		return nil, err
	}

	return []string{"HOME", split[1]}, nil
}

// AddDefault adds the home directory stored in the users Passwd
// information to the mount list. Returns a string containing the
// environment variable for inside the container
func AddDefault(p *mount.Points, rootfs string, u *user.Passwd) error {
	return add(p, rootfs, u.Dir, u.Dir)
}

// AddCustom adds the home directory specified by the -H/--home
// option at runtime to the mount list. Returns a string containing
// the environment variable for inside the container
func AddCustom(p *mount.Points, rootfs, spec string) error {
	split, err := specSplit(spec)
	if err != nil {
		return err
	}

	return add(p, rootfs, split[0], split[1])
}

// add adds the src:dest pair and returns the env var to the exported functions
func add(p *mount.Points, rootfs, src, dest string) error {
	uid := os.Getuid()

	// Ensure that dest was given as an absolute path
	if !filepath.IsAbs(dest) {
		return fmt.Errorf("dest must be absolute")
	}

	// Ensure that src is absolute, use filepath.Abs to resolve if necessary
	src, err := filepath.Abs(src)
	if err != nil {
		return err
	}

	// Ensure that src is a directory
	if !fs.IsDir(src) {
		return fmt.Errorf("src is not a dir")
	}

	// Ensure that user owns src directory
	if !fs.IsOwner(src, uint32(uid)) {
		return fmt.Errorf("user %v does not own src %v", uid, src)
	}

	srcRoot := fs.RootDir(src)
	destRoot := fs.RootDir(dest)

	sylog.Debugf("Adding home directory mount: %v => %v\n", srcRoot, destRoot)
	p.AddBind(mount.HomeTag, srcRoot, filepath.Join(rootfs, destRoot), syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_REC)

	return nil
}

func specSplit(spec string) ([]string, error) {
	split := strings.Split(spec, ":")

	if len(split) < 1 || len(split) > 2 {
		return nil, fmt.Errorf("spec is not in form src:dest or src")
	}

	if len(split) == 1 {
		split = append(split, split[0])
	}

	if len(split) != 2 {
		return nil, fmt.Errorf("something went horribly wrong")
	}

	return split, nil
}
