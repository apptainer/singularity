// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package mount

import (
	"syscall"
	"testing"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

func TestImage(t *testing.T) {
	points := &Points{}

	if err := points.AddImage("", "/fake", "ext3", 0, 0); err == nil {
		t.Errorf("should failed with empty source")
	}
	if err := points.AddImage("/fake", "", "ext3", 0, 0); err == nil {
		t.Errorf("should failed with empty destination")
	}

	if err := points.AddImage("fake", "/", "ext3", 0, 0); err == nil {
		t.Errorf("should failed as source is not an absolute path")
	}
	if err := points.AddImage("/", "fake", "ext3", 0, 0); err == nil {
		t.Errorf("should failed as destination is not an absolute path")
	}

	if err := points.AddImage("", "/", "ext3", 0, 0); err == nil {
		t.Errorf("should failed with empty source")
	}
	if err := points.AddImage("/fake", "/", "xfs", 0, 0); err == nil {
		t.Errorf("should failed with bad filesystem type")
	}
	if err := points.AddImage("/fake", "/", "ext3", syscall.MS_BIND, 0); err == nil {
		t.Errorf("should failed with bad bind flag")
	}
	if err := points.AddImage("/fake", "/", "ext3", syscall.MS_REMOUNT, 0); err == nil {
		t.Errorf("should failed with bad remount flag")
	}
	if err := points.AddImage("/fake", "/", "ext3", syscall.MS_REC, 0); err == nil {
		t.Errorf("should failed with bad recursive flag")
	}
	if err := points.AddImage("/fake", "/", "ext3", 0, 0); err != nil {
		t.Errorf("should pass with ext3 filesystem")
	}
	if err := points.AddImage("/fake", "/", "squashfs", 0, 0); err != nil {
		t.Errorf("should failed with squashfs filesystem")
	}
	points.RemoveAll()

	if err := points.AddImage("/fake", "/", "squashfs", syscall.MS_NOSUID, 31); err != nil {
		t.Fatalf("should failed with squashfs filesystem")
	}
	images := points.GetAllImages()
	if len(images) != 1 {
		t.Fatalf("should get only one registered image")
	}
	correctOffset := false
	hasNoSuid := false
	for _, option := range images[0].Options {
		if option == "offset=31" {
			correctOffset = true
		} else if option == "nosuid" {
			hasNoSuid = true
		}
	}
	if !correctOffset {
		t.Errorf("offset option wasn't found or is invalid")
	}
	if !hasNoSuid {
		t.Errorf("nosuid option wasn't applied")
	}
	points.Remove("/")
	if len(points.GetAllImages()) != 0 {
		t.Errorf("failed to remove image from mount point")
	}
}

func TestOverlay(t *testing.T) {
	points := &Points{}

	if err := points.AddOverlay("", 0, "/", "", ""); err == nil {
		t.Errorf("should failed with empty destination")
	}
	if err := points.AddOverlay("/fake", 0, "", "/upper", "/work"); err == nil {
		t.Errorf("should failed with empty lowerdir")
	}
	if err := points.AddOverlay("/fake", 0, "/lower", "/upper", ""); err == nil {
		t.Errorf("should failed with empty workdir")
	}

	if err := points.AddOverlay("/", 0, "lower", "", ""); err == nil {
		t.Errorf("should failed as lowerdir is not an absolute path")
	}
	if err := points.AddOverlay("/", 0, "/lower", "upper", "/work"); err == nil {
		t.Errorf("should failed as upperdir is not an absolute path")
	}
	if err := points.AddOverlay("/", 0, "/lower", "/upper", "work"); err == nil {
		t.Errorf("should failed as workdir is not an absolute path")
	}

	if err := points.AddOverlay("/fake", syscall.MS_BIND, "/lower", "", ""); err == nil {
		t.Errorf("should failed with bad bind flag")
	}
	if err := points.AddOverlay("/fake", syscall.MS_REMOUNT, "/lower", "", ""); err == nil {
		t.Errorf("should failed with bad remount flag")
	}
	if err := points.AddOverlay("/fake", syscall.MS_REC, "/lower", "", ""); err == nil {
		t.Errorf("should failed with bad recursive flag")
	}
	points.RemoveAll()

	if err := points.AddOverlay("/fake", 0, "/lower", "", ""); err != nil {
		t.Errorf("%s", err)
	}
	points.RemoveAll()

	if err := points.AddOverlay("/fake", 0, "/lower", "/upper", "/work"); err != nil {
		t.Errorf("%s", err)
	}
	points.RemoveAll()

	if err := points.AddOverlay("/mnt", syscall.MS_NOSUID, "/lower", "/upper", "/work"); err != nil {
		t.Fatalf("%s", err)
	}

	overlay := points.Get("/mnt")
	if len(overlay) != 1 {
		t.Fatalf("one filesystem mount points should be returned")
	}
	hasNoSuid := false
	for _, option := range overlay[0].Options {
		if option == "nosuid" {
			hasNoSuid = true
		}
	}
	if !hasNoSuid {
		t.Errorf("option nosuid not applied for /mnt")
	}
}

func TestFS(t *testing.T) {
	points := &Points{}

	if err := points.AddFS("", "tmpfs", 0, ""); err == nil {
		t.Errorf("should failed with empty destination")
	}
	if err := points.AddFS("fake", "tmpfs", 0, ""); err == nil {
		t.Errorf("should failed as destination is not an absolute path")
	}

	if err := points.AddFS("fake", "tmpfs", syscall.MS_BIND, ""); err == nil {
		t.Errorf("should failed with bad bind flag")
	}
	if err := points.AddFS("fake", "tmpfs", syscall.MS_REMOUNT, ""); err == nil {
		t.Errorf("should failed with bad remount flag")
	}
	if err := points.AddFS("fake", "tmpfs", syscall.MS_REC, ""); err == nil {
		t.Errorf("should failed with bad recursive flag")
	}

	points.RemoveAll()

	if err := points.AddFS("/fields/of", "cows", 0, ""); err == nil {
		t.Errorf("should failed as filesystem is not authorized")
	}

	fs := points.GetAllFS()
	if len(fs) != 0 {
		t.Errorf("no filesystem mount points should be returned")
	}
	points.RemoveAll()

	if err := points.AddFS("/mnt", "tmpfs", syscall.MS_NOSUID, ""); err != nil {
		t.Fatalf("%s", err)
	}

	fs = points.Get("/mnt")
	if len(fs) != 1 {
		t.Fatalf("one filesystem mount points should be returned")
	}
	hasNoSuid := false
	for _, option := range fs[0].Options {
		if option == "nosuid" {
			hasNoSuid = true
		}
	}
	if !hasNoSuid {
		t.Errorf("option nosuid not applied for /mnt")
	}
}

func TestBind(t *testing.T) {
	points := &Points{}

	if err := points.AddBind("/", "", 0); err == nil {
		t.Errorf("should failed with empty destination")
	}

	if err := points.AddBind("fake", "/", 0); err == nil {
		t.Errorf("should failed as source is not an absolute path")
	}
	if err := points.AddBind("/", "fake", 0); err == nil {
		t.Errorf("should failed as destination is not an absolute path")
	}
	points.RemoveAll()

	if err := points.AddBind("/", "/mnt", syscall.MS_BIND); err != nil {
		t.Fatalf("%s", err)
	}
	bind := points.Get("/mnt")
	if len(bind) != 1 {
		t.Fatalf("more than one mount point for /mnt has been returned")
	}
	hasBind := false
	for _, option := range bind[0].Options {
		if option == "bind" {
			hasBind = true
		}
	}
	if !hasBind {
		t.Errorf("option bind not applied for /mnt")
	}
	points.RemoveAll()

	if err := points.AddBind("/", "/mnt", syscall.MS_BIND|syscall.MS_REC); err != nil {
		t.Fatalf("%s", err)
	}
	bind = points.Get("/mnt")
	if len(bind) != 1 {
		t.Fatalf("more than one mount point for /mnt has been returned")
	}
	hasBind = false
	for _, option := range bind[0].Options {
		if option == "rbind" {
			hasBind = true
		}
	}
	if !hasBind {
		t.Errorf("option rbind not applied for /mnt")
	}
}

func TestRemount(t *testing.T) {
	points := &Points{}

	if err := points.AddRemount("", 0); err == nil {
		t.Errorf("should failed with empty destination")
	}
	if err := points.AddRemount("fake", 0); err == nil {
		t.Errorf("should failed as destination is not an absolute path")
	}
	points.RemoveAll()
}

func TestImport(t *testing.T) {
	points := &Points{}

	validImport := []specs.Mount{
		{
			Source:      "/",
			Destination: "/mnt",
			Type:        "",
			Options:     []string{"rbind"},
		},
		{
			Source:      "",
			Destination: "/mnt",
			Type:        "",
			Options:     []string{"rbind", "nosuid", "remount"},
		},
		{
			Source:      "",
			Destination: "/tmp",
			Type:        "tmpfs",
			Options:     []string{"nosuid", "nodev", "mode=1777"},
		},
		{
			Source:      "",
			Destination: "/opt",
			Type:        "overlay",
			Options:     []string{"nosuid", "nodev", "lowerdir=/", "upperdir=/upper", "workdir=/work"},
		},
		{
			Source:      "/image.simg",
			Destination: "/tmp/image",
			Type:        "squashfs",
			Options:     []string{"nosuid", "nodev", "offset=31"},
		},
	}
	if err := points.Import(validImport); err != nil {
		t.Fatalf("%s", err)
	}
	all := points.GetAll()
	if len(all) != len(validImport) {
		t.Errorf("returned a wrong number of mount points %d instead of %d", len(all), len(validImport))
	}
	image := points.GetAllImages()
	if len(image) != 1 {
		t.Errorf("wrong number of image mount point found")
	}
	overlay := points.GetAllOverlays()
	if len(overlay) != 1 {
		t.Errorf("wrong number of overlay mount point found")
	}
	bind := points.GetAllBinds()
	if len(bind) != 2 {
		t.Errorf("wrong number of bind mount point found")
	}
	fs := points.GetAllFS()
	if len(fs) != 1 {
		t.Errorf("wrong number of filesystem mount point found")
	}
	points.Remove("/mnt")
	all = points.GetAll()
	if len(all) != 3 {
		t.Errorf("returned a wrong number of mount points %d instead of %d", len(all), 3)
	}
	points.Remove("/tmp")
	all = points.GetAll()
	if len(all) != 2 {
		t.Errorf("returned a wrong number of mount points %d instead of %d", len(all), 2)
	}
	points.Remove("/opt")
	all = points.GetAll()
	if len(all) != 1 {
		t.Errorf("returned a wrong number of mount points %d instead of %d", len(all), 1)
	}
	points.Remove("/tmp/image")
	all = points.GetAll()
	if len(all) != 0 {
		t.Errorf("returned a wrong number of mount points %d instead of %d", len(all), 0)
	}
	points.RemoveAll()

	invalidImport := []specs.Mount{
		{
			Source:      "/",
			Destination: "/mnt",
			Type:        "",
			Options:     []string{"rbind"},
		},
		{
			Source:      "",
			Destination: "/mnt",
			Type:        "",
			Options:     []string{"rbind", "nosuid"},
		},
	}
	if err := points.Import(invalidImport); err == nil {
		t.Errorf("import should failed")
	}
}
