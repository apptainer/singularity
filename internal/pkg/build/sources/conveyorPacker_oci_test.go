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

	"github.com/sylabs/singularity/internal/pkg/build/sources"
	"github.com/sylabs/singularity/internal/pkg/test"
	"github.com/sylabs/singularity/pkg/build/types"
	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
)

const (
	dockerURI         = "docker://alpine"
	dockerArchiveURI  = "https://s3.amazonaws.com/singularity-ci-public/alpine-docker-save.tar"
	ociArchiveURI     = "https://s3.amazonaws.com/singularity-ci-public/alpine-oci-archive.tar"
	dockerDaemonImage = "alpine:latest"
)

func TestMain(m *testing.M) {
	useragent.InitValue("singularity", "3.0.0-alpha.1-303-gaed8d30-dirty")

	os.Exit(m.Run())
}

// TestOCIConveyorDocker tests if we can pull an alpine image from dockerhub
func TestOCIConveyorDocker(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	test.SetCacheDir(t)
	defer test.CleanCacheDir(t)

	b, err := types.NewBundle("", "sbuild-oci")
	if err != nil {
		return
	}

	b.Recipe, err = types.NewDefinitionFromURI(dockerURI)
	if err != nil {
		t.Fatalf("unable to parse URI %s: %v\n", dockerURI, err)
	}

	cp := &sources.OCIConveyorPacker{}

	err = cp.Get(b)
	// clean up tmpfs since assembler isnt called
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

	// Since we manipulate archives/images, we make sure to have a clean image cache
	test.SetCacheDir(t)
	defer test.CleanCacheDir(t)

	archive, err := getTestTar(dockerArchiveURI)
	if err != nil {
		t.Fatalf("Could not download docker archive test file: %v", err)
	}
	defer os.Remove(archive)

	b, err := types.NewBundle("", "sbuild-oci")
	if err != nil {
		return
	}

	archiveURI := "docker-archive:" + archive
	b.Recipe, err = types.NewDefinitionFromURI(archiveURI)
	if err != nil {
		t.Fatalf("unable to parse URI %s: %v\n", archiveURI, err)
	}

	cp := &sources.OCIConveyorPacker{}

	err = cp.Get(b)
	// clean up tmpfs since assembler isnt called
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

	b, err := types.NewBundle("", "sbuild-oci")
	if err != nil {
		return
	}

	daemonURI := "docker-daemon:" + dockerDaemonImage
	b.Recipe, err = types.NewDefinitionFromURI(daemonURI)
	if err != nil {
		t.Fatalf("unable to parse URI %s: %v\n", daemonURI, err)
	}

	cp := &sources.OCIConveyorPacker{}

	err = cp.Get(b)
	// clean up tmpfs since assembler isnt called
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

	test.SetCacheDir(t)
	defer test.CleanCacheDir(t)

	archive, err := getTestTar(ociArchiveURI)
	if err != nil {
		t.Fatalf("Could not download oci archive test file: %v", err)
	}
	defer os.Remove(archive)

	b, err := types.NewBundle("", "sbuild-oci")
	if err != nil {
		return
	}

	archiveURI := "oci-archive:" + archive
	b.Recipe, err = types.NewDefinitionFromURI(archiveURI)
	if err != nil {
		t.Fatalf("unable to parse URI %s: %v\n", archiveURI, err)
	}

	cp := &sources.OCIConveyorPacker{}

	err = cp.Get(b)
	// clean up tmpfs since assembler isnt called
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

	test.SetCacheDir(t)
	defer test.CleanCacheDir(t)

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

	b, err := types.NewBundle("", "sbuild-oci")
	if err != nil {
		return
	}

	layoutURI := "oci:" + dir
	b.Recipe, err = types.NewDefinitionFromURI(layoutURI)
	if err != nil {
		t.Fatalf("unable to parse URI %s: %v\n", layoutURI, err)
	}

	cp := &sources.OCIConveyorPacker{}

	err = cp.Get(b)
	// clean up tmpfs since assembler isnt called
	defer cp.CleanUp()
	if err != nil {
		t.Fatalf("failed to Get from %s: %v\n", layoutURI, err)
	}
}

// TestOCIPacker checks if we can create a Kitchen
func TestOCIPacker(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	test.SetCacheDir(t)
	defer test.CleanCacheDir(t)

	b, err := types.NewBundle("", "sbuild-oci")
	if err != nil {
		return
	}

	b.Recipe, err = types.NewDefinitionFromURI(dockerURI)
	if err != nil {
		t.Fatalf("unable to parse URI %s: %v\n", dockerURI, err)
	}

	ocp := &sources.OCIConveyorPacker{}

	err = ocp.Get(b)
	// clean up tmpfs since assembler isnt called
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
