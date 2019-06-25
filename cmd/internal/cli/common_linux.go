// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/opencontainers/runtime-tools/generate"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config/oci"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/exec"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/pkg/image"
	"github.com/sylabs/singularity/pkg/image/unpacker"
	singularityConfig "github.com/sylabs/singularity/pkg/runtime/engines/singularity/config"
)

// singularityConfPath holds the path to singularity.conf file
var singularityConfPath = filepath.Join(buildcfg.SYSCONFDIR, "singularity/singularity.conf")

// convertSquashfsToSandbox take a single image file in SIF or squashfs format
// to convert it to a sandbox image and returns the path to the sandbox.
func convertSquashfsToSandbox(filename string, unsquashfsPath string) (string, error) {
	img, err := image.Init(filename, false)
	if err != nil {
		return "", fmt.Errorf("could not open image %s: %s", filename, err)
	}
	defer img.File.Close()

	if !img.HasRootFs() {
		return "", fmt.Errorf("no root filesystem found in %s", filename)
	}

	// squashfs only
	if img.Partitions[0].Type != image.SQUASHFS {
		return "", fmt.Errorf("not a squashfs root filesystem")
	}

	// create a reader for rootfs partition
	reader, err := image.NewPartitionReader(img, "", 0)
	if err != nil {
		return "", fmt.Errorf("could not extract root filesystem: %s", err)
	}
	s := unpacker.NewSquashfs()
	if !s.HasUnsquashfs() && unsquashfsPath != "" {
		s.UnsquashfsPath = unsquashfsPath
	}

	// keep compatibility with v2
	tmpdir := os.Getenv("SINGULARITY_LOCALCACHEDIR")
	if tmpdir == "" {
		tmpdir = os.Getenv("SINGULARITY_CACHEDIR")
	}

	// create temporary sandbox
	dir, err := ioutil.TempDir(tmpdir, "rootfs-")
	if err != nil {
		return "", fmt.Errorf("could not create temporary sandbox: %s", err)
	}

	// extract root filesystem
	if err := s.ExtractAll(reader, dir); err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("root filesystem extraction failed: %s", err)
	}

	return dir, err
}

// getfileConfig returns singularity configuration read from singularity.conf
// file.
func getFileConfig() (*singularityConfig.FileConfig, error) {
	fileConfig := new(singularityConfig.FileConfig)

	if err := config.Parser(singularityConfPath, fileConfig); err != nil {
		return nil, fmt.Errorf("unable to parse %s file: %s", singularityConfPath, err)
	}
	return fileConfig, nil
}

// genericStarterCommand executes a single command inside container image with
// provided args and returns command output or any encountered error while
// executing the command.
func genericStarterCommand(procname, name, abspath string, args []string) ([]byte, error) {
	Env := []string{sylog.GetEnvVar()}

	engineConfig := singularityConfig.NewConfig()
	ociConfig := &oci.Config{}
	generator := generate.Generator{Config: &ociConfig.Spec}
	engineConfig.OciConfig = ociConfig

	generator.SetProcessArgs(args)
	generator.SetProcessCwd("/")
	engineConfig.SetImage(abspath)

	uid := uint32(os.Getuid())
	gid := uint32(os.Getgid())

	isPrivileged := uid == 0
	fileConfig, err := getFileConfig()
	if err != nil {
		return nil, fmt.Errorf("could not read configuration file: %s", err)
	}
	starterBinary, starterSuid := exec.LookStarterPath(!isPrivileged, fileConfig.AllowSetuid)

	// check if users can use setuid workflow
	if !isPrivileged {
		// setuid workflow not allowed, need to fallback with user namespace
		if !starterSuid || !fileConfig.AllowSetuid {
			sylog.Verbosef("Use unprivileged workflow with user namespace, setuid workflow disabled")
			if fs.IsFile(abspath) {
				// when running with user namespace, we can't use file
				// images so we need to convert it to a sandbox image
				// if it's possible. Conversion will fail if the image
				// is an ext3 filesystem
				sylog.Verbosef("Convert SIF file to sandbox...")
				unsquashfsPath := ""
				if fileConfig.MksquashfsPath != "" {
					d := filepath.Dir(fileConfig.MksquashfsPath)
					unsquashfsPath = filepath.Join(d, "unsquashfs")
				}
				img, err := convertSquashfsToSandbox(abspath, unsquashfsPath)
				if err != nil {
					return nil, fmt.Errorf("error while converting image %s: %s", abspath, err)
				}
				engineConfig.SetImage(img)
				engineConfig.SetDeleteImage(true)
			}
			generator.AddOrReplaceLinuxNamespace("user", "")
			generator.AddLinuxUIDMapping(uid, uid, 1)
			generator.AddLinuxGIDMapping(gid, gid, 1)
		}
	}

	cfg := &config.Common{
		EngineName:   singularityConfig.Name,
		ContainerID:  name,
		EngineConfig: engineConfig,
	}

	cmd, err := exec.StarterCommand(starterBinary, []string{procname}, Env, cfg)
	if err != nil {
		args := strings.Join(cmd.Args, ",")
		return nil, fmt.Errorf("unable to execute singularity with arguments %q: %s", args, err)
	}

	return cmd.Output()
}
