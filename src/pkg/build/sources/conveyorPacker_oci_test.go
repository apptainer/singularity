// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources_test

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"testing"

	"github.com/singularityware/singularity/src/pkg/build/sources"
	"github.com/singularityware/singularity/src/pkg/build/types"
	"github.com/singularityware/singularity/src/pkg/test"
)

const (
	dockerURI         = "docker://alpine"
	dockerArchiveURI  = "https://s3.amazonaws.com/singularity-ci-public/alpine-docker-save.tar"
	ociArchiveURI     = "https://s3.amazonaws.com/singularity-ci-public/alpine-oci-archive.tar"
	dockerDaemonImage = "alpine:latest"
)

// TestOCIConveyorDocker tests if we can pull an alpine image from dockerhub
func TestOCIConveyorDocker(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	def, err := types.NewDefinitionFromURI(dockerURI)
	if err != nil {
		t.Fatalf("unable to parse URI %s: %v\n", dockerURI, err)
	}

	cp := &sources.OCIConveyorPacker{}

	err = cp.Get(def)
	//clean up tmpfs since assembler isnt called
	defer cp.CleanUp()
	if err != nil {
		t.Fatalf("failed to Get from %s: %v\n", dockerURI, err)
	}
}

// TestOCIConveyorDockerArchive tests if we can use a docker save archive
// as a source
func TestOCIConveyorDockerArchive(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	archive, err := getTestTar(dockerArchiveURI)
	if err != nil {
		t.Fatalf("Could not download docker archive test file: %v", err)
	}
	defer os.Remove(archive)

	archiveURI := "docker-archive:" + archive
	def, err := types.NewDefinitionFromURI(archiveURI)
	if err != nil {
		t.Fatalf("unable to parse URI %s: %v\n", archiveURI, err)
	}

	cp := &sources.OCIConveyorPacker{}

	err = cp.Get(def)
	//clean up tmpfs since assembler isnt called
	defer cp.CleanUp()
	if err != nil {
		t.Fatalf("failed to Get from %s: %v\n", archiveURI, err)
	}
}

// TestOCIConveyerDockerDaemon tests if we can use an oci laytout dir
// as a source
func TestOCIConveyorDockerDaemon(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	cmd := exec.Command("docker", "ps")
	err := cmd.Run()
	if err != nil {
		t.Logf("docker not available - skipping docker-daemon test")
		return
	}

	cmd = exec.Command("docker", "pull", dockerDaemonImage)
	err = cmd.Run()
	if err != nil {
		t.Fatalf("could not docker pull alpine:latest %v", err)
		return
	}

	daemonURI := "docker-daemon:" + dockerDaemonImage
	def, err := types.NewDefinitionFromURI(daemonURI)
	if err != nil {
		t.Fatalf("unable to parse URI %s: %v\n", daemonURI, err)
	}

	cp := &sources.OCIConveyorPacker{}

	err = cp.Get(def)
	//clean up tmpfs since assembler isnt called
	defer cp.CleanUp()
	if err != nil {
		t.Fatalf("failed to Get from %s: %v\n", daemonURI, err)
	}
}

// TestOCIConveyorOCIArchive tests if we can use an oci archive
// as a source
func TestOCIConveyorOCIArchive(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	archive, err := getTestTar(ociArchiveURI)
	if err != nil {
		t.Fatalf("Could not download oci archive test file: %v", err)
	}
	defer os.Remove(archive)

	archiveURI := "oci-archive:" + archive
	def, err := types.NewDefinitionFromURI(archiveURI)
	if err != nil {
		t.Fatalf("unable to parse URI %s: %v\n", archiveURI, err)
	}

	cp := &sources.OCIConveyorPacker{}

	err = cp.Get(def)
	//clean up tmpfs since assembler isnt called
	defer cp.CleanUp()
	if err != nil {
		t.Fatalf("failed to Get from %s: %v\n", archiveURI, err)
	}
}

// TestOCIConveyerOCILayout tests if we can use an oci layout dir
// as a source
func TestOCIConveyorOCILayout(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	archive, err := getTestTar(ociArchiveURI)
	if err != nil {
		t.Fatalf("Could not download oci archive test file: %v", err)
	}
	defer os.Remove(archive)

	// We need to extract the oci archive to a directory
	// Don't want to implement untar routines here, so use system tar
	dir, err := ioutil.TempDir("", "oci-test")
	if err != nil {
		t.Fatalf("Could not create temporary directory: %v", err)
	}
	defer os.RemoveAll(dir)
	cmd := exec.Command("tar", "-C", dir, "-xf", archive)
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Error extracting oci archive to layout: %v", err)
	}

	layoutURI := "oci:" + dir
	def, err := types.NewDefinitionFromURI(layoutURI)
	if err != nil {
		t.Fatalf("unable to parse URI %s: %v\n", layoutURI, err)
	}

	cp := &sources.OCIConveyorPacker{}

	err = cp.Get(def)
	//clean up tmpfs since assembler isnt called
	defer cp.CleanUp()
	if err != nil {
		t.Fatalf("failed to Get from %s: %v\n", layoutURI, err)
	}
}

// TestOCIPacker checks if we can create a Kitchen
func TestOCIPacker(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	def, err := types.NewDefinitionFromURI(dockerURI)
	if err != nil {
		t.Fatalf("unable to parse URI %s: %v\n", dockerURI, err)
	}

	ocp := &sources.OCIConveyorPacker{}

	err = ocp.Get(def)
	//clean up tmpfs since assembler isnt called
	defer ocp.CleanUp()
	if err != nil {
		t.Fatalf("failed to Get from %s: %v\n", dockerURI, err)
	}

	_, err = ocp.Pack()

	if err != nil {
		t.Fatalf("failed to Pack from %s: %v\n", dockerURI, err)
	}
}

func getTestTar(url string) (path string, err error) {
	dl, err := ioutil.TempFile("", "oci-test")
	if err != nil {
		log.Fatal(err)
	}
	defer dl.Close()

	r, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer r.Body.Close()

	_, err = io.Copy(dl, r.Body)
	if err != nil {
		return "", err
	}

	return dl.Name(), nil
}
