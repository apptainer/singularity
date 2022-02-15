// Copyright (c) 2020-2022, Sylabs Inc. All rights reserved.
// Copyright (c) 2020, Control Command Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build singularity_engine

package unpacker

import (
	"bufio"
	"bytes"
	"debug/elf"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hpcng/singularity/internal/pkg/buildcfg"
	"github.com/hpcng/singularity/pkg/sylog"
)

func init() {
	cmdFunc = unsquashfsSandboxCmd
}

// libBind represents a library bind mount required by an elf binary
// that will be run in a contained minimal filesystem.
type libBind struct {
	// source is the path to bind from, on the host.
	source string
	// dest is the path to bind to, inside the minimal filesystem.
	dest string
}

// getLibraryBinds returns the library bind mounts required by an elf binary.
// The binary path must be absolute.
func getLibraryBinds(binary string) ([]libBind, error) {
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
		return []libBind{}, nil
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

	return parseLibraryBinds(buf)
}

// parseLibrary binds parses `ld-linux-x86-64.so.2 --list <binary>` output.
// Returns a list of source->dest bind mounts required to run the binary
// in a minimal contained filesystem.
func parseLibraryBinds(buf io.Reader) ([]libBind, error) {
	libs := make([]libBind, 0)
	scanner := bufio.NewScanner(buf)

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		// /lib64/ld64.so.2 (0x00007fff96c60000)
		// Absolute path in 1st field - bind directly dest=source
		if filepath.IsAbs(fields[0]) {
			libs = append(libs, libBind{
				source: fields[0],
				dest:   fields[0],
			})
			continue
		}
		// libpthread.so.0 => /lib64/libpthread.so.0 (0x00007fff96a20000)
		//    .. or with glibc-hwcaps ..
		// libpthread.so.0 => /lib64/glibc-hwcaps/power9/libpthread-2.28.so (0x00007fff96a20000)
		//
		// Bind resolved lib to same dir, but with .so filename from 1st field.
		// e.g. source is: /lib64/glibc-hwcaps/power9/libpthread-2.28.so
		//      dest is  : /lib64/glibc-hwcaps/power9/libpthread.so.0
		if len(fields) >= 3 && fields[1] == "=>" {
			destDir := filepath.Dir(fields[2])
			dest := filepath.Join(destDir, fields[0])
			libs = append(libs, libBind{
				source: fields[2],
				dest:   dest,
			})
		}
		// linux-vdso64.so.1 (0x00007fff96c40000)
		//   .. or anything else
		// No absolute path = nothing to bind
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("while parsing library dependencies: %v", err)
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

	roFiles := []string{
		unsquashfs,
	}

	// get the library dependencies of unsquashfs
	libs, err := getLibraryBinds(unsquashfs)
	if err != nil {
		return nil, err
	}

	// Handle binding of files
	for _, b := range roFiles {
		// Ensure parent dir and file exist in container
		rootfsFile := filepath.Join(rootfs, b)
		rootfsDir := filepath.Dir(rootfsFile)
		if err := os.MkdirAll(rootfsDir, 0o700); err != nil {
			return nil, fmt.Errorf("while creating %s: %s", rootfsDir, err)
		}
		if err := ioutil.WriteFile(rootfsFile, []byte(""), 0o600); err != nil {
			return nil, fmt.Errorf("while creating %s: %s", rootfsFile, err)
		}
		// Simple read-only bind, dest in container same as source on host
		args = append(args, "-B", fmt.Sprintf("%s:%s:ro", b, b))
	}

	// Handle binding of libs and generate LD_LIBRARY_PATH
	libraryPath := make([]string, 0)
	for _, l := range libs {
		// Ensure parent dir and file exist in container
		rootfsFile := filepath.Join(rootfs, l.dest)
		rootfsDir := filepath.Dir(rootfsFile)
		if err := os.MkdirAll(rootfsDir, 0o700); err != nil {
			return nil, fmt.Errorf("while creating %s: %s", rootfsDir, err)
		}
		if err := ioutil.WriteFile(rootfsFile, []byte(""), 0o600); err != nil {
			return nil, fmt.Errorf("while creating %s: %s", rootfsFile, err)
		}
		// Read only bind, dest in container may not match source on host due
		// to .so symlinking (see getLibraryBinds comments).
		args = append(args, "-B", fmt.Sprintf("%s:%s:ro", l.source, l.dest))
		// If dir of lib not already in the LD_LIBRARY_PATH, add it.
		has := false
		libraryDir := filepath.Dir(l.dest)
		for _, lp := range libraryPath {
			if lp == libraryDir {
				has = true
				break
			}
		}
		if !has {
			libraryPath = append(libraryPath, libraryDir)
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
