// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"syscall"
	"testing"

	// Tests imports
	"github.com/sylabs/singularity/e2e/actions"
	e2ebuildcfg "github.com/sylabs/singularity/e2e/buildcfg"
	"github.com/sylabs/singularity/e2e/cache"
	"github.com/sylabs/singularity/e2e/cmdenvvars"
	"github.com/sylabs/singularity/e2e/docker"
	singularityenv "github.com/sylabs/singularity/e2e/env"
	"github.com/sylabs/singularity/e2e/help"
	"github.com/sylabs/singularity/e2e/imgbuild"
	"github.com/sylabs/singularity/e2e/inspect"
	"github.com/sylabs/singularity/e2e/instance"
	"github.com/sylabs/singularity/e2e/key"
	"github.com/sylabs/singularity/e2e/oci"
	"github.com/sylabs/singularity/e2e/pull"
	"github.com/sylabs/singularity/e2e/push"
	"github.com/sylabs/singularity/e2e/regressions"
	"github.com/sylabs/singularity/e2e/remote"
	"github.com/sylabs/singularity/e2e/run"
	"github.com/sylabs/singularity/e2e/security"
	"github.com/sylabs/singularity/e2e/sign"
	"github.com/sylabs/singularity/e2e/verify"
	"github.com/sylabs/singularity/e2e/version"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
)

var runDisabled = flag.Bool("run_disabled", false, "run tests that have been temporarily disabled")

// Run is the main func for the test framework, initializes the required vars
// and sets the environment for the RunE2ETests framework
func Run(t *testing.T) {
	flag.Parse()

	var testenv e2e.TestEnv

	if *runDisabled {
		testenv.RunDisabled = true
	}
	// init buildcfg values
	useragent.InitValue(buildcfg.PACKAGE_NAME, buildcfg.PACKAGE_VERSION)

	// Ensure binary is in $PATH
	cmdPath := filepath.Join(buildcfg.BINDIR, "singularity")
	if _, err := exec.LookPath(cmdPath); err != nil {
		log.Fatalf("singularity is not installed on this system: %v", err)
	}

	testenv.CmdPath = cmdPath

	sysconfdir := func(fn string) string {
		return filepath.Join(buildcfg.SYSCONFDIR, "singularity", fn)
	}

	// e2e tests need to run in a somehow agnostic environment, so we
	// don't use environment of user executing tests in order to not
	// wrongly interfering with cache stuff, sylabs library tokens,
	// PGP keys
	e2e.SetupHomeDirectories(t)

	// Ensure config files are installed
	configFiles := []string{
		sysconfdir("singularity.conf"),
		sysconfdir("ecl.toml"),
		sysconfdir("capability.json"),
		sysconfdir("nvliblist.conf"),
	}

	for _, cf := range configFiles {
		if fi, err := os.Stat(cf); err != nil {
			log.Fatalf("%s is not installed on this system: %v", cf, err)
		} else if !fi.Mode().IsRegular() {
			log.Fatalf("%s is not a regular file", cf)
		} else if fi.Sys().(*syscall.Stat_t).Uid != 0 {
			log.Fatalf("%s must be owned by root", cf)
		}
	}

	// Make temp dir for tests
	name, err := ioutil.TempDir("", "stest.")
	if err != nil {
		log.Fatalf("failed to create temporary directory: %v", err)
	}
	defer e2e.Privileged(func(t *testing.T) {
		os.RemoveAll(name)
	})(t)

	if err := os.Chmod(name, 0755); err != nil {
		log.Fatalf("failed to chmod temporary directory: %v", err)
	}
	testenv.TestDir = name

	// Build a base image for tests
	imagePath := path.Join(name, "test.sif")
	t.Log(imagePath)
	testenv.ImagePath = imagePath
	defer os.Remove(imagePath)

	// WARNING(Sylabs-team): Please DO NOT add a call to e2e.EnsureImage here.
	// If you need the test image, add the call at the top of your
	// own test.

	testenv.TestRegistry = "localhost:5000"
	testenv.OrasTestImage = fmt.Sprintf("oras://%s/oras_test_sif:latest", testenv.TestRegistry)

	// WARNING(Sylabs-team): Please DO NOT add a call to
	// e2e.PrepRegistry here. If you need to access the local
	// registry, add the call at the top of your own test.
	//
	// e2e.KillRegistry is called here to ensure that the registry
	// is stopped after tests run.
	defer e2e.KillRegistry(t, testenv)

	// RunE2ETests by functionality
	suites := map[string]func(*testing.T){
		"SECURITY":    security.E2ETests(testenv),
		"ACTIONS":     actions.E2ETests(testenv),
		"BUILD":       imgbuild.E2ETests(testenv),
		"BUILDCFG":    e2ebuildcfg.E2ETests(testenv),
		"CACHE":       cache.E2ETests(testenv),
		"CMDENVVARS":  cmdenvvars.E2ETests(testenv),
		"DOCKER":      docker.E2ETests(testenv),
		"ENV":         singularityenv.E2ETests(testenv),
		"HELP":        help.E2ETests(testenv),
		"INSPECT":     inspect.E2ETests(testenv),
		"INSTANCE":    instance.E2ETests(testenv),
		"KEY":         key.E2ETests(testenv),
		"OCI":         oci.E2ETests(testenv),
		"PULL":        pull.E2ETests(testenv),
		"PUSH":        push.E2ETests(testenv),
		"REMOTE":      remote.E2ETests(testenv),
		"RUN":         run.E2ETests(testenv),
		"SIGN":        sign.E2ETests(testenv),
		"VERIFY":      verify.E2ETests(testenv),
		"VERSION":     version.E2ETests(testenv),
		"REGRESSIONS": regressions.E2ETests(testenv),
	}

	for name, fn := range suites {
		t.Run(name, fn)
	}
}
