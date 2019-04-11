// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package mount

import (
	"fmt"
	"syscall"
	"testing"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity/internal/pkg/test"
)

func TestImage(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	points := &Points{}

	if err := points.AddImage(RootfsTag, "", "/fake", "ext3", 0, 0, 10); err == nil {
		t.Errorf("should have failed with empty source")
	}
	if err := points.AddImage(RootfsTag, "/fake", "", "ext3", 0, 0, 10); err == nil {
		t.Errorf("should have failed with empty destination")
	}

	if err := points.AddImage(RootfsTag, "fake", "/", "ext3", 0, 0, 10); err == nil {
		t.Errorf("should have failed as source is not an absolute path")
	}
	if err := points.AddImage(RootfsTag, "/", "fake", "ext3", 0, 0, 10); err == nil {
		t.Errorf("should have failed as destination is not an absolute path")
	}

	if err := points.AddImage(RootfsTag, "", "/", "ext3", 0, 0, 10); err == nil {
		t.Errorf("should have failed with empty source")
	}
	if err := points.AddImage(RootfsTag, "/fake", "/", "xfs", 0, 0, 10); err == nil {
		t.Errorf("should have failed with bad filesystem type")
	}
	if err := points.AddImage(RootfsTag, "/fake", "/", "ext3", syscall.MS_BIND, 0, 10); err == nil {
		t.Errorf("should have failed with bad bind flag")
	}
	if err := points.AddImage(RootfsTag, "/fake", "/", "ext3", syscall.MS_REMOUNT, 0, 10); err == nil {
		t.Errorf("should have failed with bad remount flag")
	}
	if err := points.AddImage(RootfsTag, "/fake", "/", "ext3", syscall.MS_REC, 0, 10); err == nil {
		t.Errorf("should have failed with bad recursive flag")
	}
	if err := points.AddImage(RootfsTag, "/fake", "/ext3", "ext3", 0, 0, 10); err != nil {
		t.Errorf("should have passed with ext3 filesystem")
	}
	points.RemoveAll()
	if err := points.AddImage(RootfsTag, "/fake", "/squash", "squashfs", 0, 0, 10); err != nil {
		t.Errorf("should have passed with squashfs filesystem")
	}
	if err := points.AddImage(RootfsTag, "/fake", "/", "squashfs", 0, 0, 0); err == nil {
		t.Errorf("should have failed with 0 size limit")
	}
	if err := points.AddImage(RootfsTag, "/fake", "/squash", "squashfs", 0, 0, 10); err == nil {
		t.Errorf("nil error returned, should have returned non-nil mount.ErrMountExists")
	} else if err != ErrMountExists {
		t.Errorf("non-nil error should have been mount.ErrMountExists")
	}
	points.RemoveAll()

	if err := points.AddImage(RootfsTag, "/fake", "/", "squashfs", syscall.MS_NOSUID, 31, 10); err != nil {
		t.Fatalf("should have passed with squashfs filesystem")
	}
	images := points.GetAllImages()
	if len(images) != 1 {
		t.Fatalf("should get only one registered image")
	}
	hasNoSuid := false
	for _, option := range images[0].Options {
		if option == "nosuid" {
			hasNoSuid = true
		}
	}
	if offset, err := GetOffset(images[0].InternalOptions); err != nil || offset != 31 {
		t.Errorf("offset option wasn't found or is invalid")
	}
	if size, err := GetSizeLimit(images[0].InternalOptions); err != nil || size != 10 {
		t.Errorf("sizelimit option wasn't found or is invalid")
	}
	if _, err := GetOffset([]string{}); err == nil {
		t.Errorf("should have failed, offset not provided")
	}
	if _, err := GetSizeLimit([]string{}); err == nil {
		t.Errorf("should have failed, sizelimit not provided")
	}
	if !hasNoSuid {
		t.Errorf("nosuid option wasn't applied")
	}
	points.RemoveByDest("/")
	if len(points.GetAllImages()) != 0 {
		t.Errorf("failed to remove image from mount point")
	}
}

func TestOverlay(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	points := &Points{}

	if err := points.AddOverlay(LayerTag, "", 0, "/", "", ""); err == nil {
		t.Errorf("should have failed with empty destination")
	}
	if err := points.AddOverlay(LayerTag, "/fake", 0, "", "/upper", "/work"); err == nil {
		t.Errorf("should have failed with empty lowerdir")
	}
	if err := points.AddOverlay(LayerTag, "/fake", 0, "/lower", "/upper", ""); err == nil {
		t.Errorf("should have failed with empty workdir")
	}

	if err := points.AddOverlay(LayerTag, "/", 0, "lower", "", ""); err == nil {
		t.Errorf("should have failed as lowerdir is not an absolute path")
	}
	if err := points.AddOverlay(LayerTag, "/", 0, "/lower", "upper", "/work"); err == nil {
		t.Errorf("should have failed as upperdir is not an absolute path")
	}
	if err := points.AddOverlay(LayerTag, "/", 0, "/lower", "/upper", "work"); err == nil {
		t.Errorf("should have failed as workdir is not an absolute path")
	}

	if err := points.AddOverlay(LayerTag, "/fake", syscall.MS_BIND, "/lower", "", ""); err == nil {
		t.Errorf("should have failed with bad bind flag")
	}
	if err := points.AddOverlay(LayerTag, "/fake", syscall.MS_REMOUNT, "/lower", "", ""); err == nil {
		t.Errorf("should have failed with bad remount flag")
	}
	if err := points.AddOverlay(LayerTag, "/fake", syscall.MS_REC, "/lower", "", ""); err == nil {
		t.Errorf("should have failed with bad recursive flag")
	}
	points.RemoveAll()

	if err := points.AddOverlay(LayerTag, "/fake", 0, "/lower", "", ""); err != nil {
		t.Errorf("%s", err)
	}
	points.RemoveAll()

	if err := points.AddOverlay(LayerTag, "/fake", 0, "/lower", "/upper", "/work"); err != nil {
		t.Errorf("%s", err)
	}
	points.RemoveAll()

	if err := points.AddOverlay(LayerTag, "/mnt", syscall.MS_NOSUID, "/lower", "/upper", "/work"); err != nil {
		t.Fatalf("%s", err)
	}

	overlay := points.GetByDest("/mnt")
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
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	points := &Points{}

	if err := points.AddFS(SessionTag, "", "tmpfs", 0, ""); err == nil {
		t.Errorf("should have failed with empty destination")
	}
	if err := points.AddFS(SessionTag, "fake", "tmpfs", 0, ""); err == nil {
		t.Errorf("should have failed as destination is not an absolute path")
	}

	if err := points.AddFS(SessionTag, "fake", "tmpfs", syscall.MS_BIND, ""); err == nil {
		t.Errorf("should have failed with bad bind flag")
	}
	if err := points.AddFS(SessionTag, "fake", "tmpfs", syscall.MS_REMOUNT, ""); err == nil {
		t.Errorf("should have failed with bad remount flag")
	}
	if err := points.AddFS(SessionTag, "fake", "tmpfs", syscall.MS_REC, ""); err == nil {
		t.Errorf("should have failed with bad recursive flag")
	}

	points.RemoveAll()

	if err := points.AddFS(SessionTag, "/fields/of", "cows", 0, ""); err == nil {
		t.Errorf("should have failed as filesystem is not authorized")
	}

	fs := points.GetAllFS()
	if len(fs) != 0 {
		t.Errorf("no filesystem mount points should be returned")
	}
	points.RemoveAll()

	if err := points.AddFS(SessionTag, "/mnt", "tmpfs", syscall.MS_NOSUID, ""); err != nil {
		t.Fatalf("%s", err)
	}

	fs = points.GetByDest("/mnt")
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
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	points := &Points{}

	if err := points.AddBind(UserbindsTag, "/", "", 0); err == nil {
		t.Errorf("should have failed with empty destination")
	}

	if err := points.AddBind(UserbindsTag, "fake", "/", 0); err == nil {
		t.Errorf("should have failed as source is not an absolute path")
	}
	if err := points.AddBind(UserbindsTag, "/", "fake", 0); err == nil {
		t.Errorf("should have failed as destination is not an absolute path")
	}
	points.RemoveAll()

	if err := points.AddBind(UserbindsTag, "/", "/mnt", syscall.MS_BIND); err != nil {
		t.Fatalf("%s", err)
	}
	bind := points.GetByDest("/mnt")
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

	if err := points.AddBind(UserbindsTag, "/", "/mnt", syscall.MS_BIND|syscall.MS_REC); err != nil {
		t.Fatalf("%s", err)
	}
	bind = points.GetByDest("/mnt")
	if len(bind) != 1 {
		t.Fatalf("more than one mount point for /mnt has been returned")
	}
	bind = points.GetBySource("/")
	if len(bind) != 1 {
		t.Fatalf("more than one mount point for / has been returned")
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
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	points := &Points{}

	if err := points.AddRemount(UserbindsTag, "", 0); err == nil {
		t.Errorf("should have failed with empty destination")
	}
	if err := points.AddRemount(UserbindsTag, "fake", 0); err == nil {
		t.Errorf("should have failed as destination is not an absolute path")
	}
	points.RemoveAll()
}

func TestAddPropagation(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	points := &Points{}

	if err := points.AddPropagation(UserbindsTag, "", 0); err == nil {
		t.Errorf("should have failed with empty destination")
	}
	if err := points.AddPropagation(UserbindsTag, "/mnt", 0); err == nil {
		t.Errorf("should have failed with no propagation flag found")
	}
	if err := points.AddPropagation(UserbindsTag, "/mnt", syscall.MS_SHARED|syscall.MS_REC); err != nil {
		t.Error(err)
	}

	points.RemoveAll()
}

func TestImport(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	mountLabel := "system_u:object_r:removable_t"
	points := &Points{}

	if err := points.SetContext(mountLabel); err != nil {
		t.Fatalf("should have passed since context is not set")
	}
	if err := points.SetContext(mountLabel); err == nil {
		t.Fatalf("should have failed since context has already been set")
	}
	if points.GetContext() != mountLabel {
		t.Fatalf("%s != %s", mountLabel, points.GetContext())
	}

	validImport := map[AuthorizedTag][]Point{
		UserbindsTag: {
			{
				Mount: specs.Mount{
					Source:      "/",
					Destination: "/mnt",
					Type:        "",
					Options:     []string{"rbind", "nosuid"},
				},
			},
		},
		KernelTag: {
			{
				Mount: specs.Mount{
					Source:      "proc",
					Destination: "/proc",
					Type:        "proc",
					Options:     []string{"nosuid", "nodev"},
				},
			},
			{
				Mount: specs.Mount{
					Source:      "sysfs",
					Destination: "/sys",
					Type:        "sysfs",
					Options:     []string{"nosuid", "nodev"},
				},
			},
		},
		SessionTag: {
			{
				Mount: specs.Mount{
					Source:      "",
					Destination: "/tmp",
					Type:        "tmpfs",
					Options:     []string{"nosuid", "nodev", "mode=1777"},
				},
			},
		},
		LayerTag: {
			{
				Mount: specs.Mount{
					Source:      "",
					Destination: "/opt",
					Type:        "overlay",
					Options:     []string{"nosuid", "nodev", "lowerdir=/", "upperdir=/upper", "workdir=/work"},
				},
			},
		},
		RootfsTag: {
			{
				Mount: specs.Mount{
					Source:      "/image.simg",
					Destination: "/tmp/image",
					Type:        "squashfs",
					Options:     []string{"nosuid", "nodev"},
				},
				InternalOptions: []string{"offset=31", "sizelimit=10"},
			},
		},
	}
	if err := points.Import(validImport); err != nil {
		t.Fatalf("%s", err)
	}
	if len(points.GetAll()) != len(validImport) {
		t.Errorf("returned a wrong number of mount points %d instead of %d", len(points.GetAll()), len(validImport))
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
	if len(bind) != 1 {
		t.Errorf("wrong number of bind mount point found")
	}
	fs := points.GetAllFS()
	if len(fs) != 3 {
		t.Errorf("wrong number of filesystem mount point found")
	}
	points.RemoveByDest("/mnt")
	all := points.GetByTag(UserbindsTag)
	if len(all) != 0 {
		t.Errorf("returned a wrong number of mount points %d instead of 0", len(all))
	}
	points.RemoveByDest("/tmp")
	all = points.GetByTag(SessionTag)
	if len(all) != 0 {
		t.Errorf("returned a wrong number of mount points %d instead of 0", len(all))
	}
	points.RemoveByDest("/opt")
	all = points.GetByTag(LayerTag)
	if len(all) != 0 {
		t.Errorf("returned a wrong number of mount points %d instead of 0", len(all))
	}
	points.RemoveBySource("/image.simg")
	all = points.GetByTag(RootfsTag)
	if len(all) != 0 {
		t.Errorf("returned a wrong number of mount points %d instead of 0", len(all))
	}

	proc := points.GetByDest("/proc")
	if len(proc) != 1 {
		t.Fatalf("returned a wrong number of mount points %d instead of 1", len(proc))
	}
	for _, option := range proc[0].Options {
		if option == "context="+mountLabel {
			t.Errorf("context should not be set for proc filesystem")
		}
	}
	points.RemoveByDest("/proc")

	sys := points.GetByDest("/sys")
	if len(sys) != 1 {
		t.Fatalf("returned a wrong number of mount points %d instead of 1", len(sys))
	}
	for _, option := range sys[0].Options {
		if option == "context="+mountLabel {
			t.Errorf("context should not be set for sysfs filesystem")
		}
	}
	points.RemoveByDest("/sys")

	all = points.GetByTag(KernelTag)
	if len(all) != 0 {
		t.Errorf("returned a wrong number of mount points %d instead of 0", len(all))
	}
	points.RemoveAll()

	invalidImport := map[AuthorizedTag][]Point{
		UserbindsTag: {
			{
				Mount: specs.Mount{
					Source:      "",
					Destination: "/mnt",
					Type:        "",
					Options:     []string{"rbind", "nosuid"},
				},
			},
		},
	}
	if err := points.Import(invalidImport); err == nil {
		t.Errorf("import should failed: %s", err)
	}

	validForceContextImport := map[AuthorizedTag][]Point{
		SessionTag: {
			{
				Mount: specs.Mount{
					Source:      "/",
					Destination: "/tmp",
					Type:        "tmpfs",
					Options:     []string{"nosuid", "nodev", "mode=1777"},
				},
			},
		},
	}

	if err := points.Import(validForceContextImport); err != nil {
		t.Fatalf("%s", err)
	}
	tmp := points.GetByDest("/tmp")
	if len(tmp) != 1 {
		t.Fatalf("returned a wrong number of mount points %d instead of 1", len(tmp))
	}
	hasContext := false
	context := fmt.Sprintf("context=%q", mountLabel)
	for _, option := range tmp[0].Options {
		if option == context {
			hasContext = true
		}
	}
	if !hasContext {
		t.Errorf("context should be set /tmp mount point")
	}
	points.RemoveAll()

	validContextImport := map[AuthorizedTag][]Point{
		SessionTag: {
			{
				Mount: specs.Mount{
					Source:      "/",
					Destination: "/tmp",
					Type:        "tmpfs",
					Options:     []string{"nosuid", "nodev", "mode=1777", "context=" + mountLabel},
				},
			},
		},
	}

	if err := points.Import(validContextImport); err != nil {
		t.Fatalf("%s", err)
	}
	tmp = points.GetByDest("/tmp")
	if len(tmp) != 1 {
		t.Fatalf("returned a wrong number of mount points %d instead of 1", len(tmp))
	}
	numContext := 0
	for _, option := range tmp[0].Options {
		if option == "context="+mountLabel {
			numContext++
		}
	}
	if numContext != 1 {
		t.Errorf("context option is set %d times for /tmp mount point %s", numContext, tmp[0])
	}
	points.RemoveAll()

	points = &Points{}

	validSpecs := []specs.Mount{
		{
			Source:      "/",
			Destination: "/mnt",
			Type:        "",
			Options:     []string{"rbind", "nosuid", "rshared"},
		},
		{
			Source:      "",
			Destination: "/opt",
			Type:        "overlay",
			Options:     []string{"nosuid", "nodev", "lowerdir=/", "upperdir=/upper", "workdir=/work"},
		},
		{
			Source:      "",
			Destination: "/tmp",
			Type:        "tmpfs",
			Options:     []string{"nosuid", "nodev", "mode=1777"},
		},
		{
			Source:      "sysfs",
			Destination: "/sys",
			Type:        "sysfs",
			Options:     []string{"nosuid", "nodev"},
		},
		{
			Source:      "",
			Destination: "/dev/pts",
			Type:        "devpts",
			Options:     []string{"nosuid"},
		},
	}
	if err := points.ImportFromSpec(validSpecs); err != nil {
		t.Error(err)
	}
	if len(points.GetByTag(KernelTag)) != 4 {
		t.Errorf("returned a wrong number of mount kernel mount points %d instead of 4", len(points.GetByTag(KernelTag)))
	}
	if len(points.GetByTag(UserbindsTag)) != 3 {
		t.Errorf("returned a wrong number of mount kernel mount points %d instead of 3", len(points.GetByTag(UserbindsTag)))
	}
	points.RemoveAll()

	invalidSpecs := []specs.Mount{
		{
			Source:      "/image.simg",
			Destination: "/tmp/image",
			Type:        "squashfs",
			Options:     []string{"nosuid", "nodev"},
		},
	}
	if err := points.ImportFromSpec(invalidSpecs); err == nil {
		t.Errorf("should have failed with non authorized filesystem type")
	}
}

func TestTag(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	points := &Points{}

	if err := points.AddBind(AuthorizedTag("unknown"), "/", "/mnt", syscall.MS_NOSUID); err == nil {
		t.Errorf("should have failed with a not recognized tag")
	}
	if err := points.AddFS(SessionTag, "/mnt", "tmpfs", syscall.MS_NOSUID, ""); err != nil {
		t.Errorf("%s", err)
	}
	if err := points.AddFS(SessionTag, "/mnt2", "tmpfs", syscall.MS_NOSUID, ""); err == nil {
		t.Errorf("should have failed, %s allow only a single mount point", SessionTag)
	}
	for _, tag := range GetTagList() {
		points.RemoveByTag(tag)
		if len(points.GetByTag(tag)) != 0 {
			t.Fatalf("removing mount point entries by tag failed")
		}
	}
}
