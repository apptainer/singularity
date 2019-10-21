// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"runtime"
	"strings"

	uuid "github.com/satori/go.uuid"
	"github.com/sylabs/sif/pkg/sif"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	pluginapi "github.com/sylabs/singularity/pkg/plugin"
)

// getSingularitySrcDir returns the source directory for singularity.
func getSingularitySrcDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	canary := filepath.Join(dir, "cmd", "singularity", "cli.go")

	switch _, err = os.Stat(canary); {
	case os.IsNotExist(err):
		return "", fmt.Errorf("cannot find %q", canary)

	case err != nil:
		return "", fmt.Errorf("unexpected error while looking for %q: %s", canary, err)

	default:
		return dir, nil
	}
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
func CompilePlugin(sourceDir, destSif, buildTags string) error {
	// build plugin object using go build
	soPath, err := buildPlugin(sourceDir, buildTags)
	if err != nil {
		return fmt.Errorf("while building plugin .so: %v", err)
	}
	defer os.Remove(soPath)

	// generate plugin manifest from .so
	mPath, err := generateManifest(sourceDir)
	if err != nil {
		return fmt.Errorf("while generating plugin manifest: %s", err)
	}
	defer os.Remove(mPath)

	// convert the built plugin object into a sif
	if err := makeSIF(sourceDir, destSif); err != nil {
		return fmt.Errorf("while making sif file: %s", err)
	}

	return nil
}

// buildPlugin takes sourceDir which is the string path the host which contains the source code of
// the plugin. buildPlugin returns the path to the built file, along with an error.
//
// This function essentially runs the `go build -buildmode=plugin [...]` command.
func buildPlugin(sourceDir, buildTags string) (string, error) {
	workpath, err := getSingularitySrcDir()
	if err != nil {
		return "", errors.New("singularity source directory not found")
	}

	// assuming that sourceDir is within trimpath for now
	out := pluginObjPath(sourceDir)

	goTool, err := exec.LookPath("go")
	if err != nil {
		return "", errors.New("go compiler not found")
	}

	args := []string{
		"build",
		"-o", out,
		"-trimpath",
		"-buildmode=plugin",
		"-tags", buildTags,
		sourceDir,
	}

	sylog.Debugf("Running: %s %s", goTool, strings.Join(args, " "))

	buildcmd := exec.Command(goTool, args...)

	buildcmd.Dir = workpath
	buildcmd.Stderr = os.Stderr
	buildcmd.Stdout = os.Stdout
	buildcmd.Stdin = os.Stdin
	buildcmd.Env = append(os.Environ(), "GO111MODULE=on")

	return out, buildcmd.Run()
}

// generateManifest takes the path to the plugin source, sourceDir, and generates
// its corresponding manifest file.
func generateManifest(sourceDir string) (string, error) {
	in := pluginObjPath(sourceDir)
	out := pluginManifestPath(sourceDir)

	pluginPointer, err := plugin.Open(in)
	if err != nil {
		return "", err
	}

	sym, err := pluginPointer.Lookup(pluginapi.PluginSymbol)
	if err != nil {
		return "", err
	}

	p, ok := sym.(*pluginapi.Plugin)
	if !ok {
		return "", fmt.Errorf(`symbol "Plugin" not of type Plugin`)
	}

	manifest, err := json.Marshal(p.Manifest)
	if err != nil {
		return "", fmt.Errorf("could not marshal manifest to json: %v", err)
	}

	if err := ioutil.WriteFile(out, manifest, 0644); err != nil {
		return "", fmt.Errorf("could not write manifest: %v", err)
	}
	return out, nil
}

// makeSIF takes in two arguments: sourceDir, the path to the plugin source directory;
// and sifPath, the path to the final .sif file which is ready to be used.
func makeSIF(sourceDir, sifPath string) error {
	plCreateInfo := sif.CreateInfo{
		Pathname:   sifPath,
		Launchstr:  sif.HdrLaunch,
		Sifversion: sif.HdrVersion,
		ID:         uuid.NewV4(),
	}

	// create plugin object file descriptor
	plObjInput, err := getPluginObjDescr(pluginObjPath(sourceDir))
	if err != nil {
		return err
	}

	if fp, ok := plObjInput.Fp.(io.Closer); ok {
		defer fp.Close()
	}

	// add plugin object file descriptor to sif
	plCreateInfo.InputDescr = append(plCreateInfo.InputDescr, plObjInput)

	// create plugin manifest descriptor
	plManifestInput, err := getPluginManifestDescr(pluginManifestPath(sourceDir))
	if err != nil {
		return err
	}
	if fp, ok := plManifestInput.Fp.(io.Closer); ok {
		defer fp.Close()
	}

	// add plugin manifest descriptor to sif
	plCreateInfo.InputDescr = append(plCreateInfo.InputDescr, plManifestInput)

	os.RemoveAll(sifPath)

	// create sif file
	if _, err := sif.CreateContainer(plCreateInfo); err != nil {
		return fmt.Errorf("while creating sif file: %s", err)
	}

	return nil
}

// getPluginObjDescr returns a sif.DescriptorInput which contains the raw
// data of the .so file.
//
// Datatype: sif.DataPartition
// Fstype:   sif.FsRaw
// Parttype: sif.PartData
func getPluginObjDescr(objPath string) (sif.DescriptorInput, error) {
	var err error

	objInput := sif.DescriptorInput{
		Datatype: sif.DataPartition,
		Groupid:  sif.DescrDefaultGroup,
		Link:     sif.DescrUnusedLink,
		Fname:    objPath,
	}

	// open plugin object file
	fp, err := os.Open(objInput.Fname)
	if err != nil {
		return sif.DescriptorInput{}, fmt.Errorf("while opening plugin object file %s: %s", objInput.Fname, err)
	}

	// stat file to obtain size
	fstat, err := fp.Stat()
	if err != nil {
		return sif.DescriptorInput{}, fmt.Errorf("while calling stat on plugin object file %s: %s", objInput.Fname, err)
	}

	objInput.Fp = fp
	objInput.Size = fstat.Size()

	// populate objInput.Extra with appropriate Fstype & Parttype
	err = objInput.SetPartExtra(sif.FsRaw, sif.PartData, sif.GetSIFArch(runtime.GOARCH))
	if err != nil {
		return sif.DescriptorInput{}, err
	}

	return objInput, nil
}

// getPluginManifestDescr returns a sif.DescriptorInput which contains the manifest
// in JSON form. Grabbing the Manifest is done by loading the .so using the plugin
// package, which is performed inside the container during buildPlugin() function
//
// Datatype: sif.DataGenericJSON
func getPluginManifestDescr(manifestPath string) (sif.DescriptorInput, error) {
	manifestInput := sif.DescriptorInput{
		Datatype: sif.DataGenericJSON,
		Groupid:  sif.DescrDefaultGroup,
		Link:     sif.DescrUnusedLink,
		Fname:    manifestPath,
	}

	// open plugin object file
	fp, err := os.Open(manifestInput.Fname)
	if err != nil {
		return sif.DescriptorInput{}, fmt.Errorf("while opening plugin object file %s: %s", manifestInput.Fname, err)
	}

	// stat file to obtain size
	fstat, err := fp.Stat()
	if err != nil {
		return sif.DescriptorInput{}, fmt.Errorf("while calling stat on plugin object file %s: %s", manifestInput.Fname, err)
	}

	manifestInput.Fp = fp
	manifestInput.Size = fstat.Size()

	return manifestInput, nil
}
