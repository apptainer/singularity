// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"errors"
	"fmt"
	"go/build"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	uuid "github.com/satori/go.uuid"
	"github.com/sylabs/sif/pkg/sif"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

var (
	workpath   = filepath.Join(filepath.SplitList(build.Default.GOPATH)[0], repo)
	trimpath   = filepath.Dir(workpath)
	mangenpath = filepath.Join(workpath, "cmd/plugin_manifestgen/")
)

const (
	repo = "src/github.com/sylabs/singularity"
)

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
	// build plugin object using go buiild
	_, err := buildPlugin(sourceDir, buildTags)
	if err != nil {
		return fmt.Errorf("while building plugin .so: %s", err)
	}

	// generate plugin manifest from .so
	_, err = generateManifest(sourceDir, buildTags)
	if err != nil {
		return fmt.Errorf("while generating plugin manifest: %s", err)
	}

	// convert the built plugin object into a sif
	if err := makeSIF(sourceDir, destSif); err != nil {
		return fmt.Errorf("while making sif file: %s", err)
	}

	return nil
}

// buildPlugin takes sourceDir which is the string path the host which contains the source code of
// the plugin. buildPlugin returns the path to the built file, along with an error
//
// This function essentially runs the `go build -buildmode=plugin [...]` command
func buildPlugin(sourceDir, buildTags string) (string, error) {
	// assuming that sourceDir is within trimpath for now
	out := pluginObjPath(sourceDir)

	goTool, err := exec.LookPath("go")
	if err != nil {
		return "", errors.New("go compiler not found")
	}

	args := []string{
		"build",
		"-o", out,
		"-buildmode=plugin",
		"-tags", buildTags,
		fmt.Sprintf("-gcflags=all=-trimpath=%s", trimpath),
		fmt.Sprintf("-asmflags=all=-trimpath=%s", trimpath),
		sourceDir,
	}

	sylog.Debugf("Runnig: %s %s", goTool, strings.Join(args, " "))

	buildcmd := exec.Command(goTool, args...)

	buildcmd.Dir = workpath
	buildcmd.Stderr = os.Stderr
	buildcmd.Stdout = os.Stdout
	buildcmd.Stdin = os.Stdin

	return out, buildcmd.Run()
}

// generateManifest takes the path to the plugin source, sourceDir, and generates
// its corresponding manifest file.
func generateManifest(sourceDir, buildTags string) (string, error) {
	in := pluginObjPath(sourceDir)
	out := pluginManifestPath(sourceDir)

	goTool, err := exec.LookPath("go")
	if err != nil {
		return "", errors.New("go compiler not found")
	}

	args := []string{
		"run",
		"-tags", buildTags,
		fmt.Sprintf("-gcflags=all=-trimpath=%s", trimpath),
		fmt.Sprintf("-asmflags=all=-trimpath=%s", trimpath),
		mangenpath,
		in,
		out,
	}

	gencmd := exec.Command(goTool, args...)

	gencmd.Dir = workpath
	gencmd.Stderr = os.Stderr
	gencmd.Stdout = os.Stdout
	gencmd.Stdin = os.Stdin

	return out, gencmd.Run()
}

// makeSIF takes in two arguments: sourceDir, the path to the plugin source directory;
// and sifPath, the path to the final .sif file which is ready to be used
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
