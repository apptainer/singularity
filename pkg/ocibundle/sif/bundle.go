// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sifbundle

import (
	"fmt"
	"path/filepath"
	"runtime"
	"syscall"

	specs "github.com/opencontainers/runtime-spec/specs-go"

	"github.com/opencontainers/runtime-tools/generate"
	"github.com/sylabs/singularity/pkg/image"
	"github.com/sylabs/singularity/pkg/ocibundle"
	"github.com/sylabs/singularity/pkg/ocibundle/tools"
)

type sifBundle struct {
	image      string
	bundlePath string
	writable   bool
	ocibundle.Bundle
}

// Create creates an OCI bundle from a SIF image
func (s *sifBundle) Create(ociConfig *specs.Spec) error {
	if s.image == "" {
		return fmt.Errorf("image wasn't set, need one to create bundle")
	}

	img, err := image.Init(s.image, s.writable)
	if err != nil {
		return fmt.Errorf("failed to load SIF image %s: %s", s.image, err)
	}
	defer img.File.Close()

	if img.Type != image.SIF {
		return fmt.Errorf("%s is not a SIF image", s.image)
	}
	if img.Partitions[0].Type != image.SQUASHFS {
		return fmt.Errorf("unsupported image fs type: %v", img.Partitions[0].Type)
	}
	offset := img.Partitions[0].Offset
	size := img.Partitions[0].Size

	// TODO: embed OCI image config in SIF directly and generate
	// runtime configuration from this one for environment variables
	// and command/entrypoint
	config := ociConfig
	if config == nil {
		g, err := generate.New(runtime.GOOS)
		if err != nil {
			return fmt.Errorf("failed to generate OCI config: %s", err)
		}
		g.SetProcessArgs([]string{"/.singularity.d/actions/run"})
		config = g.Config
	}

	// create OCI bundle
	if err := tools.CreateBundle(s.bundlePath, config); err != nil {
		return fmt.Errorf("failed to create OCI bundle: %s", err)
	}

	// associate SIF image with a block
	loop, err := tools.CreateLoop(img.File, offset, size)
	if err != nil {
		tools.DeleteBundle(s.bundlePath)
		return fmt.Errorf("failed to find loop device: %s", err)
	}

	rootFs := tools.RootFs(s.bundlePath).Path()
	if err := syscall.Mount(loop, rootFs, "squashfs", syscall.MS_RDONLY, "errors=remount-ro"); err != nil {
		tools.DeleteBundle(s.bundlePath)
		return fmt.Errorf("failed to mount SIF partition: %s", err)
	}

	if s.writable {
		if err := tools.CreateOverlay(s.bundlePath); err != nil {
			// best effort to release loop device
			syscall.Unmount(rootFs, syscall.MNT_DETACH)
			tools.DeleteBundle(s.bundlePath)
			return fmt.Errorf("failed to create overlay: %s", err)
		}
	}
	return nil
}

// Delete erases OCI bundle create from SIF image
func (s *sifBundle) Delete() error {
	if s.writable {
		if err := tools.DeleteOverlay(s.bundlePath); err != nil {
			return fmt.Errorf("delete error: %s", err)
		}
	}
	// Umount rootfs
	rootFsDir := tools.RootFs(s.bundlePath).Path()
	if err := syscall.Unmount(rootFsDir, syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("failed to unmount %s: %s", rootFsDir, err)
	}
	// delete bundle directory
	return tools.DeleteBundle(s.bundlePath)
}

// FromSif returns a bundle interface to create/delete OCI bundle from SIF image
func FromSif(image, bundle string, writable bool) (ocibundle.Bundle, error) {
	var err error

	s := &sifBundle{
		writable: writable,
	}
	s.bundlePath, err = filepath.Abs(bundle)
	if err != nil {
		return nil, fmt.Errorf("failed to determine bundle path: %s", err)
	}
	if image != "" {
		s.image, err = filepath.Abs(image)
		if err != nil {
			return nil, fmt.Errorf("failed to determine image path: %s", err)
		}
	}
	return s, nil
}
