// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package layout

import (
	"strings"
	"syscall"

	"github.com/singularityware/singularity/src/pkg/util/fs/mount"
)

// Overlay directory layout manager
type Overlay struct {
	*Manager
	session *Session
}

// NewOverlay creates and returns an overlay directory layout manager
func NewOverlay(slayout *Session) (overlay *Overlay, err error) {
	manager := &Manager{}
	overlay = &Overlay{Manager: manager}

	if err = slayout.AddDir("/overlay-lowerdir"); err != nil {
		return
	}
	if err = slayout.Update(); err != nil {
		return
	}
	overlayRootPath, _ := slayout.GetPath("/overlay-lowerdir")
	if err = manager.SetRootPath(overlayRootPath); err != nil {
		return
	}
	return
}

// Path returns path of overlay directory layout
func (o *Overlay) Path() string {
	p, _ := o.GetPath("/")
	return p
}

// CreateLayout creates overlay layout based on content of root filesystem
// given by rootFsPath
func (o *Overlay) CreateLayout(rootFsPath string, points *mount.Points) error {
	sessionDir, err := o.session.GetPath("/")
	st := new(syscall.Stat_t)

	if err != nil {
		return err
	}
	for _, tag := range mount.GetTagList() {
		for _, point := range points.GetByTag(tag) {
			if strings.HasPrefix(point.Destination, sessionDir) {
				continue
			}
			p := rootFsPath + point.Destination
			if err := syscall.Stat(p, st); err == nil {
				continue
			}
			// don't exist create it in overlay
			switch st.Mode & syscall.S_IFMT {
			case syscall.S_IFDIR:
				if err := o.AddDir(point.Destination); err != nil {
					return err
				}
			case syscall.S_IFREG:
				if err := o.AddFile(point.Destination, nil); err != nil {
					return err
				}
			}
		}
	}
	return o.Create()
}
