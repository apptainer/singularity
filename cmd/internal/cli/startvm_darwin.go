// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

func startVM(sifImage, singAction, cliExtra string, isInternal bool) error {
	// Setup some needed variables
	hdString := fmt.Sprintf("2:0,ahci-hd,%s", sifImage)
	bzImage := fmt.Sprintf(buildcfg.LIBEXECDIR+"%s"+runtime.GOARCH, "/singularity/vm/syos-kernel-")
	initramfs := fmt.Sprintf(buildcfg.LIBEXECDIR+"%s"+runtime.GOARCH+".gz", "/singularity/vm/initramfs_")

	// Default xhyve Arguments
	defArgs := []string{""}
	defArgs = []string{"-A", "-m", VMRAM, "-c", VMCPU, "-s", "0:0,hostbridge", "-s", hdString, "-s", "31,lpc", "-l", "com1,stdio"}

	// Bind mounts
	singBinds := []string{""}

	slot := 5
	function := 0
	for _, bindpath := range BindPaths {
		splitted := strings.Split(bindpath, ":")
		src := splitted[0]
		dst := ""
		if len(splitted) > 1 {
			dst = splitted[1]
		} else {
			dst = src
		}

		mntTag := ""

		sylog.Debugf("Bind path: " + src + " -> " + dst)
		// TODO: Figure out if src is a directory or not
		mntTag = filepath.Base(src)

		pciArgs := fmt.Sprintf("%s:%s,virtio-9p,%s=%s", strconv.Itoa(slot), strconv.Itoa(function), mntTag, src)
		defArgs = append(defArgs, "-s")
		defArgs = append(defArgs, pciArgs)

		localBind := fmt.Sprintf("%s:%s", mntTag, dst)
		singBinds = append(singBinds, localBind)

		sylog.Debugf("PCI: %s", pciArgs)

		function++
		if function > 7 {
			sylog.Fatalf("Maximum of 8 bind mounts")
		}
	}

	// Force $HOME to be mounted
	// TODO: engineConfig.GetHomeSource() / GetHomeDest() -- should probably be used eventually
	homeSrc := os.Getenv("HOME")
	pciArgs := fmt.Sprintf("4:0,virtio-9p,home=%s", homeSrc)
	homeBind := fmt.Sprintf("home:%s", homeSrc)
	singBinds = append(singBinds, homeBind)

	sylog.Debugf("PCI: %s", pciArgs)
	defArgs = append(defArgs, "-s")
	defArgs = append(defArgs, pciArgs)

	if IsSyOS {
		cliExtra = "syos"
	}

	kexecArgs := fmt.Sprintf("kexec,%s,%s,console=ttyS0 quiet root=/dev/ram0 loglevel=0 singularity_action=%s singularity_arguments=\"%s\" singularity_binds=\"%v\"", bzImage, initramfs, singAction, cliExtra, strings.Join(singBinds, "|"))

	// Add our actual kexec entry
	defArgs = append(defArgs, "-f")
	defArgs = append(defArgs, kexecArgs)

	pgmExec, lookErr := exec.LookPath("/usr/local/libexec/singularity/vm/xhyve")
	if lookErr != nil {
		sylog.Fatalf("Failed to find xhyve executable at /usr/local/libexec/singularity/vm/xhyve")
	}

	if _, err := os.Stat(sifImage); os.IsNotExist(err) {
		sylog.Fatalf("Failed to determine image absolute path for %s: %s", sifImage, err)
	}
	if _, err := os.Stat(bzImage); os.IsNotExist(err) {
		sylog.Fatalf("This functionality is not supported")
	}
	if _, err := os.Stat(initramfs); os.IsNotExist(err) {
		sylog.Fatalf("This functionality is not supported")
	}

	sylog.Debugf("%s", singBinds)
	sylog.Debugf("%s", defArgs)
	cmd := exec.Command(pgmExec)
	cmd.Args = append([]string{"Sylabs"}, defArgs...)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	if VMErr || debug {
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		sylog.Debugf("Hypervisor exit code: %v\n", err)

		if exitErr, ok := err.(*exec.ExitError); ok {
			//Program exited with non-zero return code
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				sylog.Debugf("Process exited with non-zero return code: %d\n", status.ExitStatus())
			}
		}
	}

	return nil
}
