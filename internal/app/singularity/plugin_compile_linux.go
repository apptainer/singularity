// Copyright (c) 2018-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/hpcng/sif/v2/pkg/sif"
	"github.com/hpcng/singularity/internal/pkg/buildcfg"
	"github.com/hpcng/singularity/internal/pkg/plugin"
	"github.com/hpcng/singularity/internal/pkg/util/bin"
	pluginapi "github.com/hpcng/singularity/pkg/plugin"
	"github.com/hpcng/singularity/pkg/sylog"
	"github.com/hpcng/singularity/pkg/util/archive"
)

const version = "v0.0.0"

const goVersionFile = `package main
import "fmt"
import "runtime"
func main() { fmt.Printf(runtime.Version()) }`

type buildToolchain struct {
	goPath            string
	singularitySource string
	pluginDir         string
	buildTags         string
	envs              []string
}

func getPackageName() string {
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		return buildInfo.Main.Path
	}
	return "github.com/hpcng/singularity"
}

// getSingularitySrcDir returns the source directory for singularity.
func getSingularitySrcDir() (string, error) {
	dir := buildcfg.SOURCEDIR
	pkgName := getPackageName()

	// get current file path
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("could not determine source directory")
	}

	// replace github.com/hpcng/singularity@v0.0.0
	pattern := fmt.Sprintf("%s@%s", pkgName, version)
	filename = strings.Replace(filename, pattern, "", 1)

	// look if source directory is present
	canary := filepath.Join(dir, filename)
	sylog.Debugf("Searching source file %s", canary)

	switch _, err := os.Stat(canary); {
	case os.IsNotExist(err):
		return "", fmt.Errorf("cannot find %q", canary)

	case err != nil:
		return "", fmt.Errorf("unexpected error while looking for %q: %s", canary, err)

	default:
		return dir, nil
	}
}

// checkGoVersion returns an error if the currently Go toolchain is
// different from the one used to compile singularity. Singularity
// and plugin must be compiled with the same toolchain.
func checkGoVersion(tmpDir, goPath string) error {
	var out bytes.Buffer

	path := filepath.Join(tmpDir, "rt_version.go")
	if err := ioutil.WriteFile(path, []byte(goVersionFile), 0o600); err != nil {
		return fmt.Errorf("while writing go file %s: %s", path, err)
	}
	defer os.Remove(path)

	cmd := exec.Command(goPath, "run", path)
	cmd.Dir = tmpDir
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("while executing go version: %s", err)
	}

	output := out.String()

	runtimeVersion := runtime.Version()
	if output != runtimeVersion {
		return fmt.Errorf("plugin compilation requires Go runtime %q, current is %q", runtimeVersion, output)
	}

	return nil
}

// pluginObjPath returns the path of the .so file which is built when
// running `go build -buildmode=plugin [...]`.
func pluginObjPath(sourceDir string) string {
	return filepath.Join(sourceDir, "plugin.so")
}

// pluginManifestPath returns the path of the .manifest file created
// in the container after the plugin object is built
func pluginManifestPath(sourceDir string) string {
	return filepath.Join(sourceDir, "plugin.manifest")
}

// CompilePlugin compiles a plugin. It takes as input: sourceDir, the path to the
// plugin's source code directory; and destSif, the path to the intended final
// location of the plugin SIF file.
func CompilePlugin(sourceDir, destSif, buildTags string, disableMinorCheck bool) error {
	singularitySrcDir, err := getSingularitySrcDir()
	if err != nil {
		return errors.New("singularity source directory not found")
	}
	goPath, err := bin.FindBin("go")
	if err != nil {
		return errors.New("go compiler not found")
	}

	// copy plugin directory to apply modification on-the-fly
	d, err := ioutil.TempDir("", "plugin-")
	if err != nil {
		return errors.New("temporary directory creation failed")
	}
	defer os.RemoveAll(d)

	// we need to use the exact same go runtime version used
	// to compile Singularity
	if err := checkGoVersion(d, goPath); err != nil {
		return fmt.Errorf("while checking go version: %s", err)
	}

	pluginDir := filepath.Join(d, "src")

	err = archive.CopyWithTar(sourceDir, pluginDir)
	if err != nil {
		return err
	}

	sourceLink := filepath.Join(pluginDir, plugin.SingularitySource)
	// delete it first if already present
	os.Remove(sourceLink)

	if err := os.Symlink(singularitySrcDir, sourceLink); err != nil {
		return fmt.Errorf("while creating %s symlink: %s", sourceLink, err)
	}

	bTool := buildToolchain{
		buildTags:         buildTags,
		singularitySource: singularitySrcDir,
		pluginDir:         pluginDir,
		goPath:            goPath,
		envs:              append(os.Environ(), "GO111MODULE=on"),
	}

	// generating final go.mod file
	modData, err := plugin.PrepareGoModules(sourceDir, disableMinorCheck)
	if err != nil {
		return err
	}

	goMod := filepath.Join(pluginDir, "go.mod")
	if err := ioutil.WriteFile(goMod, modData, 0o600); err != nil {
		return fmt.Errorf("while generating %s: %s", goMod, err)
	}

	// running go mod tidy for plugin go.sum and cleanup
	var e bytes.Buffer
	cmd := exec.Command(goPath, "mod", "tidy")
	cmd.Stderr = &e
	cmd.Dir = pluginDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("while verifying module: %s\nCommand error:\n%s", err, e.String())
	}

	// build plugin object using go build
	if _, err := buildPlugin(pluginDir, bTool); err != nil {
		return fmt.Errorf("while building plugin .so: %v", err)
	}

	// generate plugin manifest from .so
	if err := generateManifest(pluginDir, bTool); err != nil {
		return fmt.Errorf("while generating plugin manifest: %s", err)
	}

	// convert the built plugin object into a sif
	if err := makeSIF(pluginDir, destSif); err != nil {
		return fmt.Errorf("while making sif file: %s", err)
	}

	return nil
}

// buildPlugin takes sourceDir which is the string path the host which
// contains the source code of the plugin. buildPlugin returns the path
// to the built file, along with an error.
//
// This function essentially runs the `go build -buildmode=plugin [...]`
// command.
func buildPlugin(sourceDir string, bTool buildToolchain) (string, error) {
	// assuming that sourceDir is within trimpath for now
	out := pluginObjPath(sourceDir)
	// set pluginRootDirVar variable if required by the plugin
	pluginRootDirVar := fmt.Sprintf("-X main.%s=%s", pluginapi.PluginRootDirSymbol, buildcfg.PLUGIN_ROOTDIR)

	args := []string{
		"build",
		"-a",
		"-o", out,
		"-mod=readonly",
		"-ldflags", pluginRootDirVar,
		"-trimpath",
		"-buildmode=plugin",
		"-tags", bTool.buildTags,
		".",
	}

	sylog.Debugf("Running: %s %s", bTool.goPath, strings.Join(args, " "))

	buildcmd := exec.Command(bTool.goPath, args...)

	buildcmd.Dir = bTool.pluginDir
	buildcmd.Stderr = os.Stderr
	buildcmd.Stdout = os.Stdout
	buildcmd.Stdin = os.Stdin
	buildcmd.Env = bTool.envs

	return out, buildcmd.Run()
}

// generateManifest takes the path to the plugin source, extracts
// plugin's manifest by loading it into memory and stores it's json
// representation in a separate file.
func generateManifest(sourceDir string, bTool buildToolchain) error {
	in := pluginObjPath(sourceDir)
	out := pluginManifestPath(sourceDir)

	p, err := plugin.LoadObject(in)
	if err != nil {
		return fmt.Errorf("while loading plugin %s: %s", in, err)
	}

	f, err := os.OpenFile(out, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("while creating manifest %s: %s", out, err)
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(p.Manifest); err != nil {
		return fmt.Errorf("while writing manifest %s: %s", out, err)
	}

	return nil
}

// makeSIF takes in two arguments: sourceDir, the path to the plugin source directory;
// and sifPath, the path to the final .sif file which is ready to be used.
func makeSIF(sourceDir, sifPath string) error {
	objPath := pluginObjPath(sourceDir)

	fp, err := os.Open(objPath)
	if err != nil {
		return fmt.Errorf("while opening plugin object file %v: %w", objPath, err)
	}
	defer fp.Close()

	plObjInput, err := sif.NewDescriptorInput(sif.DataPartition, fp,
		sif.OptObjectName("plugin.so"),
		sif.OptPartitionMetadata(sif.FsRaw, sif.PartData, runtime.GOARCH),
	)
	if err != nil {
		return err
	}

	// create plugin manifest descriptor
	manifestPath := pluginManifestPath(sourceDir)

	fp, err = os.Open(manifestPath)
	if err != nil {
		return fmt.Errorf("while opening plugin manifest file %v: %w", manifestPath, err)
	}
	defer fp.Close()

	plManifestInput, err := sif.NewDescriptorInput(sif.DataGenericJSON, fp,
		sif.OptObjectName("plugin.manifest"),
	)
	if err != nil {
		return err
	}

	os.RemoveAll(sifPath)

	f, err := sif.CreateContainerAtPath(sifPath,
		sif.OptCreateWithDescriptors(plObjInput, plManifestInput),
	)
	if err != nil {
		return fmt.Errorf("while creating sif file: %w", err)
	}

	if err := f.UnloadContainer(); err != nil {
		return fmt.Errorf("while unloading sif file: %w", err)
	}

	return nil
}
