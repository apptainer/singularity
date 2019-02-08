// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"

	specs "github.com/opencontainers/runtime-spec/specs-go"

	"github.com/opencontainers/runtime-tools/generate"
)

// RootFs is the default root path for OCI bundle
type RootFs string

// Path returns the root path inside bundle
func (r RootFs) Path() string {
	return filepath.Join(string(r), "rootfs")
}

// CreateBundle generates a minimal OCI bundle directory
// with the provided OCI configuration or a default one
// if there is no configuration
func CreateBundle(bundlePath string, config *specs.Spec) error {
	var err error
	var g generate.Generator

	oldumask := syscall.Umask(0)
	defer syscall.Umask(oldumask)

	rootFsDir := RootFs(bundlePath).Path()
	if err := os.MkdirAll(rootFsDir, 0700); err != nil {
		return fmt.Errorf("failed to create %s: %s", rootFsDir, err)
	}
	defer func() {
		if err != nil {
			DeleteBundle(bundlePath)
		}
	}()

	if config == nil {
		// generate and write config.json in bundle
		g, err = generate.New(runtime.GOOS)
		if err != nil {
			return fmt.Errorf("failed to generate OCI config: %s", err)
		}
	} else {
		g = generate.Generator{
			Config:       config,
			HostSpecific: true,
		}
	}
	g.SetRootPath(rootFsDir)
	options := generate.ExportOptions{}
	return g.SaveToFile(filepath.Join(bundlePath, "config.json"), options)
}

// DeleteBundle ...
func DeleteBundle(bundlePath string) error {
	return os.RemoveAll(bundlePath)
}
