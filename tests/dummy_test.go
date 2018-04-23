package tests

import (
	"os/exec"
	"syscall"
	"testing"
)

func Test_ImageBuild(t *testing.T) {
	t.Run("Docker", docker)
}

func docker(t *testing.T) {
	singularity, err := exec.LookPath("singularity")
	if err != nil {
		t.Error("singularity is not installed on this system")
		return
	}

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
