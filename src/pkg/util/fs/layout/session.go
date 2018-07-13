// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package layout

import "fmt"

// Session directory layout manager
type Session struct {
	*Manager
	imageNumber uint
}

// NewSession creates and returns a session directory layout manager
func NewSession(path string) (session *Session, err error) {
	manager := &Manager{}
	session = &Session{Manager: manager}

	if err = manager.SetRootPath(path); err != nil {
		return
	}
	if err = manager.AddDir("/container"); err != nil {
		return
	}
	if err = manager.AddDir("/final"); err != nil {
		return
	}
	if err = manager.AddDir("/overlay"); err != nil {
		return
	}
	if err = manager.AddDir("/images"); err != nil {
		return
	}
	return
}

// ContainerPath returns the full path to session container directory
func (s *Session) ContainerPath() string {
	path, _ := s.GetPath("/container")
	return path
}

// OverlayPath returns the full path to session overlay directory
func (s *Session) OverlayPath() string {
	path, _ := s.GetPath("/overlay")
	return path
}

// FinalPath returns the full path to session final directory
func (s *Session) FinalPath() string {
	path, _ := s.GetPath("/final")
	return path
}

// AddImage adds a directory in session for dynamic image binding
func (s *Session) AddImage() string {
	p := fmt.Sprintf("/images/%d", s.imageNumber)
	if err := s.AddDir(p); err != nil {
		return ""
	}
	s.imageNumber++
	path, _ := s.GetPath(p)
	return path
}
