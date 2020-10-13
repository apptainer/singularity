// Copyright (c) 2020, Control Command Inc. All rights reserved.
// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package unpacker

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sylabs/singularity/pkg/sylog"
)

const (
	stdinFile = "/proc/self/fd/0"
)

var cmdFunc func(unsquashfs string, dest string, filename string, rootless bool) (*exec.Cmd, error)

// unsquashfsCmd is the command instance for executing unsquashfs command
// in a non sandboxed environment when this package is used for unit tests.
func unsquashfsCmd(unsquashfs string, dest string, filename string, rootless bool) (*exec.Cmd, error) {
	args := make([]string, 0)
	if rootless {
		args = append(args, "-user-xattrs")
	}
	// remove the destination directory if any, if the directory is
	// not empty (typically during image build), the unsafe option -f is
	// set, this is unfortunately required by image build
	if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
		if !os.IsExist(err) {
			return nil, fmt.Errorf("failed to remove %s: %s", dest, err)
		}
		// unsafe mode
		args = append(args, "-f")
	}
	args = append(args, "-d", dest, filename)
	return exec.Command(unsquashfs, args...), nil
}

// Squashfs represents a squashfs unpacker.
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

	// pipe over stdin by default
	stdin := true
	filename := stdinFile

	if _, ok := reader.(*os.File); !ok {
		// use the destination parent directory to store the
		// temporary archive
		tmpdir := filepath.Dir(dest)

		// unsquashfs doesn't support to send file content over
		// a stdin pipe since it use lseek for every read it does
		tmp, err := ioutil.TempFile(tmpdir, "archive-")
		if err != nil {
			return fmt.Errorf("failed to create staging file: %s", err)
		}
		filename = tmp.Name()
		stdin = false
		defer os.Remove(filename)

		if _, err := io.Copy(tmp, reader); err != nil {
			return fmt.Errorf("failed to copy content in staging file: %s", err)
		}
		if err := tmp.Close(); err != nil {
			return fmt.Errorf("failed to close staging file: %s", err)
		}
	}

	// If we are running as non-root, first try with `-user-xattrs` so we won't fail trying
	// to set system xatts. This isn't supported on unsquashfs 4.0 in RHEL6 so we
	//  have to fall back to not using that option on failure.
	if os.Geteuid() != 0 {
		sylog.Debugf("Rootless extraction. Trying -user-xattrs for unsquashfs")

		cmd, err := cmdFunc(s.UnsquashfsPath, dest, filename, true)
		if err != nil {
			return fmt.Errorf("command error: %s", err)
		}
		cmd.Args = append(cmd.Args, files...)
		if stdin {
			cmd.Stdin = reader
		}

		o, err := cmd.CombinedOutput()
		if err == nil {
			return nil
		}

		// Invalid options give output...
		// SYNTAX: unsquashfs [options] filesystem [directories or files to extract]
		if bytes.Contains(o, []byte("SYNTAX")) {
			sylog.Warningf("unsquashfs does not support -user-xattrs. Images with system xattrs may fail to extract")
		} else {
			// A different error is fatal
			return fmt.Errorf("extract command failed: %s: %s", string(o), err)
		}
	}

	cmd, err := cmdFunc(s.UnsquashfsPath, dest, filename, false)
	if err != nil {
		return fmt.Errorf("command error: %s", err)
	}
	cmd.Args = append(cmd.Args, files...)
	if stdin {
		cmd.Stdin = reader
	}
	if o, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("extract command failed: %s: %s", string(o), err)
	}
	return nil
}

// ExtractAll extracts a squashfs filesystem read from reader to a
// destination directory.
func (s *Squashfs) ExtractAll(reader io.Reader, dest string) error {
	return s.extract(nil, reader, dest)
}

// ExtractFiles extracts provided files from a squashfs filesystem
// read from reader to a destination directory.
func (s *Squashfs) ExtractFiles(files []string, reader io.Reader, dest string) error {
	if len(files) == 0 {
		return fmt.Errorf("no files to extract")
	}
	return s.extract(files, reader, dest)
}
