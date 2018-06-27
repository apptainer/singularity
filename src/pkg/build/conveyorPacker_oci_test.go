// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"testing"
)

const (
	dockerURI         = "docker://alpine"
	dockerArchiveURI  = "https://s3.amazonaws.com/singularity-ci-public/alpine-docker-save.tar"
	ociArchiveURI     = "https://s3.amazonaws.com/singularity-ci-public/alpine-oci-archive.tar"
	dockerDaemonImage = "alpine:latest"
)

// TestOCIConveyorDocker tests if we can pull an alpine image from dockerhub
func TestOCIConveyorDocker(t *testing.T) {
	def, err := NewDefinitionFromURI(dockerURI)
	if err != nil {
		t.Fatalf("unable to parse URI %s: %v\n", dockerURI, err)
	}

	oc := &OCIConveyor{}

	if err := oc.Get(def); err != nil {
		t.Fatalf("failed to Get from %s: %v\n", dockerURI, err)
	}
}

// TestOCIConveyorDockerArchive tests if we can use a docker save archive
// as a source
func TestOCIConveyorDockerArchive(t *testing.T) {
	archive, err := getTestTar(dockerArchiveURI)
	if err != nil {
		t.Fatalf("Could not download docker archive test file: %v", err)
	}
	defer os.Remove(archive)

	archiveURI := "docker-archive:" + archive
	def, err := NewDefinitionFromURI(archiveURI)
	if err != nil {
		t.Fatalf("unable to parse URI %s: %v\n", archiveURI, err)
	}

	oc := &OCIConveyor{}

	if err := oc.Get(def); err != nil {
		t.Fatalf("failed to Get from %s: %v\n", archiveURI, err)
	}
}

// TestOCIConveyerDockerDaemon tests if we can use an oci laytout dir
// as a source
func TestOCIConveyorDockerDaemon(t *testing.T) {
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
	def, err := NewDefinitionFromURI(daemonURI)
	if err != nil {
		t.Fatalf("unable to parse URI %s: %v\n", daemonURI, err)
	}

	oc := &OCIConveyor{}

	if err := oc.Get(def); err != nil {
		t.Fatalf("failed to Get from %s: %v\n", daemonURI, err)
	}
}

// TestOCIConveyorOCIArchive tests if we can use an oci archive
// as a source
func TestOCIConveyorOCIArchive(t *testing.T) {
	archive, err := getTestTar(ociArchiveURI)
	if err != nil {
		t.Fatalf("Could not download oci archive test file: %v", err)
	}
	defer os.Remove(archive)

	archiveURI := "oci-archive:" + archive
	def, err := NewDefinitionFromURI(archiveURI)
	if err != nil {
		t.Fatalf("unable to parse URI %s: %v\n", archiveURI, err)
	}

	oc := &OCIConveyor{}

	if err := oc.Get(def); err != nil {
		t.Fatalf("failed to Get from %s: %v\n", archiveURI, err)
	}
}

// TestOCIConveyerOCILayout tests if we can use an oci layout dir
// as a source
func TestOCIConveyorOCILayout(t *testing.T) {
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
	def, err := NewDefinitionFromURI(layoutURI)
	if err != nil {
		t.Fatalf("unable to parse URI %s: %v\n", layoutURI, err)
	}

	oc := &OCIConveyor{}

	if err := oc.Get(def); err != nil {
		t.Fatalf("failed to Get from %s: %v\n", layoutURI, err)
	}
}

// TestOCIPacker checks if we can create a Kitchen
func TestOCIPacker(t *testing.T) {
	def, err := NewDefinitionFromURI(dockerURI)
	if err != nil {
		t.Fatalf("unable to parse URI %s: %v\n", dockerURI, err)
	}

	ocp := &OCIConveyorPacker{}

	if err := ocp.Get(def); err != nil {
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
