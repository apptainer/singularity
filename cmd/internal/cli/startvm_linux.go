// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

func startVM(sifImage, singAction, cliExtra string, isInternal bool) error {

	// Setup some needed variables
	bzImage := fmt.Sprintf(buildcfg.LIBEXECDIR+"%s"+runtime.GOARCH, "/singularity/vm/syos-kernel-")
	initramfs := fmt.Sprintf(buildcfg.LIBEXECDIR+"%s"+runtime.GOARCH+".gz", "/singularity/vm/initramfs_")
	appendArgs := fmt.Sprintf("root=/dev/ram0 console=ttyS0 quiet singularity_action=%s singularity_arguments=\"%s\"", singAction, cliExtra)

	defArgs := []string{""}
	defArgs = []string{"-cpu", "host", "-smp", VmCpu, "-enable-kvm", "-device", "virtio-rng-pci", "-display", "none", "-realtime", "mlock=on", "-hda", sifImage, "-serial", "stdio", "-kernel", bzImage, "-initrd", initramfs, "-m", VmRam, "-append", appendArgs}

	pgmExec, lookErr := exec.LookPath("/usr/libexec/qemu-kvm")
	if lookErr != nil {
		sylog.Fatalf("Failed to find qemu-kvm executable at /usr/libexec/qemu-kvm")
	}

	if _, err := os.Stat(sifImage); os.IsNotExist(err) {
		sylog.Fatalf("Failed to determine image absolute path for %s: %s", sifImage, err)
	}
	if _, err := os.Stat(bzImage); os.IsNotExist(err) {
		sylog.Fatalf("This functionality is not supported.")
	}
	if _, err := os.Stat(initramfs); os.IsNotExist(err) {
		sylog.Fatalf("This functionality is not supported."
	}

	cmd := exec.Command(pgmExec, defArgs...)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		sylog.Debugf("Hypervisor exit code: %v\n", err)

		if exitErr, ok := err.(*exec.ExitError); ok {
			//Program exited with non-zero return code
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				sylog.Fatalf("Process exited with non-zero return code: %d\n", status.ExitStatus())
			}
		}

		sylog.Fatalf("Process exited with unknown error: %v\n", err)
	}

	return nil
}
