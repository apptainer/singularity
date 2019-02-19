// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	uuid "github.com/satori/go.uuid"
	"github.com/sylabs/sif/pkg/sif"
)

const (
	containerURI       = "library://bauer/default/plugin_compile:3.2.0-alpha"
	containedSourceDir = "/go/src/github.com/sylabs/singularity/plugins/"
)

// pluginObjPath returns the path of the .so file which is built when
// running `go build -buildmode=plugin [...]`.
func pluginObjPath(sourceDir string) string {
	b := filepath.Base(sourceDir)
	return filepath.Join(sourceDir, b+".so")
}

// pluginManifestPath returns the path of the .manifest file created
// in the container after the plugin object is built
func pluginManifestPath(sourceDir string) string {
	b := filepath.Base(sourceDir)
	return filepath.Join(sourceDir, b+".manifest")
}

// CompilePlugin compiles a plugin. It takes as input: sourceDir, the path to the
// plugin's source code directory; and destSif, the path to the intended final
// location of the plugin SIF file.
func CompilePlugin(sourceDir, destSif string) error {
	// generate plugin object via container
	if err := buildPlugin(sourceDir); err != nil {
		return fmt.Errorf("while building plugin .so: %s", err)
	}

	// convert the built plugin object into a sif
	if err := makeSIF(sourceDir, destSif); err != nil {
		return fmt.Errorf("while making sif file: %s", err)
	}

	return nil
}

// buildPlugin takes sourceDir which is the string path the host which contains the source code of
// the plugin. The output path is where the plugin .so file should
// end up.
//
// This function essentially runs the `go build -buildmode=plugin [...]` command
func buildPlugin(sourceDir string) error {
	baseDir := filepath.Base(sourceDir)
	scmd := exec.Command("singularity", "run", "--cleanenv", "-B",
		sourceDir+":"+filepath.Join(containedSourceDir, baseDir),
		containerURI, baseDir)

	scmd.Stderr = os.Stderr
	scmd.Stdout = os.Stdout
	scmd.Stdin = os.Stdin
	return scmd.Run()
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
	defer plObjInput.Fp.Close()

	// add plugin object file descriptor to sif
	plCreateInfo.InputDescr = append(plCreateInfo.InputDescr, plObjInput)

	// create plugin manifest descriptor
	plManifestInput, err := getPluginManifestDescr(pluginManifestPath(sourceDir))
	if err != nil {
		return err
	}
	defer plManifestInput.Fp.Close()

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
	objInput.Fp, err = os.Open(objInput.Fname)
	if err != nil {
		return sif.DescriptorInput{}, fmt.Errorf("while opening plugin object file %s: %s", objInput.Fname, err)
	}

	// stat file to obtain size
	fstat, err := objInput.Fp.Stat()
	if err != nil {
		return sif.DescriptorInput{}, fmt.Errorf("while calling stat on plugin object file %s: %s", objInput.Fname, err)
	}
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
	var err error

	manifestInput := sif.DescriptorInput{
		Datatype: sif.DataGenericJSON,
		Groupid:  sif.DescrDefaultGroup,
		Link:     sif.DescrUnusedLink,
		Fname:    manifestPath,
	}

	// open plugin object file
	manifestInput.Fp, err = os.Open(manifestInput.Fname)
	if err != nil {
		return sif.DescriptorInput{}, fmt.Errorf("while opening plugin object file %s: %s", manifestInput.Fname, err)
	}

	// stat file to obtain size
	fstat, err := manifestInput.Fp.Stat()
	if err != nil {
		return sif.DescriptorInput{}, fmt.Errorf("while calling stat on plugin object file %s: %s", manifestInput.Fname, err)
	}
	manifestInput.Size = fstat.Size()

	return manifestInput, nil
}
