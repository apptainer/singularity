// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity/internal/pkg/build/apps"
	"github.com/sylabs/singularity/internal/pkg/build/assemblers"
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
	// dest is the location for container after build is complete
	dest string
	// format is the format of built container, e.g., SIF, sandbox
	format string
	// c Gets and Packs data needed to build a container into a Bundle from various sources
	c ConveyorPacker
	// a Assembles a container from the information stored in a Bundle into various formats
	a Assembler
	// b is an intermediate structure that encapsulates all information for the container, e.g., metadata, filesystems
	b *types.Bundle
}

// NewBuild creates a new Build struct from a spec (URI, definition file, etc...)
func NewBuild(spec, dest, format string, libraryURL, authToken string, opts types.Options) (*Build, error) {
	def, err := makeDef(spec, false)
	if err != nil {
		return nil, fmt.Errorf("unable to parse spec %v: %v", spec, err)
	}

	return newBuild(def, dest, format, libraryURL, authToken, opts)
}

// NewBuildJSON creates a new build struct from a JSON byte slice
func NewBuildJSON(r io.Reader, dest, format string, libraryURL, authToken string, opts types.Options) (*Build, error) {
	def, err := types.NewDefinitionFromJSON(r)
	if err != nil {
		return nil, fmt.Errorf("unable to parse JSON: %v", err)
	}

	return newBuild(def, dest, format, libraryURL, authToken, opts)
}

func newBuild(d types.Definition, dest, format string, libraryURL, authToken string, opts types.Options) (*Build, error) {
	var err error

	syscall.Umask(0002)

	// always build a sandbox if updating an existing sandbox
	if opts.Update {
		format = "sandbox"
	}

	b := &Build{
		format: format,
		dest:   dest,
	}

	b.b, err = types.NewBundle(opts.TmpDir, "sbuild")
	if err != nil {
		return nil, err
	}

	b.b.Recipe = d
	b.b.Opts = opts

	// dont need to get cp if we're skipping bootstrap
	if !opts.Update || opts.Force {
		if c, err := getcp(b.b.Recipe, libraryURL, authToken); err == nil {
			b.c = c
		} else {
			return nil, fmt.Errorf("unable to get conveyorpacker: %s", err)
		}
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

// cleanUp removes remnants of build from file system unless NoCleanUp is specified
func (b Build) cleanUp() {
	if b.b.Opts.NoCleanUp {
		sylog.Infof("Build performed with no clean up option, build bundle located at: %v", b.b.Path)
		return
	}
	sylog.Debugf("Build bundle cleanup: %v", b.b.Path)
	os.RemoveAll(b.b.Path)
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

	if err := b.runPreScript(); err != nil {
		return err
	}

	if b.b.Opts.Update && !b.b.Opts.Force {
		//if updating, extract dest container to bundle
		sylog.Infof("Building into existing container: %s", b.dest)
		p, err := sources.GetLocalPacker(b.dest, b.b)
		if err != nil {
			return err
		}

		_, err = p.Pack()
		if err != nil {
			return err
		}
	} else {
		//if force, start build from scratch
		if err := b.c.Get(b.b); err != nil {
			return fmt.Errorf("conveyor failed to get: %v", err)
		}

		_, err := b.c.Pack()
		if err != nil {
			return fmt.Errorf("packer failed to pack: %v", err)
		}
	}

	// create apps in bundle
	a := apps.New()
	for k, v := range b.b.Recipe.CustomData {
		a.HandleSection(k, v)
	}

	a.HandleBundle(b.b)
	b.b.Recipe.BuildData.Post += a.HandlePost()

	if engineRequired(b.b.Recipe) {
		if err := b.runBuildEngine(); err != nil {
			return fmt.Errorf("while running engine: %v", err)
		}
	}

	sylog.Debugf("Inserting Metadata")
	if err := b.insertMetadata(); err != nil {
		return fmt.Errorf("While inserting metadata to bundle: %v", err)
	}

	sylog.Debugf("Calling assembler")
	if err := b.Assemble(b.dest); err != nil {
		return err
	}

	sylog.Infof("Build complete: %s", b.dest)
	return nil
}

// engineRequired returns true if build definition is requesting to run scripts or copy files
func engineRequired(def types.Definition) bool {
	return def.BuildData.Post != "" || def.BuildData.Setup != "" || def.BuildData.Test != "" || len(def.BuildData.Files) != 0
}

func (b *Build) copyFiles() error {

	// iterate through files transfers
	for _, transfer := range b.b.Recipe.BuildData.Files {
		// sanity
		if transfer.Src == "" {
			sylog.Warningf("Attempt to copy file with no name...")
			continue
		}
		// dest = source if not specified
		if transfer.Dst == "" {
			transfer.Dst = transfer.Src
		}
		sylog.Infof("Copying %v to %v", transfer.Src, transfer.Dst)
		// copy each file into bundle rootfs
		transfer.Dst = filepath.Join(b.b.Rootfs(), transfer.Dst)
		copy := exec.Command("/bin/cp", "-fLr", transfer.Src, transfer.Dst)
		if err := copy.Run(); err != nil {
			return fmt.Errorf("While copying %v to %v: %v", transfer.Src, transfer.Dst, err)
		}
	}

	return nil
}

func (b *Build) insertMetadata() (err error) {
	// insert help
	err = insertHelpScript(b.b)
	if err != nil {
		return fmt.Errorf("While inserting help script: %v", err)
	}

	// insert labels
	err = insertLabelsJSON(b.b)
	if err != nil {
		return fmt.Errorf("While inserting labels JSON: %v", err)
	}

	// insert definition
	err = insertDefinition(b.b)
	if err != nil {
		return fmt.Errorf("While inserting definition: %v", err)
	}

	// insert environment
	err = insertEnvScript(b.b)
	if err != nil {
		return fmt.Errorf("While inserting environment script: %v", err)
	}

	// insert startscript
	err = insertStartScript(b.b)
	if err != nil {
		return fmt.Errorf("While inserting startscript: %v", err)
	}

	// insert runscript
	err = insertRunScript(b.b)
	if err != nil {
		return fmt.Errorf("While inserting runscript: %v", err)
	}

	// insert test script
	err = insertTestScript(b.b)
	if err != nil {
		return fmt.Errorf("While inserting test script: %v", err)
	}

	return
}

func (b *Build) runPreScript() error {
	if b.runPre() && b.b.Recipe.BuildData.Pre != "" {
		if syscall.Getuid() != 0 {
			return fmt.Errorf("Attempted to build with scripts as non-root user")
		}

		// Run %pre script here
		pre := exec.Command("/bin/sh", "-cex", b.b.Recipe.BuildData.Pre)
		pre.Stdout = os.Stdout
		pre.Stderr = os.Stderr

		sylog.Infof("Running pre scriptlet\n")
		if err := pre.Start(); err != nil {
			return fmt.Errorf("failed to start %%pre proc: %v", err)
		}
		if err := pre.Wait(); err != nil {
			return fmt.Errorf("pre proc: %v", err)
		}
	}
	return nil
}

// runBuildEngine creates an imgbuild engine and creates a container out of our bundle in order to execute %post %setup scripts in the bundle
func (b *Build) runBuildEngine() error {
	if syscall.Getuid() != 0 {
		return fmt.Errorf("Attempted to build with scripts as non-root user")
	}

	sylog.Debugf("Starting build engine")
	env := []string{sylog.GetEnvVar()}
	starter := filepath.Join(buildcfg.LIBEXECDIR, "/singularity/bin/starter")
	progname := []string{"singularity image-build"}
	ociConfig := &oci.Config{}

	engineConfig := &imgbuildConfig.EngineConfig{
		Bundle:    *b.b,
		OciConfig: ociConfig,
	}

	// surface build specific environment variables for scripts
	sRootfs := "SINGULARITY_ROOTFS=" + b.b.Rootfs()
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

func getcp(def types.Definition, libraryURL, authToken string) (ConveyorPacker, error) {
	switch def.Header["bootstrap"] {
	case "library":
		return &sources.LibraryConveyorPacker{
			LibraryURL: libraryURL,
			AuthToken:  authToken,
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
		sylog.Fatalf("You must be the root user to build from a Singularity recipe file")
	}

	d, err := parser.ParseDefinitionFile(defFile)
	if err != nil {
		return types.Definition{}, fmt.Errorf("While parsing definition: %s: %v", spec, err)
	}

	return d, nil
}

// runPre determines if %pre section was specified to be run from the CLI
func (b Build) runPre() bool {
	for _, section := range b.b.Opts.Sections {
		if section == "none" {
			return false
		}
		if section == "all" || section == "pre" {
			return true
		}
	}
	return false
}

// MakeDef gets a definition object from a spec
func MakeDef(spec string, remote bool) (types.Definition, error) {
	return makeDef(spec, remote)
}

// Assemble assembles the bundle to the specified path
func (b *Build) Assemble(path string) error {
	return b.a.Assemble(b.b, path)
}

func insertEnvScript(b *types.Bundle) error {
	if b.RunSection("environment") && b.Recipe.ImageData.Environment != "" {
		sylog.Infof("Adding environment to container")
		envScriptPath := filepath.Join(b.Rootfs(), "/.singularity.d/env/90-environment.sh")
		_, err := os.Stat(envScriptPath)
		if os.IsNotExist(err) {
			err := ioutil.WriteFile(envScriptPath, []byte("#!/bin/sh\n\n"+b.Recipe.ImageData.Environment+"\n"), 0755)
			if err != nil {
				return err
			}
		} else {
			// append to script if it already exists
			f, err := os.OpenFile(envScriptPath, os.O_APPEND|os.O_WRONLY, 0755)
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = f.WriteString("\n" + b.Recipe.ImageData.Environment + "\n")
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func insertRunScript(b *types.Bundle) error {
	if b.RunSection("runscript") && b.Recipe.ImageData.Runscript != "" {
		sylog.Infof("Adding runscript")
		err := ioutil.WriteFile(filepath.Join(b.Rootfs(), "/.singularity.d/runscript"), []byte("#!/bin/sh\n\n"+b.Recipe.ImageData.Runscript+"\n"), 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

func insertStartScript(b *types.Bundle) error {
	if b.RunSection("startscript") && b.Recipe.ImageData.Startscript != "" {
		sylog.Infof("Adding startscript")
		err := ioutil.WriteFile(filepath.Join(b.Rootfs(), "/.singularity.d/startscript"), []byte("#!/bin/sh\n\n"+b.Recipe.ImageData.Startscript+"\n"), 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

func insertTestScript(b *types.Bundle) error {
	if b.RunSection("test") && b.Recipe.ImageData.Test != "" {
		sylog.Infof("Adding testscript")
		err := ioutil.WriteFile(filepath.Join(b.Rootfs(), "/.singularity.d/test"), []byte("#!/bin/sh\n\n"+b.Recipe.ImageData.Test+"\n"), 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

func insertHelpScript(b *types.Bundle) error {
	if b.RunSection("help") && b.Recipe.ImageData.Help != "" {
		_, err := os.Stat(filepath.Join(b.Rootfs(), "/.singularity.d/runscript.help"))
		if err != nil || b.Opts.Force {
			sylog.Infof("Adding help info")
			err := ioutil.WriteFile(filepath.Join(b.Rootfs(), "/.singularity.d/runscript.help"), []byte(b.Recipe.ImageData.Help+"\n"), 0644)
			if err != nil {
				return err
			}
		} else {
			sylog.Warningf("Help message already exists and force option is false, not overwriting")
		}
	}
	return nil
}

func insertDefinition(b *types.Bundle) error {

	// if update, check for existing definition and move it to bootstrap history
	if b.Opts.Update {
		if _, err := os.Stat(filepath.Join(b.Rootfs(), "/.singularity.d/Singularity")); err == nil {
			// make bootstrap_history directory if it doesnt exist
			if _, err := os.Stat(filepath.Join(b.Rootfs(), "/.singularity.d/bootstrap_history")); err != nil {
				err = os.Mkdir(filepath.Join(b.Rootfs(), "/.singularity.d/bootstrap_history"), 0755)
				if err != nil {
					return err
				}
			}

			// look at number of files in bootstrap_history to give correct file name
			files, err := ioutil.ReadDir(filepath.Join(b.Rootfs(), "/.singularity.d/bootstrap_history"))

			// name is "Singularity" concatenated with an index based on number of other files in bootstrap_history
			len := strconv.Itoa(len(files))

			histName := "Singularity" + len

			// move old definition into bootstrap_history
			err = os.Rename(filepath.Join(b.Rootfs(), "/.singularity.d/Singularity"), filepath.Join(b.Rootfs(), "/.singularity.d/bootstrap_history", histName))
			if err != nil {
				return err
			}
		}

	}

	err := ioutil.WriteFile(filepath.Join(b.Rootfs(), "/.singularity.d/Singularity"), b.Recipe.Raw, 0644)
	if err != nil {
		return err
	}

	return nil
}

func insertLabelsJSON(b *types.Bundle) (err error) {
	var text []byte
	labels := make(map[string]string)

	if err = getExistingLabels(labels, b); err != nil {
		return err
	}

	if err = addBuildLabels(labels, b); err != nil {
		return err
	}

	if b.RunSection("labels") && len(b.Recipe.ImageData.Labels) > 0 {
		sylog.Infof("Adding labels")

		// add new labels to new map and check for collisions
		for key, value := range b.Recipe.ImageData.Labels {
			// check if label already exists
			if _, ok := labels[key]; ok {
				// overwrite collision if it exists and force flag is set
				if b.Opts.Force {
					labels[key] = value
				} else {
					sylog.Warningf("Label: %s already exists and force option is false, not overwriting", key)
				}
			} else {
				// set if it doesnt
				labels[key] = value
			}
		}
	}

	// make new map into json
	text, err = json.MarshalIndent(labels, "", "\t")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(b.Rootfs(), "/.singularity.d/labels.json"), []byte(text), 0644)
	return err
}

func getExistingLabels(labels map[string]string, b *types.Bundle) error {
	// check for existing labels in bundle
	if _, err := os.Stat(filepath.Join(b.Rootfs(), "/.singularity.d/labels.json")); err == nil {

		jsonFile, err := os.Open(filepath.Join(b.Rootfs(), "/.singularity.d/labels.json"))
		if err != nil {
			return err
		}
		defer jsonFile.Close()

		jsonBytes, err := ioutil.ReadAll(jsonFile)
		if err != nil {
			return err
		}

		err = json.Unmarshal(jsonBytes, &labels)
		if err != nil {
			return err
		}
	}
	return nil
}

func addBuildLabels(labels map[string]string, b *types.Bundle) error {
	// schema version
	labels["org.label-schema.schema-version"] = "1.0"

	// build date and time, lots of time formatting
	currentTime := time.Now()
	year, month, day := currentTime.Date()
	date := strconv.Itoa(day) + `_` + month.String() + `_` + strconv.Itoa(year)
	hour, min, sec := currentTime.Clock()
	time := strconv.Itoa(hour) + `:` + strconv.Itoa(min) + `:` + strconv.Itoa(sec)
	zone, _ := currentTime.Zone()
	timeString := currentTime.Weekday().String() + `_` + date + `_` + time + `_` + zone
	labels["org.label-schema.build-date"] = timeString

	// singularity version
	labels["org.label-schema.usage.singularity.version"] = buildcfg.PACKAGE_VERSION

	// help info if help exists in the definition and is run in the build
	if b.RunSection("help") && b.Recipe.ImageData.Help != "" {
		labels["org.label-schema.usage"] = "/.singularity.d/runscript.help"
		labels["org.label-schema.usage.singularity.runscript.help"] = "/.singularity.d/runscript.help"
	}

	// bootstrap header info, only if this build actually bootstrapped
	if !b.Opts.Update || b.Opts.Force {
		for key, value := range b.Recipe.Header {
			labels["org.label-schema.usage.singularity.deffile."+key] = value
		}
	}

	return nil
}
