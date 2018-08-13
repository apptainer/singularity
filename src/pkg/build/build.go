// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/singularityware/singularity/src/pkg/build/assemblers"
	"github.com/singularityware/singularity/src/pkg/build/sources"
	"github.com/singularityware/singularity/src/pkg/build/types"
	"github.com/singularityware/singularity/src/pkg/buildcfg"
	"github.com/singularityware/singularity/src/pkg/sylog"
	syexec "github.com/singularityware/singularity/src/pkg/util/exec"
	"github.com/singularityware/singularity/src/runtime/engines/common/config"
	"github.com/singularityware/singularity/src/runtime/engines/common/oci"
	"github.com/singularityware/singularity/src/runtime/engines/imgbuild"
)

// Build is an abstracted way to look at the entire build process.
// For example calling NewBuild() will return this object.
// From there we can call Full() on this build object, which will:
// 		Call Bundle() to obtain all data needed to execute the specified build locally on the machine
// 		Execute all of a definition using AllSections()
// 		And finally call Assemble() to create our container image
type Build struct {
	// Location for container after build is complete
	dest string
	// Format of built container, e.g., SIF, sandbox
	format string
	// If sections of the definition were run on container
	ranSections bool
	// Gets and Packs data needed to build a container into a Bundle from various sources
	c ConveyorPacker
	// Assembles a container from the information stored in a Bundle into various formats
	a Assembler
	// Intermediate stucture that encapsulates all information for the container, e.g., metadata, filesystems
	b *types.Bundle
	// Describes how a container is to be built, including actions to be run in the container to reach its final state
	d types.Definition
}

// NewBuild creates a new Build struct from a spec (URI, definition file, etc...)
func NewBuild(spec, dest, format string) (*Build, error) {
	def, err := makeDef(spec)
	if err != nil {
		return nil, fmt.Errorf("unable to parse spec %v: %v", spec, err)
	}

	return newBuild(def, dest, format)
}

// NewBuildJSON creates a new build struct from a JSON byte slice
func NewBuildJSON(r io.Reader, dest, format string) (*Build, error) {
	def, err := types.NewDefinitionFromJSON(r)
	if err != nil {
		return nil, fmt.Errorf("unable to parse JSON: %v", err)
	}

	return newBuild(def, dest, format)
}

func newBuild(d types.Definition, dest, format string) (*Build, error) {
	b := &Build{
		dest: dest,
		d:    d,
		b:    nil,
	}

	if c, err := getcp(b.d); err == nil {
		b.c = c
	} else {
		return nil, fmt.Errorf("unable to get conveyorpacker: %s", err)
	}

	switch format {
	case "sandbox":
		b.a = &assemblers.SandboxAssembler{}
	case "sif":
		b.a = &assemblers.SIFAssembler{}
	default:
		return nil, fmt.Errorf("unrecognized output format %s", format)
	}

	return b, nil
}

// Full runs a standard build from start to finish
func (b *Build) Full() error {
	sylog.Debugf("Creating bundle")
	if _, err := b.Bundle(); err != nil {
		return err
	}

	sylog.Debugf("Executing all sections of definition")
	if err := b.AllSections(); err != nil {
		return err
	}

	sylog.Debugf("Calling assembler")
	if err := b.Assemble(b.dest); err != nil {
		return err
	}

	return nil
}

// WithoutSections runs the build without running any section
func (b *Build) WithoutSections() error {
	if _, err := b.Bundle(); err != nil {
		return err
	}

	if err := b.Assemble(b.dest); err != nil {
		return err
	}

	return nil
}

// WithSections runs a build but only runs the specified sections
func (b *Build) WithSections(sections []string) error {
	if _, err := b.Bundle(); err != nil {
		return err
	}

	if err := b.Sections(sections); err != nil {
		return err
	}

	if err := b.Assemble(b.dest); err != nil {
		return err
	}

	return nil
}

// AllSections runs all the sections in the definition
func (b *Build) AllSections() error {
	if syscall.Getuid() == 0 && hasScripts(b.d) {
		if err := b.runScripts(); err != nil {
			return fmt.Errorf("unable to run scripts: %v", err)
		}
	} else if hasScripts(b.d) {
		sylog.Warningf("Attempted to build with scripts as non-root user, skipping...\n")
	}

	return nil
}

// Sections runs the list of sections specified by name in s
func (b *Build) Sections(s []string) error {

	return fmt.Errorf("sections is unimplemented")
}

// hasScripts returns true if build definition is requesting to run scripts in image
func hasScripts(def types.Definition) bool {
	return def.BuildData.Post != "" || def.BuildData.Pre != "" || def.BuildData.Setup != ""
}

// runScripts runs %pre %post %setup scripts in the bundle using the imgbuild engine
func (b *Build) runScripts() error {
	env := []string{"SINGULARITY_MESSAGELEVEL=" + string(sylog.GetLevel()), "SRUNTIME=" + imgbuild.Name}
	wrapper := filepath.Join(buildcfg.SBINDIR, "/wrapper")
	progname := []string{"singularity image-build"}

	engineConfig := &imgbuild.EngineConfig{
		Bundle: *b.b,
	}
	ociConfig := &oci.Config{}

	config := &config.Common{
		EngineName:   imgbuild.Name,
		ContainerID:  "image-build",
		OciConfig:    ociConfig,
		EngineConfig: engineConfig,
	}

	configData, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config.Common: %s", err)
	}

	// Set PIPE_EXEC_FD
	pipefd, err := syexec.SetPipe(configData)
	if err != nil {
		return fmt.Errorf("failed to set PIPE_EXEC_FD: %v", err)
	}

	env = append(env, pipefd)

	// Create os/exec.Command to run wrapper and return control once finished
	wrapperCmd := &exec.Cmd{
		Path:   wrapper,
		Args:   progname,
		Env:    env,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	if err := wrapperCmd.Start(); err != nil {
		return fmt.Errorf("failed to start wrapper proc: %v", err)
	}
	if err := wrapperCmd.Wait(); err != nil {
		return fmt.Errorf("wrapper proc failed: %v", err)
	}

	return nil
}

// Bundle creates the bundle using the ConveyorPacker and returns it. If this
// function is called multiple times it will return the already created Bundle
func (b *Build) Bundle() (*types.Bundle, error) {
	if b.b != nil {
		return b.b, nil
	}

	if err := b.c.Get(b.d); err != nil {
		return nil, fmt.Errorf("conveyor failed to get: %v", err)
	}

	bundle, err := b.c.Pack()
	if err != nil {
		return nil, fmt.Errorf("packer failed to pack: %v", err)
	}

	b.b = bundle
	return b.b, nil
}

func getcp(def types.Definition) (ConveyorPacker, error) {
	switch def.Header["bootstrap"] {
	case "shub":
		return &sources.ShubConveyorPacker{}, nil
	case "docker", "docker-archive", "docker-daemon", "oci", "oci-archive":
		return &sources.OCIConveyorPacker{}, nil
	case "busybox":
		return &sources.BusyBoxConveyorPacker{}, nil
	case "debootstrap":
		return &sources.DebootstrapConveyorPacker{}, nil
	case "arch":
		return &sources.ArchConveyorPacker{}, nil
	case "localimage":
		return &sources.LocalConveyorPacker{}, nil
	default:
		return nil, fmt.Errorf("invalid build source %s", def.Header["bootstrap"])
	}
}

// makeDef gets a definition object from a spec
func makeDef(spec string) (types.Definition, error) {
	var def types.Definition

	if ok, err := IsValidURI(spec); ok && err == nil {
		// URI passed as spec
		def, err = types.NewDefinitionFromURI(spec)
		if err != nil {
			return def, fmt.Errorf("unable to parse URI %s: %v", spec, err)
		}

	} else if ok, err := types.IsValidDefinition(spec); ok && err == nil {
		// Non-URI passed as spec, check is its a definition
		defFile, err := os.Open(spec)
		if err != nil {
			return def, fmt.Errorf("unable to open file %s: %v", spec, err)
		}
		defer defFile.Close()

		def, err = types.ParseDefinitionFile(defFile)
		if err != nil {
			return def, fmt.Errorf("failed to parse definition file %s: %v", spec, err)
		}
	} else if _, err := os.Stat(spec); err == nil {
		//local image or sandbox, make sure it exists on filesystem
		def = types.Definition{
			Header: map[string]string{
				"bootstrap": "localimage",
				"from":      spec,
			},
		}
	} else {
		return def, fmt.Errorf("unable to build from %s: %v", spec, err)
	}

	return def, nil
}

// MakeDef gets a definition object from a spec
func MakeDef(spec string) (types.Definition, error) {
	return makeDef(spec)
}

// Assemble assembles the bundle to the specified path
func (b *Build) Assemble(path string) error {
	return b.a.Assemble(b.b, path)
}
