// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package imgbuild

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"

	"github.com/opencontainers/runtime-tools/generate"
	"github.com/sylabs/singularity/internal/pkg/util/env"
)

// StartProcess runs the %post script
// StartProcess is called during stage2 after RPC server finished
// environment preparation. This will execute `%post` section of a
// container recipe file.
//
// No additional privileges can be gained during this call. However,
// imgbuild engine is run by root user or fakeroot engine, so technically
// there are elevated privileges here.
func (e *EngineOperations) StartProcess(masterConn net.Conn) error {

	// clean environment in which %post and %test scripts are run in
	e.cleanEnv()

	if e.EngineConfig.RunSection("post") && e.EngineConfig.Recipe.BuildData.Post.Script != "" {
		// Run %post script here
		e.runScriptSection("post", e.EngineConfig.Recipe.BuildData.Post, true)
	}

	if e.EngineConfig.RunSection("test") {
		if !e.EngineConfig.Opts.NoTest && e.EngineConfig.Recipe.BuildData.Test.Script != "" {
			// Run %test script
			e.runScriptSection("test", e.EngineConfig.Recipe.BuildData.Test, false)
		}
	}

	os.Exit(0)
	return nil
}

// MonitorContainer is called from master once the container has
// been spawned. It will block until the container exists.
//
// Additional privileges may be gained when running
// in suid flow. However, when a user namespace is requested and it is not
// a hybrid workflow (e.g. fakeroot), then there is no privileged saved uid
// and thus no additional privileges can be gained.
//
// Particularly here no additional privileges are gained as monitor does
// not need them for wait4 and kill syscalls.
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

// CleanupContainer does nothing for imgbuild engine.
func (e *EngineOperations) CleanupContainer(context.Context, error, syscall.WaitStatus) error {
	return nil
}

// PostStartProcess does nothing for imgbuild engine.
func (e *EngineOperations) PostStartProcess(context.Context, int) error {
	return nil
}

func (e *EngineOperations) cleanEnv() {
	generator := generate.Generator{Config: &e.EngineConfig.OciConfig.Spec}

	// copy and cache environment
	environment := e.EngineConfig.OciConfig.Spec.Process.Env

	// clean environment
	e.EngineConfig.OciConfig.Spec.Process.Env = nil

	// during image build process, home destination is /root as
	// build engine is usable only by root or fakeroot, fakeroot
	// already take care of binding the real user home directory
	// to /root, so we don't need to query user database to determine
	// home directory as it's always /root
	homeDest := "/root"

	// add relevant environment variables back
	env.SetContainerEnv(&generator, environment, true, homeDest)

	// expose build specific environment variables for scripts
	for _, envVar := range environment {
		e := strings.SplitN(envVar, "=", 2)
		if e[0] == "SINGULARITY_ROOTFS" || e[0] == "SINGULARITY_ENVIRONMENT" {
			generator.Config.Process.Env = append(generator.Config.Process.Env, envVar)
		}

	}

}
