// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package tests

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/sylabs/singularity/internal/pkg/sylog"

	"github.com/sylabs/singularity/internal/pkg/test"

	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/pkg/stest"
	useragent "github.com/sylabs/singularity/pkg/util/user-agent"

	// custom builtins
	_ "github.com/sylabs/singularity/tests/builtins/net"
)

var testScripts = []struct {
	name string
	path string
}{
	{"EXAMPLE", "scripts/example/example.test"},
	{"SKIPEXAMPLE", "scripts/example/skip.test"},
	{"NETEXAMPLE", "scripts/example/netecho.test"},
	//{"BUILD", "scripts/build/build.test"},
}

func TestMain(t *testing.T) {
	for _, ts := range testScripts {
		stest.RunScript(ts.name, ts.path, t)
	}
}

func sudoExec(sudo string, args []string) error {
	cmd := exec.Command(sudo, "true")
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sudo init failed: %s", err)
	}
	return nil
}

func init() {
	useragent.InitValue(buildcfg.PACKAGE_NAME, buildcfg.PACKAGE_VERSION)

	sudo, err := exec.LookPath("sudo")
	if err != nil {
		sylog.Fatalf("sudo executable not found in $PATH")
	}

	// first sudo run to ask for password if required
	if err := sudoExec(sudo, []string{"true"}); err != nil {
		sylog.Fatalf("%s", err)
	}

	// maintain sudo session for use in test scripts without
	// password ask
	go func() {
		time.Sleep(1 * time.Minute)
		if err := sudoExec(sudo, []string{"true"}); err != nil {
			sylog.Fatalf("%s", err)
		}
	}()

	testDir, err := ioutil.TempDir("", "stest-")
	if err != nil {
		sylog.Fatalf("%s", err)
	}

	sudoCmd := fmt.Sprintf("%s HOME=/root SINGULARITY_CACHEDIR=%s PATH=%s", sudo, test.CacheDirPriv, os.Getenv("PATH"))
	os.Setenv("SUDO", sudoCmd)
	os.Setenv("TESTDIR", testDir)
	os.Setenv("SINGULARITY_CACHEDIR", test.CacheDirUnpriv)
	os.Setenv("CACHEDIR_PRIV", test.CacheDirPriv)
	os.Setenv("SOURCEDIR", filepath.Dir(buildcfg.BUILDDIR))
}
