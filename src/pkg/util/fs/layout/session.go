// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package layout

import (
	"fmt"
	"syscall"

	"github.com/singularityware/singularity/src/pkg/util/fs/mount"
)

const rootFsDir = "/rootfs"
const finalDir = "/final"

// Session directory layout manager
type Session struct {
	*Manager
	layer layer
}

// Layer describes a layer interface added on top of session layout
type layer interface {
	Add(*Session) error
	Prepare(*mount.System) error
}

// NewSession creates and returns a session directory layout manager
func NewSession(path string, fstype string, system *mount.System, layer layer) (*Session, error) {
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
	if layer != nil {
		if err := layer.Add(session); err != nil {
			return nil, fmt.Errorf("failed to init layer: %s", err)
		}
		session.layer = layer
	}
	if err := session.prepare(system, fstype); err != nil {
		return nil, err
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
	if s.layer != nil {
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

func (s *Session) prepare(system *mount.System, fstype string) error {
	err := system.Points.AddFS(mount.SessionTag, s.Path(), fstype, syscall.MS_NOSUID|syscall.MS_NODEV, "mode=1777")
	if err != nil {
		return err
	}
	if err := system.RunAfterTag(mount.SessionTag, s.createLayout); err != nil {
		return err
	}
	if s.layer != nil {
		if err := s.layer.Prepare(system); err != nil {
			return fmt.Errorf("failed to call layer Prepare: %s", err)
		}
	}
	return nil
}

func (s *Session) createLayout(system *mount.System) error {
	return s.Create()
}
