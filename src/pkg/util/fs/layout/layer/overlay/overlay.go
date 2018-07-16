// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package overlay

import (
	"fmt"
	"strings"
	"syscall"

	"github.com/singularityware/singularity/src/pkg/util/fs/layout"
	"github.com/singularityware/singularity/src/pkg/util/fs/mount"
)

const lowerDir = "/overlay-lowerdir"

// Overlay layer manager
type Overlay struct {
	session   *layout.Session
	lowerDirs []string
}

// New creates and returns an overlay layer manager
func New() *Overlay {
	return &Overlay{}
}

// Add adds required directory in session layout
func (o *Overlay) Add(session *layout.Session, system *mount.System) error {
	o.session = session
	if err := o.session.AddDir(lowerDir); err != nil {
		return err
	}
	if o.lowerDirs == nil {
		o.lowerDirs = make([]string, 0)
	}
	path, _ := o.session.GetPath(lowerDir)
	o.lowerDirs = append(o.lowerDirs, path)

	if err := system.RunBeforeTag(mount.PreLayerTag, o.createOverlay); err != nil {
		return err
	}
	return nil
}

func (o *Overlay) createOverlay(system *mount.System) error {
	lowerdir := fmt.Sprintf("%s:%s", strings.Join(o.lowerDirs, ":"), o.session.RootFsPath())
	err := system.Points.AddOverlay(mount.PreLayerTag, o.session.FinalPath(), 0, lowerdir, "", "")
	if err != nil {
		return err
	}

	points := system.Points.GetByTag(mount.RootfsTag)
	if len(points) != 1 {
		return fmt.Errorf("no root fs image found")
	}
	return o.createLayer(points[0].Destination, system)
}

// createLayer creates overlay layer based on content of root filesystem
// given by rootFsPath
func (o *Overlay) createLayer(rootFsPath string, system *mount.System) error {
	sessionDir := o.session.Path()
	st := new(syscall.Stat_t)

	if sessionDir == "" {
		return fmt.Errorf("can't determine session path")
	}
	for _, tag := range mount.GetTagList() {
		for _, point := range system.Points.GetByTag(tag) {
			if strings.HasPrefix(point.Destination, sessionDir) {
				continue
			}
			p := rootFsPath + point.Destination
			if err := syscall.Stat(p, st); err == nil {
				continue
			}
			if err := syscall.Stat(point.Source, st); err != nil {
				return err
			}
			dest := lowerDir + point.Destination
			// don't exist create it in overlay
			switch st.Mode & syscall.S_IFMT {
			case syscall.S_IFDIR:
				if err := o.session.AddDir(dest); err != nil {
					return err
				}
			default:
				if err := o.session.AddFile(dest, nil); err != nil {
					return err
				}
			}
		}
	}
	return o.session.Update()
}
