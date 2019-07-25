// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package overlay

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/internal/pkg/util/fs/layout"
	"github.com/sylabs/singularity/internal/pkg/util/fs/mount"
)

const (
	lowerDir = "/overlay-lowerdir"
)

// Overlay layer manager
type Overlay struct {
	session   *layout.Session
	lowerDirs []string
	upperDir  string
	workDir   string
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

	return system.RunBeforeTag(mount.LayerTag, o.createOverlay)
}

// Dir returns absolute overlay directory within session
func (o *Overlay) Dir() string {
	return lowerDir
}

func (o *Overlay) createOverlay(system *mount.System) error {
	flags := uintptr(syscall.MS_NODEV)
	o.lowerDirs = append(o.lowerDirs, o.session.RootFsPath())

	lowerdir := strings.Join(o.lowerDirs, ":")
	err := system.Points.AddOverlay(mount.LayerTag, o.session.FinalPath(), flags, lowerdir, o.upperDir, o.workDir)
	if err != nil {
		return err
	}

	points := system.Points.GetByTag(mount.RootfsTag)
	if len(points) <= 0 {
		return fmt.Errorf("no root fs image found")
	}
	return o.createLayer(points[0].Destination, system)
}

// AddLowerDir adds a lower directory to overlay mount
func (o *Overlay) AddLowerDir(path string) error {
	o.lowerDirs = append([]string{path}, o.lowerDirs...)
	return nil
}

// SetUpperDir sets upper directory to overlay mount
func (o *Overlay) SetUpperDir(path string) error {
	if o.upperDir != "" {
		return fmt.Errorf("upper directory was already set")
	}
	o.upperDir = path
	return nil
}

// GetUpperDir returns upper directory path
func (o *Overlay) GetUpperDir() string {
	return o.upperDir
}

// SetWorkDir sets work directory to overlay mount
func (o *Overlay) SetWorkDir(path string) error {
	if o.workDir != "" {
		return fmt.Errorf("upper directory was already set")
	}
	o.workDir = path
	return nil
}

// GetWorkDir returns work directory path
func (o *Overlay) GetWorkDir() string {
	return o.workDir
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
			if point.Type == "" {
				if err := syscall.Stat(point.Source, st); err != nil {
					sylog.Warningf("skipping mount of %s: %s", point.Source, err)
					continue
				}
			}

			dest := fs.EvalRelative(point.Destination, rootFsPath)

			dest = filepath.Join(lowerDir, dest)
			if _, err := o.session.GetPath(dest); err == nil {
				continue
			}
			// don't exist create it in overlay
			switch st.Mode & syscall.S_IFMT {
			case syscall.S_IFDIR:
				if err := o.session.AddDir(dest); err != nil {
					return err
				}
			default:
				if point.Type == "" {
					if err := o.session.AddFile(dest, nil); err != nil {
						return err
					}
				} else {
					if err := o.session.AddDir(dest); err != nil {
						return err
					}
				}
			}
		}
	}
	return o.session.Update()
}
