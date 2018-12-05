// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package image

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/user"
)

const (
	launchString = " run-singularity"
	bufferSize   = 2048
)

var registeredFormats = []struct {
	name   string
	format format
}{
	{"sif", &sifFormat{}},
	{"sandbox", &sandboxFormat{}},
	{"squashfs", &squashfsFormat{}},
	{"ext3", &ext3Format{}},
}

// format describes the interface that an image format type must implement.
type format interface {
	openMode(bool) int
	initializer(*Image, os.FileInfo) error
}

// Image describes an image object
type Image struct {
	Path     string   `json:"path"`
	Name     string   `json:"name"`
	Type     int      `json:"type"`
	File     *os.File `json:"-"`
	Fd       uintptr  `json:"fd"`
	Source   string   `json:"source"`
	Offset   uint64   `json:"offset"`
	Size     uint64   `json:"size"`
	Writable bool     `json:"writable"`
	RootFS   bool     `json:"rootFS"`
}

// AuthorizedPath checks if image is in a path supplied in paths
func (i *Image) AuthorizedPath(paths []string) (bool, error) {
	authorized := false
	dirname := i.Path

	for _, path := range paths {
		match, err := filepath.EvalSymlinks(filepath.Clean(path))
		if err != nil {
			return authorized, fmt.Errorf("failed to resolve path %s: %s", path, err)
		}
		if strings.HasPrefix(dirname, match) {
			authorized = true
			break
		}
	}
	return authorized, nil
}

// AuthorizedOwner checks if image is owned by user supplied in users list
func (i *Image) AuthorizedOwner(owners []string) (bool, error) {
	authorized := false
	fileinfo, err := i.File.Stat()
	if err != nil {
		return authorized, fmt.Errorf("failed to get stat for %s", i.Path)
	}
	uid := fileinfo.Sys().(*syscall.Stat_t).Uid
	for _, owner := range owners {
		pw, err := user.GetPwNam(owner)
		if err != nil {
			return authorized, fmt.Errorf("failed to retrieve user information for %s: %s", owner, err)
		}
		if pw.UID == uid {
			authorized = true
			break
		}
	}
	return authorized, nil
}

// AuthorizedGroup checks if image is owned by group supplied in groups list
func (i *Image) AuthorizedGroup(groups []string) (bool, error) {
	authorized := false
	fileinfo, err := i.File.Stat()
	if err != nil {
		return authorized, fmt.Errorf("failed to get stat for %s", i.Path)
	}
	gid := fileinfo.Sys().(*syscall.Stat_t).Gid
	for _, group := range groups {
		gr, err := user.GetGrNam(group)
		if err != nil {
			return authorized, fmt.Errorf("failed to retrieve group information for %s: %s", group, err)
		}
		if gr.GID == gid {
			authorized = true
			break
		}
	}
	return authorized, nil
}

// ResolvePath returns a resolved absolute path
func ResolvePath(path string) (string, error) {
	abspath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %s", err)
	}
	resolvedPath, err := filepath.EvalSymlinks(abspath)
	if err != nil {
		return "", fmt.Errorf("failed to retrieved path for %s: %s", path, err)
	}
	return resolvedPath, nil
}

// Init initilizes an image object based on given path
func Init(path string, writable bool) (*Image, error) {
	sylog.Debugf("Entering image format intializer")

	resolvedPath, err := ResolvePath(path)
	if err != nil {
		return nil, err
	}

	img := &Image{
		Path: resolvedPath,
		Name: filepath.Base(resolvedPath),
	}

	for _, rf := range registeredFormats {
		sylog.Debugf("Check for image format %s", rf.name)

		img.Writable = writable

		mode := rf.format.openMode(writable)

		if mode&os.O_RDWR != 0 {
			if err := syscall.Access(resolvedPath, 2); err != nil {
				sylog.Debugf("Opening %s in read-only mode: no write permissions", path)
			}
			mode = os.O_RDONLY
			img.Writable = false
		}

		img.File, err = os.OpenFile(resolvedPath, mode, 0)
		if err != nil {
			continue
		}
		fileinfo, err := img.File.Stat()
		if err != nil {
			img.File.Close()
			return nil, err
		}

		err = rf.format.initializer(img, fileinfo)
		if err != nil {
			sylog.Debugf("%s format initializer returns: %s", rf.name, err)
			img.File.Close()
			continue
		}
		if _, _, err := syscall.Syscall(syscall.SYS_FCNTL, img.File.Fd(), syscall.F_SETFD, syscall.O_CLOEXEC); err != 0 {
			sylog.Warningf("failed to set O_CLOEXEC flags on image")
		}

		img.Source = fmt.Sprintf("/proc/self/fd/%d", img.File.Fd())
		img.Fd = img.File.Fd()

		return img, nil
	}
	return nil, fmt.Errorf("image format not recognized")
}
