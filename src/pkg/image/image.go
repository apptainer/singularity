// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package image

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/user"
)

const (
	launchString = "singularity"
	bufferSize   = 2048
)

var registeredFormats = make(map[string]format, 0)

func registerFormat(name string, f format) {
	registeredFormats[name] = f
}

// format describes the interface that an image format type must implement.
type format interface {
	initializer(*Image, os.FileInfo) error
}

// Image describes an image object
type Image struct {
	Path     string
	Name     string
	Type     int
	File     *os.File
	Offset   uint64
	Size     uint64
	Writable bool
}

// AuthorizedPath checks if image is in a path supplied in paths
func (i *Image) AuthorizedPath(paths []string) (bool, error) {
	authorized := false
	dirname := filepath.Dir(i.Path)
	for _, path := range paths {
		match, err := filepath.EvalSymlinks(filepath.Clean(path))
		if err != nil {
			return authorized, fmt.Errorf("failed to resolve path %s: %s", path, err)
		}
		if dirname == match {
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

// Init initilizes an image object based on given path
func Init(path string, writable bool) (*Image, error) {
	sylog.Debugf("Entering image format intializer")
	flags := os.O_RDONLY
	if writable {
		flags = os.O_RDWR
	}
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieved path for %s: %s", path, err)
	}
	file, err := os.OpenFile(resolvedPath, flags, 0)
	if err != nil {
		return nil, fmt.Errorf("Error while opening image %s: %s", path, err)
	}
	procPath, err := os.Readlink(fmt.Sprintf("/proc/self/fd/%d", file.Fd()))
	if err != nil {
		return nil, fmt.Errorf("failed to readlink of path file descriptor: %s", err)
	}
	if procPath != resolvedPath {
		return nil, fmt.Errorf("path doesn't match %s != %s", resolvedPath, procPath)
	}
	img := &Image{
		Path:     resolvedPath,
		Name:     filepath.Base(resolvedPath),
		File:     file,
		Writable: writable,
	}
	fileinfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if _, _, err := syscall.Syscall(syscall.SYS_FCNTL, file.Fd(), syscall.F_SETFD, syscall.O_CLOEXEC); err != 0 {
		sylog.Warningf("failed to set O_CLOEXEC flags on image")
	}
	for name, f := range registeredFormats {
		if offset, err := img.File.Seek(0, os.SEEK_SET); err != nil || offset != 0 {
			return nil, err
		}
		err := f.initializer(img, fileinfo)
		if err == nil {
			return img, nil
		}
		sylog.Debugf("%s format initializer returns: %s", name, err)
	}
	file.Close()
	return nil, fmt.Errorf("image format not recognized")
}

func init() {
	registerFormat("ext3", &ext3Format{})
	registerFormat("sandbox", &sandboxFormat{})
	registerFormat("sif", &sifFormat{})
	registerFormat("squashfs", &squashfsFormat{})
}
