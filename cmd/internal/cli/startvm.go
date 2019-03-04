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
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

func execVM(cmd *cobra.Command, image string, args []string) {
	// SIF image we are running
	sifImage := image

	cliExtra := ""
	singAction := cmd.Name()

	imgPath := strings.Split(sifImage, ":")
	isInternal := false
	if strings.HasPrefix("internal", filepath.Base(imgPath[0])) {
		cliExtra = "syos"
		isInternal = true
	} else {
		// Get our "action" (run, exec, shell) based on the action script being called
		singAction = filepath.Base(args[0])
		cliExtra = strings.Join(args[1:], " ")
	}

	if err := startVM(sifImage, singAction, cliExtra, isInternal); err != nil {
		sylog.Errorf("VM instance failed: %s", err)
		os.Exit(2)
	}
}

func startVM(sifImage, singAction, cliExtra string, isInternal bool) error {
	bzImage := fmt.Sprintf("%s/%s-%s", buildcfg.LIBEXECDIR, "/singularity/vm/syos-kernel", runtime.GOARCH)
	initramfs := fmt.Sprintf("%s/%s_%s.gz", buildcfg.LIBEXECDIR, "/singularity/vm/initramfs", runtime.GOARCH)

	args := getHypervisorArgs(sifImage, bzImage, initramfs, singAction, cliExtra)

	sylog.Debugf("About to launch VM using: %+v", args)

	hvExec, lookErr := exec.LookPath(args[0])
	if lookErr != nil {
		sylog.Fatalf("Failed to find hypervisor executable at %s", args[0])
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

	args[0] = "Sylabs"
	cmd := exec.Command(hvExec)
	cmd.Args = args
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	if VMErr || debug {
		cmd.Stderr = os.Stderr
	}

	err := cmd.Run()

	if err != nil {
		sylog.Debugf("Hypervisor exit code: %v\n", err)

		if exitErr, ok := err.(*exec.ExitError); ok {
			//Program exited with non-zero return code
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				sylog.Debugf("Process exited with non-zero return code: %d\n", status.ExitStatus())
			}
		}
	}

	return err
}
