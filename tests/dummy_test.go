package tests

import (
	"fmt"
	"os/exec"
	"syscall"
	"testing"
)

func Test_example(t *testing.T) {
	fmt.Println("Hi there, this is a failing test!")
}

func Test_ImageBuild(t *testing.T) {
	t.Run("Docker", docker)
}

func docker(t *testing.T) {
	dockerBuild := exec.Command("../core/buildtree/singularity", "build", "image.sif", "docker://ubuntu")

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
