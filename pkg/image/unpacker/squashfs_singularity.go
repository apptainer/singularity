// Copyright (c) 2020, Control Command Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build singularity_engine

package unpacker

import (
	"bytes"
	"debug/elf"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/pkg/sylog"
)

func init() {
	cmdFunc = unsquashfsSandboxCmd
}

// getLibraries returns the libraries required by the elf binary,
// the binary path must be absolute.
func getLibraries(binary string) ([]string, error) {
	libs := make([]string, 0)

	exe, err := elf.Open(binary)
	if err != nil {
		return nil, err
	}
	defer exe.Close()

	interp := ""

	// look for the interpreter
	for _, p := range exe.Progs {
		if p.Type != elf.PT_INTERP {
			continue
		}
		buf := make([]byte, 4096)
		n, err := p.ReadAt(buf, 0)
		if err != nil && err != io.EOF {
			return nil, err
		} else if n > cap(buf) {
			return nil, fmt.Errorf("buffer too small to store interpreter")
		}
		// trim null byte to avoid an execution failure with
		// an invalid argument error
		interp = string(bytes.Trim(buf, "\x00"))
	}

	// this is a static binary, nothing to do
	if interp == "" {
		return libs, nil
	}

	// run interpreter to list library dependencies for the
	// corresponding binary, eg:
	// /lib64/ld-linux-x86-64.so.2 --list <program>
	// /lib/ld-musl-x86_64.so.1 --list <program>
	errBuf := new(bytes.Buffer)
	buf := new(bytes.Buffer)

	cmd := exec.Command(interp, "--list", binary)
	cmd.Stdout = buf
	cmd.Stderr = errBuf

	// set an empty environment as LD_LIBRARY_PATH
	// may mix dependencies, just rely only on the library
	// cache or its own lookup mechanism, see issue:
	// https://github.com/hpcng/singularity/issues/5666
	cmd.Env = []string{}

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("while getting library dependencies: %s\n%s", err, errBuf.String())
	}

	// parse the output to get matches for ' /an/absolute/path ('
	re := regexp.MustCompile(`[[:blank:]]?(\/.*)[[:blank:]]\(`)

	match := re.FindAllStringSubmatch(buf.String(), -1)
	for _, m := range match {
		if len(m) < 2 {
			continue
		}
		lib := m[1]
		has := false
		for _, l := range libs {
			if l == lib {
				has = true
				break
			}
		}
		if !has {
			libs = append(libs, lib)
		}
	}

	return libs, nil
}

// unsquashfsSandboxCmd is the command instance for executing unsquashfs command
// in a sandboxed environment with singularity.
func unsquashfsSandboxCmd(unsquashfs string, dest string, filename string, filter string, opts ...string) (*exec.Cmd, error) {
	const (
		// will contain both dest and filename inside the sandbox
		rootfsImageDir = "/image"
	)

	// create the sandbox temporary directory
	tmpdir := filepath.Dir(dest)
	rootfs, err := ioutil.TempDir(tmpdir, "tmp-rootfs-")
	if err != nil {
		return nil, fmt.Errorf("failed to create chroot directory: %s", err)
	}

	overwrite := false

	// remove the destination directory if any, if the directory is
	// not empty (typically during image build), the unsafe option -f is
	// set, this is unfortunately required by image build
	if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
		if !os.IsExist(err) {
			return nil, fmt.Errorf("failed to remove %s: %s", dest, err)
		}
		overwrite = true
	}

	// map destination into the sandbox
	rootfsDest := filepath.Join(rootfsImageDir, filepath.Base(dest))

	// sandbox required directories
	rootfsDirs := []string{
		// unsquashfs get available CPU from /sys/devices/system/cpu/online
		filepath.Join(rootfs, "/sys"),
		filepath.Join(rootfs, "/dev"),
		filepath.Join(rootfs, rootfsImageDir),
	}

	for _, d := range rootfsDirs {
		if err := os.Mkdir(d, 0700); err != nil {
			return nil, fmt.Errorf("while creating %s: %s", d, err)
		}
	}

	// the decision to use user namespace is left to singularity
	// which will detect automatically depending of the configuration
	// what workflow it could use
	args := []string{
		"-q",
		"exec",
		"--no-home",
		"--no-nv",
		"--no-rocm",
		"-C",
		"--no-init",
		"--writable",
		"-B", fmt.Sprintf("%s:%s", tmpdir, rootfsImageDir),
	}

	if filename != stdinFile {
		filename = filepath.Join(rootfsImageDir, filepath.Base(filename))
	}

	// get the library dependencies of unsquashfs
	libs, err := getLibraries(unsquashfs)
	if err != nil {
		return nil, err
	}
	libraryPath := make([]string, 0)

	roFiles := []string{
		unsquashfs,
	}

	// add libraries for bind mount and also generate
	// LD_LIBRARY_PATH
	for _, l := range libs {
		dir := filepath.Dir(l)
		roFiles = append(roFiles, l)
		has := false
		for _, lp := range libraryPath {
			if lp == dir {
				has = true
				break
			}
		}
		if !has {
			libraryPath = append(libraryPath, dir)
		}
	}

	// create files and directories in the sandbox and
	// add singularity bind mount options
	for _, b := range roFiles {
		file := filepath.Join(rootfs, b)
		dir := filepath.Dir(file)
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, fmt.Errorf("while creating %s: %s", dir, err)
		}
		if err := ioutil.WriteFile(file, []byte(""), 0600); err != nil {
			return nil, fmt.Errorf("while creating %s: %s", file, err)
		}
		args = append(args, "-B", fmt.Sprintf("%s:%s:ro", b, b))
	}

	// singularity sandbox
	args = append(args, rootfs)

	// unsquashfs execution arguments
	args = append(args, unsquashfs)
	args = append(args, opts...)

	if overwrite {
		args = append(args, "-f")
	}

	args = append(args, "-d", rootfsDest, filename)

	if filter != "" {
		args = append(args, filter)
	}

	sylog.Debugf("Calling wrapped unsquashfs: singularity %v", args)
	cmd := exec.Command(filepath.Join(buildcfg.BINDIR, "singularity"), args...)
	cmd.Dir = "/"
	cmd.Env = []string{
		fmt.Sprintf("LD_LIBRARY_PATH=%s", strings.Join(libraryPath, string(os.PathListSeparator))),
	}

	return cmd, nil
}
