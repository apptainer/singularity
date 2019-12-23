// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/sylabs/singularity/pkg/util/fs/proc"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	uuid "github.com/satori/go.uuid"
	"github.com/sylabs/singularity/internal/pkg/build/apps"
	"github.com/sylabs/singularity/internal/pkg/build/assemblers"
	"github.com/sylabs/singularity/internal/pkg/build/files"
	"github.com/sylabs/singularity/internal/pkg/build/sources"
	"github.com/sylabs/singularity/internal/pkg/runtime/engine/config/oci"
	imgbuildConfig "github.com/sylabs/singularity/internal/pkg/runtime/engine/imgbuild/config"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/fs/squashfs"
	"github.com/sylabs/singularity/internal/pkg/util/starter"
	"github.com/sylabs/singularity/internal/pkg/util/uri"
	"github.com/sylabs/singularity/pkg/build/types"
	"github.com/sylabs/singularity/pkg/build/types/parser"
	"github.com/sylabs/singularity/pkg/image"
	"github.com/sylabs/singularity/pkg/image/packer"
	"github.com/sylabs/singularity/pkg/runtime/engine/config"
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
	// Conf contains cross stage build configuration.
	Conf Config
}

// Config defines how build is executed, including things like where final image is written.
type Config struct {
	// Dest is the location for container after build is complete.
	Dest string
	// Format is the format of built container, e.g. SIF, sandbox.
	Format string
	// NoCleanUp allows a user to prevent a bundle from being cleaned
	// up after a failed build, useful for debugging.
	NoCleanUp bool
	// Opts for bundles.
	Opts types.Options
}

// NewBuild creates a new Build struct from a spec (URI, definition file, etc...).
func NewBuild(spec string, conf Config) (*Build, error) {
	def, err := makeDef(spec)
	if err != nil {
		return nil, fmt.Errorf("unable to parse spec %v: %v", spec, err)
	}

	return newBuild([]types.Definition{def}, conf)
}

// New creates a new build struct form a slice of definitions.
func New(defs []types.Definition, conf Config) (*Build, error) {
	return newBuild(defs, conf)
}

func newBuild(defs []types.Definition, conf Config) (*Build, error) {
	sandboxCopy := false
	oldumask := syscall.Umask(0002)
	defer syscall.Umask(oldumask)

	dest, err := filepath.Abs(conf.Dest)
	if err != nil {
		return nil, fmt.Errorf("failed to determine absolute path for %q: %v", conf.Dest, err)
	}
	conf.Dest = dest

	// always build a sandbox if updating an existing sandbox
	if conf.Opts.Update {
		conf.Format = "sandbox"
	}

	b := &Build{
		Conf: conf,
	}

	// look if there is mount options set which could conflict
	// with the build process like nodev and noexec
	entries, err := proc.GetMountInfoEntry("/proc/self/mountinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve mount information: %v", err)
	}

	lastStageIndex := len(defs) - 1

	// create stages
	for i, d := range defs {
		// verify every definition has a header if there are multiple stages
		if d.Header == nil {
			return nil, fmt.Errorf("multiple stages detected, all must have headers")
		}

		rootfsParent := conf.Opts.TmpDir
		if conf.Format == "sandbox" {
			rootfsParent = filepath.Dir(conf.Dest)
		}
		rootfs := filepath.Join(rootfsParent, "rootfs-"+uuid.NewV1().String())

		var s stage
		var err error
		if conf.Opts.EncryptionKeyInfo != nil {
			s.b, err = types.NewEncryptedBundle(rootfs, conf.Opts.TmpDir, conf.Opts.EncryptionKeyInfo)
		} else {
			s.b, err = types.NewBundle(rootfs, conf.Opts.TmpDir)
		}
		if err != nil {
			return nil, err
		}
		s.name = d.Header["stage"]
		s.b.Recipe = d

		if conf.Format == "sandbox" && lastStageIndex == i {
			// rootfs path changed during bundle creation it means that chown
			// is not possible within the temporary rootfs, we will switch to
			// the old behavior which is to create the temporary rootfs inside
			// $TMPDIR and copy the final root filesystem to the destination
			// provided
			if s.b.RootfsPath != rootfs {
				sandboxCopy = true
				sylog.Warningf("The underlying filesystem on which resides %q won't allow to set ownership, "+
					"as a consequence the sandbox could not preserve image's files/directories ownerships", conf.Dest)
			} else {
				// check if the final sandbox directory doesn't have noexec set
				destEntry, err := proc.FindParentMountEntry(rootfsParent, entries)
				if err != nil {
					return nil, fmt.Errorf("failed to find mount point for %s: %v", rootfsParent, err)
				}
				for _, opt := range destEntry.Options {
					if opt == "noexec" {
						return nil, fmt.Errorf("'noexec' mount option set on %s, sandbox %s won't be usable at this location", destEntry.Point, conf.Dest)
					}
				}
			}
		}
		if lastStageIndex == i {
			// check if TMPDIR mount point have nodev and/or noexec set
			tmpdirEntry, err := proc.FindParentMountEntry(conf.Opts.TmpDir, entries)
			if err != nil {
				return nil, fmt.Errorf("failed to find mount point for %s: %v", conf.Opts.TmpDir, err)
			}
			for _, opt := range tmpdirEntry.Options {
				switch opt {
				case "nodev":
					sylog.Warningf("'nodev' mount option set on %s, it could be a source of failure during build process", tmpdirEntry.Point)
				case "noexec":
					return nil, fmt.Errorf("'noexec' mount option set on %s, temporary root filesystem won't be usable at this location", tmpdirEntry.Point)
				}
			}
		}

		s.b.Opts = conf.Opts
		// dont need to get cp if we're skipping bootstrap
		if !conf.Opts.Update || conf.Opts.Force {
			if c, err := conveyorPacker(d); err == nil {
				s.c = c
			} else {
				return nil, fmt.Errorf("unable to get conveyorpacker: %s", err)
			}
		}

		b.stages = append(b.stages, s)
	}

	// only need an assembler for last stage
	switch conf.Format {
	case "sandbox":
		b.stages[lastStageIndex].a = &assemblers.SandboxAssembler{Copy: sandboxCopy}
	case "sif":
		mksquashfsPath, err := squashfs.GetPath()
		if err != nil {
			return nil, fmt.Errorf("while searching for mksquashfs: %v", err)
		}

		flag, err := ensureGzipComp(b.stages[lastStageIndex].b.TmpDir, mksquashfsPath)
		if err != nil {
			return nil, fmt.Errorf("while ensuring correct compression algorithm: %v", err)
		}
		b.stages[lastStageIndex].a = &assemblers.SIFAssembler{
			GzipFlag:       flag,
			MksquashfsPath: mksquashfsPath,
		}
	default:
		return nil, fmt.Errorf("unrecognized output format %s", conf.Format)
	}

	return b, nil
}

// ensureGzipComp builds dummy squashfs images and checks the type of compression used
// to deduce if we can successfully build with gzip compression. It returns an error
// if we cannot and a boolean to indicate if the `-comp` flag is needed to specify
// gzip compression when the final squashfs is built
func ensureGzipComp(tmpdir, mksquashfsPath string) (bool, error) {
	sylog.Debugf("Ensuring gzip compression for mksquashfs")

	var err error
	s := packer.NewSquashfs()
	s.MksquashfsPath = mksquashfsPath

	srcf, err := ioutil.TempFile(tmpdir, "squashfs-gzip-comp-test-src")
	if err != nil {
		return false, fmt.Errorf("while creating temporary file for squashfs source: %v", err)
	}

	srcf.Write([]byte("Test File Content"))
	srcf.Close()

	f, err := ioutil.TempFile(tmpdir, "squashfs-gzip-comp-test-")
	if err != nil {
		return false, fmt.Errorf("while creating temporary file for squashfs: %v", err)
	}
	f.Close()

	flags := []string{"-noappend"}
	if err := s.Create([]string{srcf.Name()}, f.Name(), flags); err != nil {
		return false, fmt.Errorf("while creating squashfs: %v", err)
	}

	content, err := ioutil.ReadFile(f.Name())
	if err != nil {
		return false, fmt.Errorf("while reading test squashfs: %v", err)
	}

	comp, err := image.GetSquashfsComp(content)
	if err != nil {
		return false, fmt.Errorf("could not verify squashfs compression type: %v", err)
	}

	if comp == "gzip" {
		sylog.Debugf("Gzip compression by default ensured")
		return false, nil
	}

	flags = []string{"-noappend", "-comp", "gzip"}
	if err := s.Create([]string{srcf.Name()}, f.Name(), flags); err != nil {
		return false, fmt.Errorf("could not build squashfs with required gzip compression")
	}

	content, err = ioutil.ReadFile(f.Name())
	if err != nil {
		return false, fmt.Errorf("while reading test squashfs: %v", err)
	}

	comp, err = image.GetSquashfsComp(content)
	if err != nil {
		return false, fmt.Errorf("could not verify squashfs compression type: %v", err)
	}

	if comp == "gzip" {
		sylog.Debugf("Gzip compression with -comp flag ensured")
		return true, nil
	}

	return false, fmt.Errorf("could not build squashfs with required gzip compression")
}

// cleanUp removes remnants of build from file system unless NoCleanUp is specified.
func (b Build) cleanUp() {
	if b.Conf.NoCleanUp {
		var bundlePaths []string
		for _, s := range b.stages {
			bundlePaths = append(bundlePaths, s.b.RootfsPath, s.b.TmpDir)
		}
		sylog.Infof("Build performed with no clean up option, build bundle(s) located at: %v", bundlePaths)
		return
	}

	for _, s := range b.stages {
		sylog.Debugf("Cleaning up %q and %q", s.b.RootfsPath, s.b.TmpDir)
		err := s.b.Remove()
		if err != nil {
			sylog.Errorf("Could not remove bundle: %v", err)
		}
	}
}

// Full runs a standard build from start to finish.
func (b *Build) Full(ctx context.Context) error {
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

	oldumask := syscall.Umask(0002)

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

			_, err = p.Pack(ctx)
			if err != nil {
				return err
			}
		} else {
			// regular build or force, start build from scratch
			if b.Conf.Opts.ImgCache == nil {
				return fmt.Errorf("undefined image cache")
			}
			if err := stage.c.Get(ctx, stage.b); err != nil {
				return fmt.Errorf("conveyor failed to get: %v", err)
			}

			_, err := stage.c.Pack(ctx)
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

	syscall.Umask(oldumask)

	sylog.Debugf("Calling assembler")
	if err := b.stages[len(b.stages)-1].Assemble(b.Conf.Dest); err != nil {
		return err
	}

	sylog.Verbosef("Build complete: %s", b.Conf.Dest)
	return nil
}

// engineRequired returns true if build definition is requesting to run scripts or copy files
func engineRequired(def types.Definition) bool {
	return def.BuildData.Post.Script != "" || def.BuildData.Setup.Script != "" || def.BuildData.Test.Script != "" || len(def.BuildData.Files) != 0
}

// runBuildEngine creates an imgbuild engine and creates a container out of our bundle in order to execute %post %setup scripts in the bundle
func runBuildEngine(b *types.Bundle) error {
	if syscall.Getuid() != 0 {
		return fmt.Errorf("attempted to build with scripts as non-root user or without --fakeroot")
	}

	sylog.Debugf("Starting build engine")
	ociConfig := &oci.Config{}

	engineConfig := &imgbuildConfig.EngineConfig{
		Bundle:    *b,
		OciConfig: ociConfig,
	}

	// surface build specific environment variables for scripts
	sRootfs := "SINGULARITY_ROOTFS=" + b.RootfsPath
	sEnvironment := "SINGULARITY_ENVIRONMENT=" + "/.singularity.d/env/91-environment.sh"

	ociConfig.Process = &specs.Process{}
	ociConfig.Process.Env = append(os.Environ(), sRootfs, sEnvironment)

	config := &config.Common{
		EngineName:   imgbuildConfig.Name,
		ContainerID:  "image-build",
		EngineConfig: engineConfig,
	}

	return starter.Run(
		"Singularity image-build",
		config,
		starter.WithStdout(os.Stdout),
		starter.WithStderr(os.Stderr),
	)
}

// makeDef gets a definition object from a spec.
func makeDef(spec string) (types.Definition, error) {
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

	d, err := parser.ParseDefinitionFile(defFile)
	if err != nil {
		return types.Definition{}, fmt.Errorf("while parsing definition: %s: %v", spec, err)
	}

	return d, nil
}

// MakeAllDefs gets a definition object from a spec
func MakeAllDefs(spec string) ([]types.Definition, error) {
	if ok, err := uri.IsValid(spec); ok && err == nil {
		// URI passed as spec
		d, err := types.NewDefinitionFromURI(spec)
		return []types.Definition{d}, err
	}

	// check if spec is an image/sandbox
	if i, err := image.Init(spec, false); err == nil {
		_ = i.File.Close()
		d, err := types.NewDefinitionFromURI("localimage://" + spec)
		return []types.Definition{d}, err
	}

	// default to reading file as definition
	defFile, err := os.Open(spec)
	if err != nil {
		return nil, fmt.Errorf("unable to open file %s: %v", spec, err)
	}
	defer defFile.Close()

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
