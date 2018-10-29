// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package layout

import (
	"fmt"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/util/fs/mount"
)

const rootFsDir = "/rootfs"
const finalDir = "/final"

// Session directory layout manager
type Session struct {
	*Manager
	Layer layer
}

// Layer describes a layer interface added on top of session layout
type layer interface {
	Add(*Session, *mount.System) error
}

// NewSession creates and returns a session directory layout manager
func NewSession(path string, fstype string, size int, system *mount.System, layer layer) (*Session, error) {
	manager := &Manager{}
	session := &Session{Manager: manager}

	if err := manager.SetRootPath(path); err != nil {
		return nil, err
	}
	if err := manager.AddDir(rootFsDir); err != nil {
		return nil, err
	}
	if err := manager.AddDir(finalDir); err != nil {
		return nil, err
	}
	options := "mode=1777"
	if size >= 0 {
		options = fmt.Sprintf("mode=1777,size=%dm", size)
	}
	err := system.Points.AddFS(mount.SessionTag, path, fstype, syscall.MS_NOSUID, options)
	if err != nil {
		return nil, err
	}
	if err := system.RunAfterTag(mount.SessionTag, session.createLayout); err != nil {
		return nil, err
	}
	if layer != nil {
		if err := layer.Add(session, system); err != nil {
			return nil, fmt.Errorf("failed to init layer: %s", err)
		}
		session.Layer = layer
	}
	return session, nil
}

// Path returns the full path of session directory
func (s *Session) Path() string {
	path, _ := s.GetPath("/")
	return path
}

// FinalPath returns the full path to session final directory
func (s *Session) FinalPath() string {
	if s.Layer != nil {
		path, _ := s.GetPath(finalDir)
		return path
	}
	return s.RootFsPath()
}

// RootFsPath returns the full path to session rootfs directory
func (s *Session) RootFsPath() string {
	path, _ := s.GetPath(rootFsDir)
	return path
}

func (s *Session) createLayout(system *mount.System) error {
	return s.Create()
}
