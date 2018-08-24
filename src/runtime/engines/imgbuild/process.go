// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
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

	"github.com/singularityware/singularity/src/pkg/build/types"
	"github.com/singularityware/singularity/src/pkg/sylog"
)

// StartProcess runs the %post script
func (e *EngineOperations) StartProcess(masterConn net.Conn) error {

	if e.EngineConfig.Recipe.BuildData.Post != "" {
		// Run %post script here
		post := exec.Command("/bin/sh", "-c", e.EngineConfig.Recipe.BuildData.Post)
		post.Env = e.CommonConfig.OciConfig.Process.Env
		post.Stdout = os.Stdout
		post.Stderr = os.Stderr

		sylog.Infof("Running %%post script\n")
		if err := post.Start(); err != nil {
			sylog.Fatalf("failed to start %%post proc: %v\n", err)
		}
		if err := post.Wait(); err != nil {
			sylog.Fatalf("post proc: %v\n", err)
		}
		sylog.Infof("Finished running %%post script. exit status 0\n")
	}

	//append environment
	if err := insertEnvScript(e.EngineConfig.Recipe); err != nil {
		return fmt.Errorf("While inserting environment script: %v", err)
	}

	//insert startscript
	if err := insertStartScript(e.EngineConfig.Recipe); err != nil {
		return fmt.Errorf("While inserting startscript: %v", err)
	}

	//insert runscript
	if err := insertRunScript(e.EngineConfig.Recipe); err != nil {
		return fmt.Errorf("While inserting runscript: %v", err)
	}

	//insert test script
	if err := insertTestScript(e.EngineConfig.Recipe); err != nil {
		return fmt.Errorf("While inserting test script: %v", err)
	}

	// Run %test script here if its defined
	// this also needs to consider the --notest flag from the CLI eventually
	if !e.EngineConfig.NoTest && e.EngineConfig.Recipe.BuildData.Test != "" {
		test := exec.Command("/bin/sh", "-c", e.EngineConfig.Recipe.BuildData.Test)
		test.Stdout = os.Stdout
		test.Stderr = os.Stderr

		sylog.Infof("Running %%test script\n")
		if err := test.Start(); err != nil {
			sylog.Fatalf("failed to start %%test proc: %v\n", err)
		}
		if err := test.Wait(); err != nil {
			sylog.Fatalf("test proc: %v\n", err)
		}
		sylog.Infof("Finished running %%test script. exit status 0\n")
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

func insertEnvScript(d types.Definition) error {
	if d.ImageData.Environment != "" {
		sylog.Infof("Adding environment to container")
		err := ioutil.WriteFile("/.singularity.d/env/90-environment.sh", []byte("#!/bin/sh\n\n"+d.ImageData.Environment+"\n"), 0775)
		if err != nil {
			return err
		}
	}
	return nil
}

func insertRunScript(d types.Definition) error {
	if d.ImageData.Runscript != "" {
		sylog.Infof("Adding runscript")
		err := ioutil.WriteFile("/.singularity.d/runscript", []byte("#!/bin/sh\n\n"+d.ImageData.Runscript+"\n"), 0775)
		if err != nil {
			return err
		}
	}
	return nil
}

func insertStartScript(d types.Definition) error {
	if d.ImageData.Startscript != "" {
		sylog.Infof("Adding startscript")
		err := ioutil.WriteFile("/.singularity.d/startscript", []byte("#!/bin/sh\n\n"+d.ImageData.Startscript+"\n"), 0775)
		if err != nil {
			return err
		}
	}
	return nil
}

func insertTestScript(d types.Definition) error {
	if d.ImageData.Test != "" {
		sylog.Infof("Adding testscript")
		err := ioutil.WriteFile("/.singularity.d/test", []byte("#!/bin/sh\n\n"+d.ImageData.Test+"\n"), 0775)
		if err != nil {
			return err
		}
	}
	return nil
}
