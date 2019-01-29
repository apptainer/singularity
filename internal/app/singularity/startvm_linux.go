// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

func startVM(sifImage, singAction, cliExtra string, isInternal bool) error {
	const defaultFailedCode = 1
	var exitCode int

	var stdoutBuf, stderrBuf bytes.Buffer
	hdString := fmt.Sprintf("-hda %s", sifImage)

	bzImage := fmt.Sprintf(buildcfg.LIBEXECDIR+"%s"+runtime.GOARCH, "/singularity/vm/syos-kernel-")
	initramfs := fmt.Sprintf(buildcfg.LIBEXECDIR+"%s"+runtime.GOARCH+".gz", "/singularity/vm/initramfs_")
	appendArgs := fmt.Sprintf("root=/dev/ram0 console=ttyS0 quiet singularity_action=%s singularity_arguments=\"%s\"", singAction, cliExtra)

	defArgs := []string{""}
	if cliExtra == "syos" && isInternal {
		//fmt.Println("defArgs - without -hda")
		defArgs = []string{"-cpu", "host", "-enable-kvm", "-device", "virtio-rng-pci", "-display", "none", "-realtime", "mlock=on", "-serial", "stdio", "-kernel", "%s", "-initrd", "%s", "-m", "4096", "-append", bzImage, initramfs, appendArgs}
	} else {
		//fmt.Println("defArgs - with -hda")
		defArgs = []string{"-cpu", "host", "-enable-kvm", "-device", "virtio-rng-pci", "-display", "none", "-realtime", "mlock=on", hdString, "-serial", "stdio", "-kernel", "%s", "-initrd", "%s", "-m", "4096", "-append", bzImage, initramfs, appendArgs}
	}

	pgmexec, lookErr := exec.LookPath("/usr/libexec/qemu-kvm")
	if lookErr != nil {
		sylog.Fatalf("Failed to find qemu-kvm executable at /usr/libexec/qemu-kvm")
	}

	if _, err := os.Stat(sifImage); os.IsNotExist(err) {
		sylog.Fatalf("Failed to determine image absolute path for %s: %s", sifImage, err)
	}
	if _, err := os.Stat(bzImage); os.IsNotExist(err) {
		sylog.Fatalf("Failed to determine image absolute path for %s: %s \nPlease contact sales@sylabs.io for info on how to license SyOS. \n\n", bzImage, err)
	}
	if _, err := os.Stat(initramfs); os.IsNotExist(err) {
		sylog.Fatalf("Failed to determine image absolute path for %s: %s \nPlease contact sales@sylabs.io for info on how to license SyOS. \n\n", initramfs, err)
	}

	cmd := exec.Command(pgmexec, defArgs...)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin

	stdoutIn, _ := cmd.StdoutPipe()
	stderrIn, _ := cmd.StderrPipe()

	var errStdout, errStderr error
	stdout := io.MultiWriter(os.Stdout, &stdoutBuf)
	stderr := io.MultiWriter(os.Stderr, &stderrBuf)

	exitCode = defaultFailedCode

	cmdErr := cmd.Run()
	if cmdErr != nil {
		// try to get the exit code
		if exitError, ok := cmdErr.(*exec.ExitError); ok {
			ws := exitError.Sys().(syscall.WaitStatus)
			exitCode = ws.ExitStatus()
		}
	} else {
		// success, exitCode should be 0 if go is ok
		ws := cmd.ProcessState.Sys().(syscall.WaitStatus)
		exitCode = ws.ExitStatus()
	}
	sylog.Debugf("command result, stdout: %v, stderr: %v, exitCode: %v", errStdout, errStderr, exitCode)

	go func() {
		_, errStdout = io.Copy(stdout, stdoutIn)
	}()

	go func() {
		_, errStderr = io.Copy(stderr, stderrIn)
	}()

	if errStdout != nil || errStderr != nil {
		sylog.Fatalf("failed to capture stdout or stderr\n")
	}

	return nil
}
