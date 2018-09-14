// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package imgbuild

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/singularityware/singularity/src/pkg/sylog"
)

// StartProcess runs the %post script
func (e *EngineOperations) StartProcess(masterConn net.Conn) error {

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

	// append environment
	if err := e.insertEnvScript(); err != nil {
		return fmt.Errorf("While inserting environment script: %v", err)
	}

	// insert startscript
	if err := e.insertStartScript(); err != nil {
		return fmt.Errorf("While inserting startscript: %v", err)
	}

	// insert runscript
	if err := e.insertRunScript(); err != nil {
		return fmt.Errorf("While inserting runscript: %v", err)
	}

	// insert test script
	if err := e.insertTestScript(); err != nil {
		return fmt.Errorf("While inserting test script: %v", err)
	}

	if e.EngineConfig.RunSection("test") {
		if !e.EngineConfig.NoTest && e.EngineConfig.Recipe.BuildData.Test != "" {
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
func (e *EngineOperations) MonitorContainer(pid int) (syscall.WaitStatus, error) {
	var status syscall.WaitStatus

	signals := make(chan os.Signal, 1)
	signal.Notify(signals)

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
			return status, fmt.Errorf("interrupted by signal %s", s.String())
		}
	}
}

// CleanupContainer _
func (e *EngineOperations) CleanupContainer() error {
	return nil
}

// PostStartProcess actually does nothing for build engine
func (e *EngineOperations) PostStartProcess(pid int) error {
	return nil
}

func (e *EngineOperations) insertEnvScript() error {
	if e.EngineConfig.RunSection("environment") && e.EngineConfig.Recipe.ImageData.Environment != "" {
		sylog.Infof("Adding environment to container")
		err := ioutil.WriteFile("/.singularity.d/env/90-environment.sh", []byte("#!/bin/sh\n\n"+e.EngineConfig.Recipe.ImageData.Environment+"\n"), 0775)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *EngineOperations) insertRunScript() error {
	if e.EngineConfig.RunSection("runscript") && e.EngineConfig.Recipe.ImageData.Runscript != "" {
		sylog.Infof("Adding runscript")
		err := ioutil.WriteFile("/.singularity.d/runscript", []byte("#!/bin/sh\n\n"+e.EngineConfig.Recipe.ImageData.Runscript+"\n"), 0775)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *EngineOperations) insertStartScript() error {
	if e.EngineConfig.RunSection("startscript") && e.EngineConfig.Recipe.ImageData.Startscript != "" {
		sylog.Infof("Adding startscript")
		err := ioutil.WriteFile("/.singularity.d/startscript", []byte("#!/bin/sh\n\n"+e.EngineConfig.Recipe.ImageData.Startscript+"\n"), 0775)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *EngineOperations) insertTestScript() error {
	if e.EngineConfig.RunSection("test") && e.EngineConfig.Recipe.ImageData.Test != "" {
		sylog.Infof("Adding testscript")
		err := ioutil.WriteFile("/.singularity.d/test", []byte("#!/bin/sh\n\n"+e.EngineConfig.Recipe.ImageData.Test+"\n"), 0775)
		if err != nil {
			return err
		}
	}
	return nil
}
