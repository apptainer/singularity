// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package image

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/user"
)

const (
	// SQUASHFS constant for squashfs format
	SQUASHFS = iota + 0x1000
	// EXT3 constant for ext3 format
	EXT3
	// SANDBOX constant for directory format
	SANDBOX
	// SIF constant for sif format
	SIF
	// ENCRYPTSQUASHFS constant for encrypted squashfs format
	ENCRYPTSQUASHFS
)

const (
	// RootFs partition name
	RootFs       = "!__rootfs__!"
	launchString = " run-singularity"
	bufferSize   = 2048
)

// debugError represents an error considered for debugging
// purpose rather than real error, this helps to distinguish
// those errors between real image format error during
// initializer loop.
type debugError string

func (e debugError) Error() string { return string(e) }

func debugErrorf(format string, a ...interface{}) error {
	e := fmt.Sprintf(format, a...)
	return debugError(e)
}

// ErrUnknownFormat represents an unknown image format error.
var ErrUnknownFormat = errors.New("image format not recognized")

var registeredFormats = []struct {
	name   string
	format format
}{
	{"sandbox", &sandboxFormat{}},
	{"sif", &sifFormat{}},
	{"squashfs", &squashfsFormat{}},
	{"ext3", &ext3Format{}},
}

// format describes the interface that an image format type must implement.
type format interface {
	openMode(bool) int
	initializer(*Image, os.FileInfo) error
}

// Section identifies and locates a data section in image object.
type Section struct {
	Size   uint64 `json:"size"`
	Offset uint64 `json:"offset"`
	Type   uint32 `json:"type"`
	Name   string `json:"name"`
}

// Image describes an image object, an image is composed of one
// or more partitions (eg: container root filesystem, overlay),
// image format like SIF contains descriptors pointing to chunk of
// data, chunks position and size are stored as image sections.
type Image struct {
	Path       string    `json:"path"`
	Name       string    `json:"name"`
	Type       int       `json:"type"`
	File       *os.File  `json:"-"`
	Fd         uintptr   `json:"fd"`
	Source     string    `json:"source"`
	Writable   bool      `json:"writable"`
	Partitions []Section `json:"partitions"`
	Sections   []Section `json:"sections"`
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

// AuthorizedOwner checks whether the image is owned by any user from the supplied users list.
func (i *Image) AuthorizedOwner(owners []string) (bool, error) {
	fileinfo, err := i.File.Stat()
	if err != nil {
		return false, fmt.Errorf("failed to get stat for %s", i.Path)
	}

	uid := fileinfo.Sys().(*syscall.Stat_t).Uid
	for _, owner := range owners {
		pw, err := user.GetPwNam(owner)
		if err != nil {
			return false, fmt.Errorf("failed to retrieve user information for %s: %s", owner, err)
		}
		if pw.UID == uid {
			return true, nil
		}
	}
	return false, nil
}

// AuthorizedGroup checks whether the image is owned by any group from the supplied groups list.
func (i *Image) AuthorizedGroup(groups []string) (bool, error) {
	fileinfo, err := i.File.Stat()
	if err != nil {
		return false, fmt.Errorf("failed to get stat for %s", i.Path)
	}

	gid := fileinfo.Sys().(*syscall.Stat_t).Gid
	for _, group := range groups {
		gr, err := user.GetGrNam(group)
		if err != nil {
			return false, fmt.Errorf("failed to retrieve group information for %s: %s", group, err)
		}
		if gr.GID == gid {
			return true, nil
		}
	}
	return false, nil
}

// HasRootFs returns true if image contains a root filesystem partition.
func (i *Image) HasRootFs() bool {
	for _, p := range i.Partitions {
		if p.Name == RootFs {
			return true
		}
	}
	return false
}

// ResolvePath returns a resolved absolute path.
func ResolvePath(path string) (string, error) {
	abspath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %s", err)
	}
	resolvedPath, err := filepath.EvalSymlinks(abspath)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve path for %s: %s", path, err)
	}
	return resolvedPath, nil
}

// Init initializes an image object based on given path.
func Init(path string, writable bool) (*Image, error) {
	sylog.Debugf("Image format detection")

	resolvedPath, err := ResolvePath(path)
	if err != nil {
		return nil, err
	}

	img := &Image{
		Path: resolvedPath,
		Name: filepath.Base(resolvedPath),
	}

	for _, rf := range registeredFormats {
		sylog.Debugf("Check for %s image format", rf.name)

		img.Writable = writable

		mode := rf.format.openMode(writable)

		if mode&os.O_RDWR != 0 {
			if err := syscall.Access(resolvedPath, 2); err != nil {
				sylog.Debugf("Opening %s in read-only mode: no write permissions", path)
				mode = os.O_RDONLY
				img.Writable = false
			}
		}

		img.File, err = os.OpenFile(resolvedPath, mode, 0)
		if err != nil {
			continue
		}
		fileinfo, err := img.File.Stat()
		if err != nil {
			_ = img.File.Close()
			return nil, err
		}

		err = rf.format.initializer(img, fileinfo)
		if _, ok := err.(debugError); ok {
			sylog.Debugf("%s format initializer returned: %s", rf.name, err)
			_ = img.File.Close()
			continue
		} else if err != nil {
			_ = img.File.Close()
			return nil, err
		}

		sylog.Debugf("%s image format detected", rf.name)

		if _, _, err := syscall.Syscall(syscall.SYS_FCNTL, img.File.Fd(), syscall.F_SETFD, syscall.O_CLOEXEC); err != 0 {
			sylog.Warningf("failed to set O_CLOEXEC flags on image")
		}

		img.Source = fmt.Sprintf("/proc/self/fd/%d", img.File.Fd())
		img.Fd = img.File.Fd()

		return img, nil
	}
	return nil, ErrUnknownFormat
}
