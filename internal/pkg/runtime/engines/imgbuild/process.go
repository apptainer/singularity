// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package imgbuild

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/opencontainers/runtime-tools/generate"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/env"
)

// StartProcess runs the %post script
func (e *EngineOperations) StartProcess(masterConn net.Conn) error {

	// clean environment in which %post and %test scripts are run in
	e.cleanEnv()

	if e.EngineConfig.RunSection("post") && e.EngineConfig.Recipe.BuildData.Post != "" {
		// Run %post script here
		post := exec.Command("/bin/sh", "-cex", e.EngineConfig.Recipe.BuildData.Post)
		post.Env = e.EngineConfig.OciConfig.Process.Env
		post.Stdout = os.Stdout
		post.Stderr = os.Stderr

		sylog.Infof("Running post scriptlet\n")
		if err := post.Start(); err != nil {
			sylog.Fatalf("failed to start %%post proc: %v\n", err)
		}
		if err := post.Wait(); err != nil {
			sylog.Fatalf("post proc: %v\n", err)
		}
	}

	if e.EngineConfig.RunSection("test") {
		if !e.EngineConfig.Opts.NoTest && e.EngineConfig.Recipe.BuildData.Test != "" {
			// Run %test script
			test := exec.Command("/bin/sh", "-cex", e.EngineConfig.Recipe.BuildData.Test)
			test.Stdout = os.Stdout
			test.Stderr = os.Stderr

			sylog.Infof("Running test scriptlet\n")
			if err := test.Start(); err != nil {
				sylog.Fatalf("failed to start %%test proc: %v\n", err)
			}
			if err := test.Wait(); err != nil {
				sylog.Fatalf("test proc: %v\n", err)
			}
		}
	}

	os.Exit(0)
	return nil
}

// MonitorContainer is responsible for waiting on container process
func (e *EngineOperations) MonitorContainer(pid int, signals chan os.Signal) (syscall.WaitStatus, error) {
	var status syscall.WaitStatus

	for {
		s := <-signals
		switch s {
		case syscall.SIGCHLD:
			if wpid, err := syscall.Wait4(pid, &status, syscall.WNOHANG, nil); err != nil {
				return status, fmt.Errorf("error while waiting child: %s", err)
			} else if wpid != pid {
				continue
			}
			return status, nil
		default:
			if err := syscall.Kill(pid, s.(syscall.Signal)); err != nil {
				return status, fmt.Errorf("interrupted by signal %s", s.String())
			}
		}
	}
}

// CleanupContainer _
func (e *EngineOperations) CleanupContainer(fatal error, status syscall.WaitStatus) error {
	return nil
}

// PostStartProcess actually does nothing for build engine
func (e *EngineOperations) PostStartProcess(pid int) error {
	return nil
}

func (e *EngineOperations) cleanEnv() {
	generator := generate.Generator{Config: &e.EngineConfig.OciConfig.Spec}

	// copy and cache environment
	environment := e.EngineConfig.OciConfig.Spec.Process.Env

	// clean environment
	e.EngineConfig.OciConfig.Spec.Process.Env = nil

	// add relevant environment variables back
	env.SetContainerEnv(&generator, environment, true, "")

	// expose build specific environment variables for scripts
	for _, envVar := range environment {
		e := strings.SplitN(envVar, "=", 2)
		if e[0] == "SINGULARITY_ROOTFS" || e[0] == "SINGULARITY_ENVIRONMENT" {
			generator.Config.Process.Env = append(generator.Config.Process.Env, envVar)
		}

	}

}
