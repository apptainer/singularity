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
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/util/fs"

	imageSpecs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sylabs/singularity/pkg/image/unpacker"
)

func downloadImage(t *testing.T) string {
	sexec, err := exec.LookPath("singularity")
	if err != nil {
		t.SkipNow()
	}
	f, err := ioutil.TempFile("", "image-")
	if err != nil {
		t.Fatal(err)
	}
	name := f.Name()
	f.Close()

	cmd := exec.Command(sexec, "build", "-F", name, "docker://busybox")
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
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
