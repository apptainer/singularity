// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity/internal/pkg/build/apps"
	"github.com/sylabs/singularity/internal/pkg/build/assemblers"
	"github.com/sylabs/singularity/internal/pkg/build/copy"
	"github.com/sylabs/singularity/internal/pkg/build/sources"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config/oci"
	imgbuildConfig "github.com/sylabs/singularity/internal/pkg/runtime/engines/imgbuild/config"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	syexec "github.com/sylabs/singularity/internal/pkg/util/exec"
	"github.com/sylabs/singularity/internal/pkg/util/uri"
	"github.com/sylabs/singularity/pkg/build/types"
	"github.com/sylabs/singularity/pkg/build/types/parser"
	"github.com/sylabs/singularity/pkg/image"
)

// Build is an abstracted way to look at the entire build process.
// For example calling NewBuild() will return this object.
// From there we can call Full() on this build object, which will:
// 		Call Bundle() to obtain all data needed to execute the specified build locally on the machine
// 		Execute all of a definition using AllSections()
// 		And finally call Assemble() to create our container image
type Build struct {
	// stages of the build
	stages []stage
	// Conf contains cross stage build configuration
	Conf Config
}

// Config defines how build is executed, including things like where final image is written.
type Config struct {
	// Dest is the location for container after build is complete
	Dest string
	// Format is the format of built container, e.g., SIF, sandbox
	Format string
	// NoCleanUp allows a user to prevent a bundle from being cleaned up after a failed build
	// useful for debugging
	NoCleanUp bool
	// Opts for bundles
	Opts types.Options
}

// NewBuild creates a new Build struct from a spec (URI, definition file, etc...)
func NewBuild(spec string, conf Config, allowUnauthenticatedBuild, localVerifyBuild bool) (*Build, error) {
	def, err := makeDef(spec, false)
	if err != nil {
		return nil, fmt.Errorf("unable to parse spec %v: %v", spec, err)
	}

	return newBuild([]types.Definition{def}, conf, allowUnauthenticatedBuild, localVerifyBuild)
}

// NewBuildJSON creates a new build struct from a JSON byte slice
func NewBuildJSON(r io.Reader, conf Config) (*Build, error) {
	def, err := types.NewDefinitionFromJSON(r)
	if err != nil {
		return nil, fmt.Errorf("unable to parse JSON: %v", err)
	}

	return newBuild([]types.Definition{def}, conf, false, false)
}

// New creates a new build struct form a slice of definitions
func New(defs []types.Definition, conf Config, allowUnauthenticatedBuild, localVerifyBuild bool) (*Build, error) {
	return newBuild(defs, conf, allowUnauthenticatedBuild, localVerifyBuild)
}

func newBuild(defs []types.Definition, conf Config, allowUnauthenticatedBuild, localVerifyBuild bool) (*Build, error) {
	var err error

	syscall.Umask(0002)

	// always build a sandbox if updating an existing sandbox
	if conf.Opts.Update {
		conf.Format = "sandbox"
	}

	b := &Build{
		Conf: conf,
	}

	// create stages
	for _, d := range defs {
		// verify every definition has a header if there are multiple stages
		if d.Header == nil {
			return nil, fmt.Errorf("multiple stages detected, all must have headers")
		}

		s := stage{}
		s.b, err = types.NewBundle(conf.Opts.TmpDir, "sbuild")
		if err != nil {
			return nil, err
		}
		s.name = d.Header["stage"]
		s.b.Recipe = d

		s.b.Opts = conf.Opts
		// dont need to get cp if we're skipping bootstrap
		if !conf.Opts.Update || conf.Opts.Force {
			if c, err := getcp(d, conf.Opts.LibraryURL, conf.Opts.LibraryAuthToken, allowUnauthenticatedBuild, localVerifyBuild); err == nil {
				s.c = c
			} else {
				return nil, fmt.Errorf("unable to get conveyorpacker: %s", err)
			}
		}

		b.stages = append(b.stages, s)
	}

	// only need an assembler for last stage
	lastStageIndex := len(defs) - 1
	switch conf.Format {
	case "sandbox":
		b.stages[lastStageIndex].a = &assemblers.SandboxAssembler{}
	case "sif":
		b.stages[lastStageIndex].a = &assemblers.SIFAssembler{}
	default:
		return nil, fmt.Errorf("unrecognized output format %s", conf.Format)
	}

	return b, nil
}

// cleanUp removes remnants of build from file system unless NoCleanUp is specified
func (b Build) cleanUp() {
	var bundlePaths []string
	for _, s := range b.stages {
		bundlePaths = append(bundlePaths, s.b.Path)
	}

	if b.Conf.NoCleanUp {
		sylog.Infof("Build performed with no clean up option, build bundle(s) located at: %v", bundlePaths)
		return
	}

	for _, path := range bundlePaths {
		os.RemoveAll(path)
	}
	sylog.Debugf("Build bundle(s) cleaned: %v", bundlePaths)

}

// Full runs a standard build from start to finish
func (b *Build) Full() error {
	sylog.Infof("Starting build...")

	// monitor build for termination signal and clean up
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		b.cleanUp()
		os.Exit(1)
	}()
	// clean up build normally
	defer b.cleanUp()

	// build each stage one after the other
	for i, stage := range b.stages {
		if err := stage.runPreScript(); err != nil {
			return err
		}

		// only update last stage if specified
		update := stage.b.Opts.Update && !stage.b.Opts.Force && i == len(b.stages)-1
		if update {
			// updating, extract dest container to bundle
			sylog.Infof("Building into existing container: %s", b.Conf.Dest)
			p, err := sources.GetLocalPacker(b.Conf.Dest, stage.b)
			if err != nil {
				return err
			}

			_, err = p.Pack()
			if err != nil {
				return err
			}
		} else {
			// regular build or force, start build from scratch
			if err := stage.c.Get(stage.b); err != nil {
				return fmt.Errorf("conveyor failed to get: %v", err)
			}

			_, err := stage.c.Pack()
			if err != nil {
				return fmt.Errorf("packer failed to pack: %v", err)
			}
		}

		// create apps in bundle
		a := apps.New()
		for k, v := range stage.b.Recipe.CustomData {
			a.HandleSection(k, v)
		}

		a.HandleBundle(stage.b)
		stage.b.Recipe.BuildData.Post.Script += a.HandlePost()

		if stage.b.RunSection("files") {
			if err := stage.copyFiles(b); err != nil {
				return fmt.Errorf("unable to copy files a stage to container fs: %v", err)
			}
		}

		if engineRequired(stage.b.Recipe) {
			if err := runBuildEngine(stage.b); err != nil {
				return fmt.Errorf("while running engine: %v", err)
			}
		}

		sylog.Debugf("Inserting Metadata")
		if err := stage.insertMetadata(); err != nil {
			return fmt.Errorf("while inserting metadata to bundle: %v", err)
		}
	}

	sylog.Debugf("Calling assembler")
	if err := b.stages[len(b.stages)-1].Assemble(b.Conf.Dest); err != nil {
		return err
	}

	sylog.Infof("Build complete: %s", b.Conf.Dest)
	return nil
}

// engineRequired returns true if build definition is requesting to run scripts or copy files
func engineRequired(def types.Definition) bool {
	return def.BuildData.Post.Script != "" || def.BuildData.Setup.Script != "" || def.BuildData.Test.Script != "" || len(def.BuildData.Files) != 0
}

// runBuildEngine creates an imgbuild engine and creates a container out of our bundle in order to execute %post %setup scripts in the bundle
func runBuildEngine(b *types.Bundle) error {
	if syscall.Getuid() != 0 {
		return fmt.Errorf("Attempted to build with scripts as non-root user")
	}

	sylog.Debugf("Starting build engine")
	env := []string{sylog.GetEnvVar()}
	starter := filepath.Join(buildcfg.LIBEXECDIR, "/singularity/bin/starter")
	progname := []string{"singularity image-build"}
	ociConfig := &oci.Config{}

	engineConfig := &imgbuildConfig.EngineConfig{
		Bundle:    *b,
		OciConfig: ociConfig,
	}

	// surface build specific environment variables for scripts
	sRootfs := "SINGULARITY_ROOTFS=" + b.Rootfs()
	sEnvironment := "SINGULARITY_ENVIRONMENT=" + "/.singularity.d/env/91-environment.sh"

	ociConfig.Process = &specs.Process{}
	ociConfig.Process.Env = append(os.Environ(), sRootfs, sEnvironment)

	config := &config.Common{
		EngineName:   imgbuildConfig.Name,
		ContainerID:  "image-build",
		EngineConfig: engineConfig,
	}

	configData, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config.Common: %s", err)
	}

	starterCmd, err := syexec.PipeCommand(starter, progname, env, configData)
	if err != nil {
		return fmt.Errorf("failed to create cmd type: %v", err)
	}

	starterCmd.Stdout = os.Stdout
	starterCmd.Stderr = os.Stderr

	return starterCmd.Run()
}

func getcp(def types.Definition, libraryURL, authToken string, unauthenticatedBuild, localVerifyBuild bool) (ConveyorPacker, error) {
	switch def.Header["bootstrap"] {
	case "library":
		return &sources.LibraryConveyorPacker{
			AllowUnauthenticated: unauthenticatedBuild,
			LocalVerify:          localVerifyBuild,
		}, nil
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
	case "yum":
		return &sources.YumConveyorPacker{}, nil
	case "zypper":
		return &sources.ZypperConveyorPacker{}, nil
	case "scratch":
		return &sources.ScratchConveyorPacker{}, nil
	case "":
		return nil, fmt.Errorf("no bootstrap specification found")
	default:
		return nil, fmt.Errorf("invalid build source %s", def.Header["bootstrap"])
	}
}

// MakeDef gets a definition object from a spec
func MakeDef(spec string, remote bool) (types.Definition, error) {
	return makeDef(spec, remote)
}

// makeDef gets a definition object from a spec
func makeDef(spec string, remote bool) (types.Definition, error) {
	if ok, err := uri.IsValid(spec); ok && err == nil {
		// URI passed as spec
		return types.NewDefinitionFromURI(spec)
	}

	// Check if spec is an image/sandbox
	if _, err := image.Init(spec, false); err == nil {
		return types.NewDefinitionFromURI("localimage" + "://" + spec)
	}

	// default to reading file as definition
	defFile, err := os.Open(spec)
	if err != nil {
		return types.Definition{}, fmt.Errorf("unable to open file %s: %v", spec, err)
	}
	defer defFile.Close()

	// must be root to build from a definition
	if os.Getuid() != 0 && !remote {
		return types.Definition{}, fmt.Errorf("you must be the root user to build from a definition file")
	}

	d, err := parser.ParseDefinitionFile(defFile)
	if err != nil {
		return types.Definition{}, fmt.Errorf("while parsing definition: %s: %v", spec, err)
	}

	return d, nil
}

// MakeAllDefs gets a definition slice from a spec
func MakeAllDefs(spec string, remote bool) ([]types.Definition, error) {
	return makeAllDefs(spec, remote)
}

// makeAllDef gets a definition object from a spec
func makeAllDefs(spec string, remote bool) ([]types.Definition, error) {
	if ok, err := uri.IsValid(spec); ok && err == nil {
		// URI passed as spec
		d, err := types.NewDefinitionFromURI(spec)
		return []types.Definition{d}, err
	}

	// Check if spec is an image/sandbox
	if _, err := image.Init(spec, false); err == nil {
		d, err := types.NewDefinitionFromURI("localimage" + "://" + spec)
		return []types.Definition{d}, err
	}

	// default to reading file as definition
	defFile, err := os.Open(spec)
	if err != nil {
		return nil, fmt.Errorf("unable to open file %s: %v", spec, err)
	}
	defer defFile.Close()

	// must be root to build from a definition
	if os.Getuid() != 0 && !remote {
		return nil, fmt.Errorf("you must be the root user to build from a definition file")
	}

	d, err := parser.All(defFile)
	if err != nil {
		return nil, fmt.Errorf("while parsing definition: %s: %v", spec, err)
	}

	return d, nil
}

func (b *Build) findStageIndex(name string) (int, error) {
	for i, s := range b.stages {
		if name == s.name {
			return i, nil
		}
	}

	return -1, fmt.Errorf("stage %s was not found", name)
}

func (s *stage) copyFiles(b *Build) error {
	def := s.b.Recipe
	for _, f := range def.BuildData.Files {
		if f.Args == "" {
			continue
		}
		args := strings.Fields(f.Args)
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
			transfer.Src = filepath.Join(b.stages[stageIndex].b.Rootfs(), transfer.Src)
			transfer.Dst = filepath.Join(s.b.Rootfs(), transfer.Dst)
			sylog.Infof("Copying %v to %v", transfer.Src, transfer.Dst)
			if err := copy.Copy(transfer.Src, transfer.Dst); err != nil {
				return err
			}
		}
	}

	return nil
}
