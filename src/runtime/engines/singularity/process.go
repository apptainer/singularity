// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/singularityware/singularity/src/pkg/sylog"
)

// StartProcess starts the process
func (engine *EngineOperations) StartProcess(masterConn net.Conn) error {
	isInstance := engine.EngineConfig.GetInstance()

	if err := os.Chdir(engine.CommonConfig.OciConfig.Process.Cwd); err != nil {
		os.Chdir("/")
	}

	args := engine.CommonConfig.OciConfig.Process.Args
	env := engine.CommonConfig.OciConfig.Process.Env

	if !isInstance || (isInstance && engine.EngineConfig.GetBootInstance()) {
		err := syscall.Exec(args[0], args, env)
		return err
	}

	cmd := exec.Command(args[0], args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = env

	var status syscall.WaitStatus
	errChan := make(chan error, 1)
	signals := make(chan os.Signal, 1)

	if err := cmd.Start(); err != nil {
		sylog.Fatalf("exec %s failed: %s", args[0], err)
	}

	go func() {
		errChan <- cmd.Wait()
	}()

	masterConn.Close()

	signal.Notify(signals)

	for {
		select {
		case s := <-signals:
			sylog.Debugf("Received signal %s", s.String())
			switch s {
			case syscall.SIGCHLD:
				for {
					wpid, err := syscall.Wait4(-1, &status, syscall.WNOHANG, nil)
					if wpid <= 0 || err != nil {
						break
					}
				}
			case syscall.SIGCONT:
			default:
				if isInstance {
					syscall.Kill(-1, s.(syscall.Signal))
				} else {
					// kill ourself with SIGKILL whatever signal was received
					syscall.Kill(syscall.Gettid(), syscall.SIGKILL)
				}
			}
		case err := <-errChan:
			if e, ok := err.(*exec.ExitError); ok {
				if status, ok := e.Sys().(syscall.WaitStatus); ok {
					if status.Signaled() {
						syscall.Kill(syscall.Gettid(), syscall.SIGKILL)
					}
					os.Exit(status.ExitStatus())
				}
				sylog.Fatalf("command exit with error: %s", err)
			}
			if err != nil {
				os.Exit(1)
			}
			if !isInstance {
				os.Exit(0)
			}
		}
	}
}
