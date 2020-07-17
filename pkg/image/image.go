// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
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

	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/internal/pkg/util/user"
	"github.com/sylabs/singularity/pkg/sylog"
	"github.com/sylabs/singularity/pkg/util/fs/lock"
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
	// RAW constant for raw format
	RAW
)

type Usage uint8

const (
	// RootFsUsage defines flag for image/partition
	// usable as root filesystem.
	RootFsUsage = Usage(1 << iota)
	// OverlayUsage defines flag for image/partition
	// usable as overlay.
	OverlayUsage
	// DataUsage defines flag for image/partition
	// usable as data.
	DataUsage
)

const (
	// RootFs partition name
	RootFs       = "!__rootfs__!"
	launchString = " run-singularity"
	bufferSize   = 2048
	emptyFd      = ^uintptr(0)
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

// readOnlyFilesystemError represents an error returned by
// read-only filesystem image when attempted to be opened
// as writable.
type readOnlyFilesystemError struct {
	s string
}

func (e *readOnlyFilesystemError) Error() string {
	return e.s
}

// IsReadOnlyFilesytem returns if the corresponding error
// is a read-only filesystem error or not.
func IsReadOnlyFilesytem(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*readOnlyFilesystemError)
	return ok
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
	lock(*Image) error
}

// Section identifies and locates a data section in image object.
type Section struct {
	Name         string `json:"name"`
	Size         uint64 `json:"size"`
	Offset       uint64 `json:"offset"`
	ID           uint32 `json:"id"`
	Type         uint32 `json:"type"`
	AllowedUsage Usage  `json:"allowed_usage"`
}

// Image describes an image object, an image is composed of one
// or more partitions (eg: container root filesystem, overlay),
// image format like SIF contains descriptors pointing to chunk of
// data, chunks position and size are stored as image sections.
type Image struct {
	Partitions []Section `json:"partitions"`
	Sections   []Section `json:"sections"`
	Path       string    `json:"path"`
	Name       string    `json:"name"`
	Source     string    `json:"source"`
	Type       int       `json:"type"`
	File       *os.File  `json:"-"`
	Fd         uintptr   `json:"fd"`
	Writable   bool      `json:"writable"`
	Usage      Usage     `json:"usage"`
}

// AuthorizedPath checks if image is in a path supplied in paths
func (i *Image) AuthorizedPath(paths []string) (bool, error) {
	if err := i.initFile(); err != nil {
		return false, err
	}

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
	if err := i.initFile(); err != nil {
		return false, err
	}

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
	if err := i.initFile(); err != nil {
		return false, err
	}

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

// getPartitions returns partitions based on their usage.
func (i *Image) getPartitions(usage Usage) ([]Section, error) {
	sections := make([]Section, 0)

	if i.Usage&usage == 0 {
		return sections, nil
	}

	if err := i.initFile(); err != nil {
		return nil, err
	}

	for _, p := range i.Partitions {
		if p.AllowedUsage&usage != 0 {
			sections = append(sections, p)
		}
	}

	return sections, nil
}

// GetAllPartitions returns all partitions found in the image.
func (i *Image) GetAllPartitions() ([]Section, error) {
	return i.getPartitions(RootFsUsage | OverlayUsage | DataUsage)
}

// GetRootFsPartition returns the first root filesystem partition
// found in the image.
func (i *Image) GetRootFsPartition() (*Section, error) {
	partitions, err := i.GetRootFsPartitions()
	if err != nil {
		return nil, err
	} else if len(partitions) == 0 {
		return nil, fmt.Errorf("no root filesystem found")
	}
	return &partitions[0], nil
}

// GetRootFsPartitions returns root filesystem partitions found
// in the image.
func (i *Image) GetRootFsPartitions() ([]Section, error) {
	return i.getPartitions(RootFsUsage)
}

// GetOverlayPartitions returns overlay partitions found in the image.
func (i *Image) GetOverlayPartitions() ([]Section, error) {
	return i.getPartitions(OverlayUsage)
}

// GetDataPartitions returns data partitions found in the image.
func (i *Image) GetDataPartitions() ([]Section, error) {
	return i.getPartitions(DataUsage)
}

// initFile ensures file descriptor is associated to a file handle.
func (i *Image) initFile() error {
	if i.File != nil {
		return nil
	}
	if i.Path == "" {
		return fmt.Errorf("no image path")
	}
	if i.Fd == emptyFd || i.Source == "" {
		return fmt.Errorf("%s is not open", i.Path)
	}
	if err := os.NewFile(i.Fd, i.Path); err == nil {
		return fmt.Errorf("image file descriptor for %s is not valid", i.Path)
	}
	return nil
}

// writeLocks tracks write locks for the current process.
var writeLocks = make(map[string][]Section)

// readLocks tracks read locks for the current process.
var readLocks = make(map[string][]Section)

// lockSection puts a file byte-range lock on a section to prevent
// from concurrent writes depending if the image is writable or
// not. If the image is writable, calling this function will place
// a write lock for the corresponding section preventing further use
// if the section is used for writing or reading only, if the image is
// not writable this function place a read lock to prevent section
// from being written while the section is used in read-only mode.
func lockSection(i *Image, section Section) error {
	fd := int(i.Fd)
	start := int64(section.Offset)
	size := int64(section.Size)

	br := lock.NewByteRange(fd, start, size)

	var err error

	if i.Writable {
		err = br.Lock()
		if err == nil {
			// sadly we need to track same write locks from
			// the same process because a process may place
			// as many write lock without any error
			if sections, ok := readLocks[i.Path]; ok {
				for _, s := range sections {
					if s.Offset == section.Offset && s.Size == section.Size {
						return fmt.Errorf("can't open %s for writing, already used for reading by this process", i.Path)
					}
				}
			}
			if sections, ok := writeLocks[i.Path]; ok {
				for _, s := range sections {
					if s.Offset == section.Offset && s.Size == section.Size {
						return fmt.Errorf("can't open %s for writing, already used for writing by this process", i.Path)
					}
				}
			}
			writeLocks[i.Path] = append(writeLocks[i.Path], section)
		}
	} else {
		err = br.RLock()
		if err == nil {
			if sections, ok := writeLocks[i.Path]; ok {
				for _, s := range sections {
					if s.Offset == section.Offset && s.Size == section.Size {
						return fmt.Errorf("can't open %s for reading, already used for writing by this process", i.Path)
					}
				}
			}
			readLocks[i.Path] = append(readLocks[i.Path], section)
		}
	}

	if err == lock.ErrByteRangeAcquired {
		if i.Writable {
			return fmt.Errorf("can't open %s for writing, currently in use by another process", i.Path)
		}
		return fmt.Errorf("can't open %s for reading, currently in use for writing by another process", i.Path)
	} else if err == lock.ErrLockNotSupported {
		// ENOLCK means that the underlying filesystem doesn't support
		// lock, so we simply ignore the error in order to allow ext3
		// images located on the underlying filesystem to run correctly
		// and advertise user in log
		sylog.Verbosef("Could not set lock on %s section %q, underlying filesystem seems to not support lock", i.Path, section.Name)
		sylog.Verbosef("Data corruptions may occur if %s is open for writing by multiple processes", i.Path)
		return nil
	}

	return err
}

// ResolvePath returns a resolved absolute path.
func ResolvePath(path string) (string, error) {
	abspath, err := fs.Abs(path)
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
		Path:  resolvedPath,
		Name:  filepath.Base(resolvedPath),
		Fd:    emptyFd,
		Usage: RootFsUsage,
	}

	for _, rf := range registeredFormats {
		sylog.Debugf("Check for %s image format", rf.name)

		img.Writable = writable

		mode := rf.format.openMode(writable)

		if mode&os.O_RDWR != 0 {
			if !fs.IsWritable(resolvedPath) {
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

		// readOnlyFilesystemError is allowed here and passed back
		// to the caller because there is basically no error with
		// the image format just a mismatch with writable parameter,
		// so the decision is delegated to the caller
		initErr := rf.format.initializer(img, fileinfo)
		if _, ok := initErr.(debugError); ok {
			sylog.Debugf("%s format initializer returned: %v", rf.name, initErr)
			_ = img.File.Close()
			continue
		} else if initErr != nil && !IsReadOnlyFilesytem(initErr) {
			_ = img.File.Close()
			return nil, initErr
		}

		sylog.Debugf("%s image format detected", rf.name)

		if _, _, err := syscall.Syscall(syscall.SYS_FCNTL, img.File.Fd(), syscall.F_SETFD, syscall.O_CLOEXEC); err != 0 {
			sylog.Warningf("failed to set O_CLOEXEC flags on image")
		}

		img.Source = fmt.Sprintf("/proc/self/fd/%d", img.File.Fd())
		img.Fd = img.File.Fd()

		if err := rf.format.lock(img); err != nil {
			_ = img.File.Close()
			return nil, err
		}

		return img, initErr
	}

	return nil, ErrUnknownFormat
}
