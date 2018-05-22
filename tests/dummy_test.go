// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package tests

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"testing"
)

var singularity string

func TestMain(m *testing.M) {
	var err error
	singularity, err = exec.LookPath("singularity")
	if err != nil {
		fmt.Println("singularity is not installed on this system")
		os.Exit(1)
	}

	exitcode := m.Run()

	if err := os.Remove("image.sif"); err != nil {
		fmt.Printf("Unable to remove file: %s", err)
	}

	os.Exit(exitcode)
}

func Test_ImageBuild(t *testing.T) {
	t.Run("Docker", docker)
	t.Run("Exec", sExec)
}

func docker(t *testing.T) {
	dockerBuild := exec.Command(singularity, "build", "image.sif", "docker://ubuntu")

	if out, err := dockerBuild.CombinedOutput(); err != nil {
		t.Error(string(out))
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				t.Errorf("Exit Status: %d", status.ExitStatus())
			}
		} else {
			t.Errorf("cmd.Wait: %v", err)
		}
	}
}

func sExec(t *testing.T) {
	singularityExec := exec.Command(singularity, "exec", "image.sif", "ps")

	if out, err := singularityExec.CombinedOutput(); err != nil {
		t.Error(string(out))
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				t.Errorf("Exit Status: %d", status.ExitStatus())
			}
		} else {
			t.Errorf("cmd.Wait: %v", err)
		}
	}
}
