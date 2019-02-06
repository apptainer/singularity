package tools

import (
	"os"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"

	"github.com/opencontainers/runtime-tools/generate"
)

// RootFs is the default root path for OCI bundle
type RootFs string

// Path returns the root path inside bundle
func (r RootFs) Path() string {
	return filepath.Join(string(r), "rootfs")
}

// CreateBundle ...
func CreateBundle(bundlePath string, config *specs.Spec) error {
	var err error
	var g generate.Generator

	oldumask := syscall.Umask(0)
	syscall.Umask(oldumask)

	rootFsDir := RootFs(bundlePath).Path()
	if err := os.MkdirAll(rootFsDir, 0700); err != nil {
		return err
	}

	if config == nil {
		// generate and write config.json in bundle
		g, err = generate.New(runtime.GOOS)
		if err != nil {
			return err
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
