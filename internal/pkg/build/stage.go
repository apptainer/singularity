// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/build/files"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/pkg/build/types"
	"github.com/sylabs/singularity/pkg/sylog"
)

// stage represents the process of constructing a root filesystem.
type stage struct {
	// name of the stage.
	name string
	// c Gets and Packs data needed to build a container into a Bundle from various sources.
	c ConveyorPacker
	// a Assembles a container from the information stored in a Bundle into various formats.
	a Assembler
	// b is an intermediate structure that encapsulates all information for the container, e.g., metadata, filesystems.
	b *types.Bundle
}

const (
	sLabelsPath  = "/.build.labels"
	sEnvironment = "SINGULARITY_ENVIRONMENT=/.singularity.d/env/91-environment.sh"
	sLabels      = "SINGULARITY_LABELS=" + sLabelsPath
)

// Assemble assembles the bundle to the specified path.
func (s *stage) Assemble(path string) error {
	return s.a.Assemble(s.b, path)
}

// runSetupScript executes the stage's pre script on host.
func (s *stage) runSectionScript(name string, script types.Script) error {
	if s.b.RunSection(name) && script.Script != "" {
		if syscall.Getuid() != 0 {
			return fmt.Errorf("attempted to build with scripts as non-root user or without --fakeroot")
		}

		sRootfs := "SINGULARITY_ROOTFS=" + s.b.RootfsPath

		scriptPath := filepath.Join(s.b.TmpDir, name)
		if err := createScript(scriptPath, []byte(script.Script)); err != nil {
			return fmt.Errorf("while creating %s script: %s", name, err)
		}
		defer os.Remove(scriptPath)

		args, err := getSectionScriptArgs(name, scriptPath, script)
		if err != nil {
			return fmt.Errorf("while processing section %%%s arguments: %s", name, err)
		}

		// Run script section here
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, sEnvironment, sRootfs)

		sylog.Infof("Running %s scriptlet", name)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to run %%%s script: %v", name, err)
		}
	}
	return nil
}

func (s *stage) runPostScript(configFile, sessionResolv, sessionHosts string) error {
	if s.b.Recipe.BuildData.Post.Script != "" {
		cmdArgs := []string{"-s", "-c", configFile, "exec", "--pwd", "/", "--writable"}
		cmdArgs = append(cmdArgs, "--cleanenv", "--env", sEnvironment, "--env", sLabels)

		if sessionResolv != "" {
			cmdArgs = append(cmdArgs, "-B", sessionResolv+":/etc/resolv.conf")
		}
		if sessionHosts != "" {
			cmdArgs = append(cmdArgs, "-B", sessionHosts+":/etc/hosts")
		}

		script := s.b.Recipe.BuildData.Post
		scriptPath := filepath.Join(s.b.RootfsPath, ".post.script")
		if err := createScript(scriptPath, []byte(script.Script)); err != nil {
			return fmt.Errorf("while creating post script: %s", err)
		}
		defer os.Remove(scriptPath)

		args, err := getSectionScriptArgs("post", "/.post.script", script)
		if err != nil {
			return fmt.Errorf("while processing section %%post arguments: %s", err)
		}

		exe := filepath.Join(buildcfg.BINDIR, "singularity")

		cmdArgs = append(cmdArgs, s.b.RootfsPath)
		cmdArgs = append(cmdArgs, args...)
		cmd := exec.Command(exe, cmdArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = "/"
		cmd.Env = currentEnvNoSingularity()

		sylog.Infof("Running post scriptlet")
		return cmd.Run()
	}
	return nil
}

func (s *stage) runTestScript(configFile, sessionResolv, sessionHosts string) error {
	if !s.b.Opts.NoTest && s.b.Recipe.BuildData.Test.Script != "" {
		cmdArgs := []string{"-s", "-c", configFile, "test", "--pwd", "/"}

		if sessionResolv != "" {
			cmdArgs = append(cmdArgs, "-B", sessionResolv+":/etc/resolv.conf")
		}
		if sessionHosts != "" {
			cmdArgs = append(cmdArgs, "-B", sessionHosts+":/etc/hosts")
		}

		exe := filepath.Join(buildcfg.BINDIR, "singularity")

		cmdArgs = append(cmdArgs, s.b.RootfsPath)
		cmd := exec.Command(exe, cmdArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = "/"
		cmd.Env = currentEnvNoSingularity()

		sylog.Infof("Running testscript")
		return cmd.Run()
	}
	return nil
}

func (s *stage) copyFilesFrom(b *Build) error {
	def := s.b.Recipe
	for _, f := range def.BuildData.Files {
		// Trim comments from args
		cleanArgs := strings.Split(f.Args, "#")[0]
		if cleanArgs == "" {
			continue
		}

		args := strings.Fields(cleanArgs)
		if len(args) != 2 {
			continue
		}

		stageIndex, err := b.findStageIndex(args[1])
		if err != nil {
			return err
		}

		sylog.Debugf("Copying files from stage: %s", args[1])

		// iterate through filetransfers
		for _, transfer := range f.Files {
			// sanity
			if transfer.Src == "" {
				sylog.Warningf("Attempt to copy file with no name, skipping.")
				continue
			}
			// dest = source if not specified
			if transfer.Dst == "" {
				transfer.Dst = transfer.Src
			}

			// copy each file into bundle rootfs
			// prepend appropriate bundle path to supplied paths
			// copying between stages should not follow symlinks
			transfer.Src = files.AddPrefix(b.stages[stageIndex].b.RootfsPath, transfer.Src)
			transfer.Dst = files.AddPrefix(s.b.RootfsPath, transfer.Dst)
			sylog.Infof("Copying %v to %v", transfer.Src, transfer.Dst)
			if err := files.Copy(transfer.Src, transfer.Dst, false); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *stage) copyFiles() error {
	def := s.b.Recipe
	filesSection := types.Files{}
	for _, f := range def.BuildData.Files {
		// Trim comments from args
		cleanArgs := strings.Split(f.Args, "#")[0]
		if cleanArgs == "" {
			filesSection.Files = append(filesSection.Files, f.Files...)
		}
	}
	// iterate through filetransfers
	for _, transfer := range filesSection.Files {
		// sanity
		if transfer.Src == "" {
			sylog.Warningf("Attempt to copy file with no name, skipping.")
			continue
		}
		// dest = source if not specified
		if transfer.Dst == "" {
			transfer.Dst = transfer.Src
		}
		// copy each file into bundle rootfs
		// copying from host to container should follow symlinks
		transfer.Dst = files.AddPrefix(s.b.RootfsPath, transfer.Dst)
		sylog.Infof("Copying %v to %v", transfer.Src, transfer.Dst)
		if err := files.Copy(transfer.Src, transfer.Dst, true); err != nil {
			return err
		}
	}

	return nil
}
