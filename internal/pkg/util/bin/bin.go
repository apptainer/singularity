// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// Package bin provides access to system binaries
package bin

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	singularity "github.com/sylabs/singularity/pkg/runtime/engines/singularity/config"
)

var (
	// errCryptsetupNotFound is returned when cryptsetup is not found
	errCryptsetupNotFound = errors.New("cryptsetup not found")

	cache struct {
		sync.Once
		cryptsetup string
		err        error
	}
)

// Cryptsetup looks for the "cryptsetup" program returning the absolute
// path to it. If the cryptsetup program is not available, this function
// returns a non-nil error.
func Cryptsetup() (string, error) {
	cache.Do(func() {
		cfgpath := filepath.Join(buildcfg.SINGULARITY_CONFDIR, "singularity.conf")
		cache.cryptsetup, cache.err = cryptsetup(cfgpath)
		sylog.Debugf("Using cryptsetup at %q", cache.cryptsetup)
	})

	return cache.cryptsetup, cache.err
}

// cryptsetup checks that cryptsetup is available in the location
// specified in the configuration file, falling back to the build time
// value if necessary.
//
// This function is the test-friendly version of Cryptsetup above.
func cryptsetup(cfgpath string) (string, error) {
	// this is the value determined at build time; if it's empty,
	// cryptsetup was not available for this platform at build time.
	if buildcfg.CRYPTSETUP_PATH == "" {
		return "", errCryptsetupNotFound
	}

	cfg := singularity.FileConfig{}
	if err := config.Parser(cfgpath, &cfg); err != nil {
		return "", errors.Wrap(err, "unable to parse singularity configuration file")
	}

	path := cfg.CryptsetupPath

	if path == "" {
		if buildcfg.CRYPTSETUP_PATH == "" {
			return "", errors.New("unable to obtain path to cryptsetup program")
		}

		path = buildcfg.CRYPTSETUP_PATH
	} else {
		switch fi, err := os.Stat(path); {
		case err != nil:
			return "", errors.Wrapf(err, "unable to stat %s", path)

		case fi.IsDir():
			// configuration entry is a directory, append binary
			// name
			path = filepath.Join(path, "cryptsetup")
		}
	}

	// at this point we have an absolute path one way or the other,
	// use exec.LookPath to verify it's an executable.
	return exec.LookPath(path)
}
