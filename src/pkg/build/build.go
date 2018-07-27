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

	"github.com/singularityware/singularity/src/pkg/buildcfg"
	"github.com/singularityware/singularity/src/pkg/sylog"
	syexec "github.com/singularityware/singularity/src/pkg/util/exec"
	"github.com/singularityware/singularity/src/runtime/engines/common/config"
	"github.com/singularityware/singularity/src/runtime/engines/common/oci"
)

// EngineName is the engine name of the imgbuild engine
const EngineName = "imgbuild"

// EngineConfig is the engineConfig for the imgbuild Engine
type EngineConfig struct {
	Bundle
}

// MarshalJSON implements json.Marshaler interface
func (c *EngineConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.Bundle)
}

// UnmarshalJSON implements json.Unmarshaler interface
func (c *EngineConfig) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &c.Bundle)
}

// Build is a
type Build struct {
	dest   string
	format string
	c      ConveyorPacker
	a      Assembler
	b      *Bundle
	d      Definition
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
	def, err := NewDefinitionFromJSON(r)
	if err != nil {
		return nil, fmt.Errorf("unable to parse JSON: %v", err)
	}

	return newBuild(def, dest, format)
}

func newBuild(d Definition, dest, format string) (*Build, error) {
	b := &Build{
		dest: dest,
		d:    d,
		b:    nil,
	}

	switch format {
	case "sandbox":
		b.a = &SandboxAssembler{}
	case "sif":
		b.a = &SIFAssembler{}
	default:
		return nil, fmt.Errorf("unrecognized output format %s", format)
	}

	if c, err := getcp(b.d); err != nil {
		b.c = c
	} else {
		return nil, fmt.Errorf("unable to get conveyorpacker: %s", err)
	}

	return b, nil
}

// Full runs a standard build from start to finish
func (b *Build) Full(path string) error {

	return nil
}

// WithoutSections runs the build without running any section
func (b *Build) WithoutSections(path string) error {

	return nil
}

// AllSections runs all the sections in the definition
func (b *Build) AllSections() error {
	if syscall.Getuid() == 0 && hasScripts(b.d) {
		if err := b.runScripts(); err != nil {
			return fmt.Errorf("unable to run scripts: %v", err)
		}
	}

	return nil
}

// Sections runs the list of sections specified by name in s
func (b *Build) Sections(s []string) error {

	return nil
}

// hasScripts returns true if build definition is requesting to run scripts in image
func hasScripts(def Definition) bool {
	return def.BuildData.Post != "" || def.BuildData.Pre != "" || def.BuildData.Setup != ""
}

// runScripts runs %pre %post %setup scripts in the bundle using the imgbuild engine
func (b *Build) runScripts() error {
	env := []string{"SINGULARITY_MESSAGELEVEL=" + string(sylog.GetLevel()), "SRUNTIME=" + EngineName}
	wrapper := filepath.Join(buildcfg.SBINDIR, "/wrapper")
	progname := []string{"singularity image-build"}

	engineConfig := &EngineConfig{
		Bundle: *b.b,
	}
	ociConfig := &oci.Config{}

	config := &config.Common{
		EngineName:   EngineName,
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
}

// Bundle creates the bundle using the ConveyorPacker and returns it. If this
// function is called multiple times it will return the already created Bundle
func (b *Build) Bundle() (*Bundle, error) {
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

func getcp(def Definition) (ConveyorPacker, error) {
	switch def.Header["bootstrap"] {
	case "shub":
		return &ShubConveyorPacker{}, nil
	case "docker", "docker-archive", "docker-daemon", "oci", "oci-archive":
		return &OCIConveyorPacker{}, nil
	case "busybox":
		return &BusyBoxConveyorPacker{}, nil
	case "debootstrap":
		return &DebootstrapConveyorPacker{}, nil
	case "arch":
		return &ArchConveyorPacker{}, nil
	case "localimage":
		return &LocalConveyorPacker{}, nil
	default:
		return nil, fmt.Errorf("invalid build source %s", def.Header["bootstrap"])
	}
}

// makeDef gets a definition object from a spec
func makeDef(spec string) (Definition, error) {
	var def Definition

	if ok, err := IsValidURI(spec); ok && err == nil {
		// URI passed as arg[1]
		def, err = NewDefinitionFromURI(spec)
		if err != nil {
			return def, fmt.Errorf("unable to parse URI %s: %v", spec, err)
		}

	} else if ok, err := IsValidDefinition(spec); ok && err == nil {
		// Non-URI passed as arg[1]
		defFile, err := os.Open(spec)
		if err != nil {
			return def, fmt.Errorf("unable to open file %s: %v", spec, err)
		}
		defer defFile.Close()

		def, err = ParseDefinitionFile(defFile)
		if err != nil {
			return def, fmt.Errorf("failed to parse definition file %s: %v", spec, err)
		}
	} else if _, err := os.Stat(spec); err == nil {
		//local image or sandbox
		def = Definition{
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
