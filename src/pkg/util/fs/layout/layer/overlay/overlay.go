// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package overlay

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/singularityware/singularity/src/pkg/util/fs"
	"github.com/singularityware/singularity/src/pkg/util/fs/layout"
	"github.com/singularityware/singularity/src/pkg/util/fs/mount"
)

const (
	ovLowerDir = "/overlay-lowerdir"
	ovUpperDir = "/overlay-upperdir"
	ovWorkDir  = "/overlay-workdir"
)

// Overlay layer manager
type Overlay struct {
	session *layout.Session
}

// New creates and returns an overlay layer manager
func New() *Overlay {
	return &Overlay{}
}

// Add adds required directory in session layout
func (o *Overlay) Add(session *layout.Session, system *mount.System) error {
	o.session = session
	if err := o.session.AddDir(ovLowerDir); err != nil {
		return err
	}
	if err := system.RunBeforeTag(mount.LayerTag, o.createOverlay); err != nil {
		return err
	}
	return nil
}

func (o *Overlay) createOverlay(system *mount.System) error {
	lowerDirs := make([]string, 0)
	upperDir := ""
	workDir := ""

	path, _ := o.session.GetPath(ovLowerDir)
	lowerDirs = append(lowerDirs, path)

	for _, point := range system.Points.GetByTag(mount.PreLayerTag) {
		switch point.Type {
		case "ext3":
			if upperDir != "" {
				return fmt.Errorf("there is already a writable overlay image")
			}
			u := point.Destination + "/upper"
			w := point.Destination + "/work"
			if fs.IsLink(u) {
				return fmt.Errorf("symlink detected, upper overlay %s must be a directory", u)
			}
			if fs.IsLink(w) {
				return fmt.Errorf("symlink detected, work overlay %s must be a directory", w)
			}
			if !fs.IsDir(u) {
				if err := fs.MkdirAll(u, 0755); err != nil {
					return fmt.Errorf("failed to create %s directory: %s", u, err)
				}
			}
			if !fs.IsDir(w) {
				if err := fs.MkdirAll(w, 0755); err != nil {
					return fmt.Errorf("failed to create %s directory: %s", w, err)
				}
			}
			upperDir = u
			workDir = w
		case "squashfs":
			lowerDirs = append([]string{point.Destination}, lowerDirs...)
		default:
			lowerDirs = append([]string{point.Destination}, lowerDirs...)
		}
	}

	if upperDir == "" {
		if err := o.session.AddDir(ovUpperDir); err != nil {
			return fmt.Errorf("failed to add /overlay-upper directory")
		}
		if err := o.session.AddDir(ovWorkDir); err != nil {
			return fmt.Errorf("failed to add /overlay-work directory")
		}
		upperDir, _ = o.session.GetPath(ovUpperDir)
		workDir, _ = o.session.GetPath(ovWorkDir)
	}

	lowerDirs = append(lowerDirs, o.session.RootFsPath())

	lowerDir := strings.Join(lowerDirs, ":")
	err := system.Points.AddOverlay(mount.LayerTag, o.session.FinalPath(), 0, lowerDir, upperDir, workDir)
	if err != nil {
		return err
	}

	points := system.Points.GetByTag(mount.RootfsTag)
	if len(points) <= 0 {
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
			flags, _ := mount.ConvertOptions(point.Options)
			if flags&syscall.MS_REMOUNT != 0 {
				continue
			}
			if strings.HasPrefix(point.Destination, sessionDir) {
				continue
			}
			p := rootFsPath + point.Destination
			if syscall.Stat(p, st) == nil {
				continue
			}
			if err := syscall.Stat(point.Source, st); os.IsNotExist(err) {
				return fmt.Errorf("stat failed for %s: %s", point.Source, err)
			}
			dest := ovLowerDir + point.Destination
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
