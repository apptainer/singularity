// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sifbundle

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	imageSpecs "github.com/opencontainers/image-spec/specs-go/v1"
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

func (s *sifBundle) writeConfig(img *image.Image, g *generate.Generator) error {
	// check if SIF file contain an OCI image configuration
	reader, err := image.NewSectionReader(img, "oci-config.json", -1)
	if err != nil && err != image.ErrNoSection {
		return fmt.Errorf("failed to read oci-config.json section: %s", err)
	} else if err == image.ErrNoSection {
		return tools.SaveBundleConfig(s.bundlePath, g)
	}

	var imgConfig imageSpecs.ImageConfig

	if err := json.NewDecoder(reader).Decode(&imgConfig); err != nil {
		return fmt.Errorf("failed to decode oci-config.json: %s", err)
	}

	if len(g.Config.Process.Args) == 1 && g.Config.Process.Args[0] == tools.RunScript {
		args := imgConfig.Entrypoint
		args = append(args, imgConfig.Cmd...)
		if len(args) > 0 {
			g.SetProcessArgs(args)
		}
	}

	if g.Config.Process.Cwd == "" && imgConfig.WorkingDir != "" {
		g.SetProcessCwd(imgConfig.WorkingDir)
	}
	for _, e := range imgConfig.Env {
		found := false
		k := strings.SplitN(e, "=", 2)
		for _, pe := range g.Config.Process.Env {
			if strings.HasPrefix(pe, k[0]+"=") {
				found = true
				break
			}
		}
		if !found {
			g.AddProcessEnv(k[0], k[1])
		}
	}

	volumes := tools.Volumes(s.bundlePath).Path()
	for dst := range imgConfig.Volumes {
		replacer := strings.NewReplacer(string(os.PathSeparator), "_")
		src := filepath.Join(volumes, replacer.Replace(dst))
		if err := os.MkdirAll(src, 0755); err != nil {
			return fmt.Errorf("failed to create volume directory %s: %s", src, err)
		}
		g.AddMount(specs.Mount{
			Source:      src,
			Destination: dst,
			Type:        "none",
			Options:     []string{"bind", "rw"},
		})
	}

	return tools.SaveBundleConfig(s.bundlePath, g)
}

// Create creates an OCI bundle from a SIF image
func (s *sifBundle) Create(ociConfig *specs.Spec) error {
	if s.image == "" {
		return fmt.Errorf("image wasn't set, need one to create bundle")
	}

	flag := os.O_RDONLY
	if s.writable {
		flag = os.O_RDWR
	}
	file, err := os.OpenFile(s.image, flag, 0)
	if err != nil {
		return fmt.Errorf("can't open image %s: %s", s.image, err)
	}
	defer file.Close()

	fimg, err := sif.LoadContainerFp(file, !s.writable)
	if err != nil {
		return fmt.Errorf("could not load image fp: %v", err)
	}
	part, _, err := fimg.GetPartPrimSys()
	if err != nil {
		return fmt.Errorf("could not get primaty partitions: %v", err)
	}
	fstype, err := part.GetFsType()
	if err != nil {
		return fmt.Errorf("could not get fs type: %v", err)
	}
	if fstype != sif.FsSquash {
		return fmt.Errorf("unsuported image fs type: %v", fstype)
	}
	offset := uint64(part.Fileoff)
	size := uint64(part.Filelen)

	// generate OCI bundle directory and config
	g, err := tools.GenerateBundleConfig(s.bundlePath, ociConfig)
	if err != nil {
		return fmt.Errorf("failed to generate OCI bundle/config: %s", err)
	}

	// associate SIF image with a block
	loop, err := tools.CreateLoop(file, offset, size)
	if err != nil {
		tools.DeleteBundle(s.bundlePath)
		return fmt.Errorf("failed to find loop device: %s", err)
	}

	rootFs := tools.RootFs(s.bundlePath).Path()
	if err := syscall.Mount(loop, rootFs, "squashfs", syscall.MS_RDONLY, "errors=remount-ro"); err != nil {
		tools.DeleteBundle(s.bundlePath)
		return fmt.Errorf("failed to mount SIF partition: %s", err)
	}

	if err := s.writeConfig(img, g); err != nil {
		// best effort to release loop device
		syscall.Unmount(rootFs, syscall.MNT_DETACH)
		tools.DeleteBundle(s.bundlePath)
		return fmt.Errorf("failed to write OCI configuration: %s", err)
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
