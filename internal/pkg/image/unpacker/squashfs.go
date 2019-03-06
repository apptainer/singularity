// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package unpacker

import (
	"fmt"
	"io"
	"os/exec"
)

// Squashfs represents a squashfs unpacker
type Squashfs struct {
	UnsquashfsPath string
}

// NewSquashfs initializes and returns a Squahfs unpacker instance
func NewSquashfs() *Squashfs {
	s := &Squashfs{}
	s.UnsquashfsPath, _ = exec.LookPath("unsquashfs")
	return s
}

// HasUnsquashfs returns if unsquashfs binary has been found or not
func (s *Squashfs) HasUnsquashfs() bool {
	return s.UnsquashfsPath != ""
}

func (s *Squashfs) extract(files []string, reader io.Reader, dest string) error {
	if !s.HasUnsquashfs() {
		return fmt.Errorf("could not extract squashfs data, unsquashfs not found")
	}
	args := []string{"-f", "-d", dest, "/proc/self/fd/0"}
	for _, f := range files {
		args = append(args, f)
	}
	cmd := exec.Command(s.UnsquashfsPath, args...)
	cmd.Stdin = reader
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("extract command failed: %s", err)
	}
	return nil
}

// ExtractAll extracts a squashfs filesystem read from reader to a
// destination directory
func (s *Squashfs) ExtractAll(reader io.Reader, dest string) error {
	return s.extract([]string{}, reader, dest)
}

// ExtractFiles extracts provided files from a squashfs filesystem
// read from reader to a destination directory
func (s *Squashfs) ExtractFiles(files []string, reader io.Reader, dest string) error {
	if len(files) == 0 {
		return fmt.Errorf("no files to extract")
	}
	return s.extract(files, reader, dest)
}
