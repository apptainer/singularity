// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package image

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/util/fs"

	imageSpecs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sylabs/singularity/pkg/image/unpacker"
)

func downloadImage(t *testing.T) string {
	sexec, err := exec.LookPath("singularity")
	if err != nil {
		t.Log("cannot find singularity path, skipping test")
		t.SkipNow()
	}
	f, err := ioutil.TempFile("", "image-")
	if err != nil {
		t.Fatalf("cannot create temporary file: %s\n", err)
	}
	name := f.Name()
	f.Close()

	cmd := exec.Command(sexec, "build", "-F", name, "docker://busybox")
	if err := cmd.Run(); err != nil {
		t.Fatalf("cannot create image (cmd: %s build -F %s docker://busybox): %s\n", sexec, name, err)
	}
	return name
}

func checkPartition(reader io.Reader) error {
	extracted := "/bin/busybox"
	dir, err := ioutil.TempDir("", "extract-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	s := unpacker.NewSquashfs()
	if s.HasUnsquashfs() {
		if err := s.ExtractFiles([]string{extracted}, reader, dir); err != nil {
			return fmt.Errorf("extraction failed: %s", err)
		}
		if !fs.IsExec(filepath.Join(dir, extracted)) {
			return fmt.Errorf("%s extraction failed", extracted)
		}
	}
	return nil
}

func checkSection(reader io.Reader) error {
	dec := json.NewDecoder(reader)
	imgSpec := &imageSpecs.ImageConfig{}
	if err := dec.Decode(imgSpec); err != nil {
		return fmt.Errorf("failed to decode oci image config")
	}
	if len(imgSpec.Cmd) == 0 {
		return fmt.Errorf("no command found")
	}
	if imgSpec.Cmd[0] != "sh" {
		return fmt.Errorf("unexpected value: %s instead of sh", imgSpec.Cmd[0])
	}
	return nil
}

func TestReader(t *testing.T) {
	filename := downloadImage(t)
	defer os.Remove(filename)

	for _, e := range []struct {
		fn       func(*Image, string, int) (io.Reader, error)
		fnCheck  func(io.Reader) error
		errCheck error
		name     string
		index    int
	}{
		{
			fn:       NewPartitionReader,
			fnCheck:  checkPartition,
			errCheck: ErrNoPartition,
			name:     RootFs,
			index:    -1,
		},
		{
			fn:       NewPartitionReader,
			fnCheck:  checkPartition,
			errCheck: ErrNoPartition,
			index:    0,
		},
		{
			fn:       NewSectionReader,
			fnCheck:  checkSection,
			errCheck: ErrNoSection,
			name:     "oci-config.json",
			index:    -1,
		},
	} {
		// test with nil image parameter
		if _, err := e.fn(nil, "", -1); err == nil {
			t.Errorf("unexpected success with nil image parameter")
		}
		// test with non opened file
		if _, err := e.fn(&Image{}, "", -1); err == nil {
			t.Errorf("unexpected success with non opened file")
		}

		img, err := Init(filename, false)
		if err != nil {
			t.Fatal(err)
		}

		if img.Type != SIF {
			t.Errorf("unexpected image format: %v", img.Type)
		}
		if !img.HasRootFs() {
			t.Errorf("no root filesystem found")
		}
		// test without match criteria
		if _, err := e.fn(img, "", -1); err == nil {
			t.Errorf("unexpected success without match criteria")
		}
		// test with large index
		if _, err := e.fn(img, "", 999999); err == nil {
			t.Errorf("unexpected success with large index")
		}
		// test with unknown name
		if _, err := e.fn(img, "fakefile.name", -1); err != e.errCheck {
			t.Errorf("unexpected error with unknown name")
		}
		// test with match criteria
		if r, err := e.fn(img, e.name, e.index); err == e.errCheck {
			t.Error(err)
		} else {
			if err := e.fnCheck(r); err != nil {
				t.Error(err)
			}
		}
		img.File.Close()
	}
}

func TestAuthorizedPath(t *testing.T) {
	tests := []struct {
		name       string
		path       []string
		shouldPass bool
	}{
		{"empty path", []string{""}, false},
		{"invalid path", []string{"/a/random/invalid/path"}, false},
		{"valid path", []string{"/"}, true},
	}

	// Create a temporary image
	path := downloadImage(t)
	defer os.Remove(path)

	// Now load the image which will be used next for a bunch of tests
	img, err := Init(path, true)
	if err != nil {
		t.Fatal("impossible to load image for testing")
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			auth, err := img.AuthorizedPath(test.path)
			if test.shouldPass == false && (auth == true && err == nil) {
				t.Fatal("invalid path was reported as authorized")
			}
			if test.shouldPass == true && (auth == false || err != nil) {
				if err != nil {
					t.Fatalf("valid path was reported as not authorized: %s", err)
				} else {
					t.Fatal("valid path was reported as not authorized")
				}
			}
		})
	}
}

func TestAuthorizedOwner(t *testing.T) {
	type ownerGroup struct {
		name       string
		owners     []string
		shouldPass bool
	}

	tests := []ownerGroup{
		{"empty owner list", []string{""}, false},
		{"invalid owner list", []string{"2"}, false},
		{"root", []string{"root"}, false},
	}

	// If the test is not running as root, we test with the current username,
	// i.e., the owner of the image. Note that it is not supposed to work with
	// root.
	me, err := user.Current()
	if err != nil {
		t.Fatalf("cannot get current user name for testing purposes: %s", err)
	}
	if me.Username != "root" {
		localUser := ownerGroup{"valid owner list", []string{me.Username}, true}
		tests = append(tests, localUser)
	}

	// Create a temporary image
	path := downloadImage(t)
	defer os.Remove(path)

	// Now load the image which will be used next for a bunch of tests
	img, err := Init(path, true)
	if err != nil {
		t.Fatal("impossible to load image for testing")
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			auth, err := img.AuthorizedOwner(test.owners)
			if test.shouldPass == false && (auth == true && err == nil) {
				t.Fatal("invalid owner list was reported as authorized")
			}
			if test.shouldPass == true && (auth == false || err != nil) {
				if err != nil {
					t.Fatalf("valid owner list was reported as not authorized: %s", err)
				} else {
					t.Fatal("valid owner list was reported as not authorized")
				}
			}
		})
	}
}

func TestAuthorizedGroup(t *testing.T) {
	type groupTest struct {
		name       string
		groups     []string
		shouldPass bool
	}

	tests := []groupTest{
		{"empty group list", []string{""}, false},
		{"invalid group list", []string{"-"}, false},
		{"root", []string{"root"}, false},
	}

	// If the current group is not root, we test the function with its name,
	// which is a valid test since the owner of the image
	me, err := user.Current()
	if err != nil {
		t.Fatalf("cannot get the current username: %s", err)
	}
	myGroup, gpErr := user.LookupGroupId(me.Gid)
	if gpErr != nil {
		t.Fatalf("cannot lookup the current user's group: %s", err)
	}
	if myGroup.Name != "root" {
		validTest := groupTest{"valid group list", []string{myGroup.Name}, true}
		tests = append(tests, validTest)
	}

	// Create a temporary image
	path := downloadImage(t)
	defer os.Remove(path)

	// Now load the image which will be used next for a bunch of tests
	img, err := Init(path, true)
	if err != nil {
		t.Fatal("impossible to load image for testing")
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			auth, err := img.AuthorizedGroup(test.groups)
			if test.shouldPass == false && (auth == true && err == nil) {
				t.Fatal("invalid group list was reported as authorized")
			}
			if test.shouldPass == true && (auth == false || err != nil) {
				if err != nil {
					t.Fatalf("valid group list was reported as not authorized: %s", err)
				} else {
					t.Fatal("valid group list was reported as not authorized")
				}
			}
		})
	}
}
