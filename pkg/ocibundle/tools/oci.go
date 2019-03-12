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

// Volumes is the parent volumes path
type Volumes string

// Path returns the volumes path inside bundle
func (v Volumes) Path() string {
	return filepath.Join(string(v), "volumes")
}

// Config is the OCI configuration path
type Config string

// Path returns the OCI configuration path
func (c Config) Path() string {
	return filepath.Join(string(c), "config.json")
}

// RunScript is the default process argument
const RunScript = "/.singularity.d/actions/run"

// GenerateBundleConfig generates a minimal OCI bundle directory
// with the provided OCI configuration or a default one
// if there is no configuration
func GenerateBundleConfig(bundlePath string, config *specs.Spec) (*generate.Generator, error) {
	var err error
	var g generate.Generator

	oldumask := syscall.Umask(0)
	defer syscall.Umask(oldumask)

	rootFsDir := RootFs(bundlePath).Path()
	if err := os.MkdirAll(rootFsDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create %s: %s", rootFsDir, err)
	}
	volumesDir := Volumes(bundlePath).Path()
	if err := os.MkdirAll(volumesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create %s: %s", volumesDir, err)
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
			return nil, fmt.Errorf("failed to generate OCI config: %s", err)
		}
		g.SetProcessArgs([]string{RunScript})
	} else {
		g = generate.Generator{
			Config:       config,
			HostSpecific: true,
		}
	}
	g.SetRootPath(rootFsDir)
	return &g, nil
}

// SaveBundleConfig creates config.json in OCI bundle directory and
// saves OCI configuration
func SaveBundleConfig(bundlePath string, g *generate.Generator) error {
	options := generate.ExportOptions{}
	return g.SaveToFile(Config(bundlePath).Path(), options)
}

// DeleteBundle deletes bundle directory
func DeleteBundle(bundlePath string) error {
	if err := os.RemoveAll(Volumes(bundlePath).Path()); err != nil {
		return fmt.Errorf("failed to delete volumes directory: %s", err)
	}
	if err := os.Remove(RootFs(bundlePath).Path()); err != nil {
		return fmt.Errorf("failed to delete rootfs directory: %s", err)
	}
	if err := os.Remove(Config(bundlePath).Path()); err != nil {
		return fmt.Errorf("failed to delete config.json file: %s", err)
	}
	if err := os.Remove(bundlePath); err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to delete bundle %s directory: %s", bundlePath, err)
	}
	return nil
}
