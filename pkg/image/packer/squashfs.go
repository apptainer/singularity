// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package packer

import (
	"bytes"
	"fmt"
	"os/exec"
)

// Squashfs represents a squashfs packer
type Squashfs struct {
	MksquashfsPath string
}

// NewSquashfs initializes and returns a Squashfs packer instance
func NewSquashfs() *Squashfs {
	s := &Squashfs{}
	s.MksquashfsPath, _ = exec.LookPath("mksquashfs")
	return s
}

// HasMksquashfs returns if mksquashfs binary has set or not
func (s *Squashfs) HasMksquashfs() bool {
	return s.MksquashfsPath != ""
}

func (s *Squashfs) create(files []string, dest string, opts []string) error {
	var stderr bytes.Buffer

	if !s.HasMksquashfs() {
		return fmt.Errorf("could not create squashfs, mksquashfs not found")
	}

	// mksquashfs takes args of the form: source1 source2 ... destination [options]
	args := files
	args = append(args, dest)
	args = append(args, opts...)

	cmd := exec.Command(s.MksquashfsPath, args...)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("create command failed: %v: %s", err, stderr.String())
	}
	return nil
}

// Create makes a squashfs filesystem from a list of source files/directories to a
// destination file
func (s *Squashfs) Create(src []string, dest string, opts []string) error {
	return s.create(src, dest, opts)
}
