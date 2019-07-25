// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package underlay

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/sylog"

	"github.com/sylabs/singularity/internal/pkg/util/fs/layout"
	"github.com/sylabs/singularity/internal/pkg/util/fs/mount"
)

const underlayDir = "/underlay"

type pathLen struct {
	path string
	len  uint16
}

// Underlay layer manager
type Underlay struct {
	session *layout.Session
}

// New creates and returns an overlay layer manager
func New() *Underlay {
	return &Underlay{}
}

// Add adds required directory in session layout
func (u *Underlay) Add(session *layout.Session, system *mount.System) error {
	u.session = session
	if err := u.session.AddDir(underlayDir); err != nil {
		return err
	}
	return system.RunBeforeTag(mount.PreLayerTag, u.createUnderlay)
}

// Dir returns absolute underlay directory within session
func (u *Underlay) Dir() string {
	return underlayDir
}

func (u *Underlay) createUnderlay(system *mount.System) error {
	points := system.Points.GetByTag(mount.RootfsTag)
	if len(points) <= 0 {
		return fmt.Errorf("no root fs image found")
	}
	return u.createLayer(points[0].Destination, system)
}

// createLayer creates underlay layer based on content of root filesystem
func (u *Underlay) createLayer(rootFsPath string, system *mount.System) error {
	st := new(syscall.Stat_t)
	points := system.Points
	createdPath := make([]pathLen, 0)

	sessionDir := u.session.Path()
	for _, tag := range mount.GetTagList() {
		for _, point := range points.GetByTag(tag) {
			flags, _ := mount.ConvertOptions(point.Options)
			if flags&syscall.MS_REMOUNT != 0 {
				continue
			}
			if strings.HasPrefix(point.Destination, sessionDir) {
				continue
			}
			if err := syscall.Stat(rootFsPath+point.Destination, st); err == nil {
				continue
			}
			if err := syscall.Stat(point.Source, st); err != nil {
				sylog.Warningf("skipping mount of %s: %s", point.Source, err)
				continue
			}
			dst := underlayDir + point.Destination
			if _, err := u.session.GetPath(dst); err == nil {
				continue
			}
			switch st.Mode & syscall.S_IFMT {
			case syscall.S_IFDIR:
				if err := u.session.AddDir(dst); err != nil {
					return err
				}
			default:
				if err := u.session.AddFile(dst, nil); err != nil {
					return err
				}
			}
			createdPath = append(createdPath, pathLen{path: point.Destination, len: uint16(strings.Count(point.Destination, "/"))})
		}
	}

	sort.SliceStable(createdPath, func(i, j int) bool { return createdPath[i].len < createdPath[j].len })

	for _, pl := range createdPath {
		splitted := strings.Split(filepath.Dir(pl.path), string(os.PathSeparator))
		l := len(splitted)
		p := ""
		for i := 1; i < l; i++ {
			s := splitted[i : i+1][0]
			p += "/" + s
			if s != "" {
				if _, err := u.session.GetPath(p); err != nil {
					if err := u.session.AddDir(p); err != nil {
						return err
					}
				}
				if err := u.duplicateDir(p, system, pl.path); err != nil {
					return err
				}
			}
		}
	}

	if err := u.duplicateDir("/", system, ""); err != nil {
		return err
	}

	flags := uintptr(syscall.MS_BIND | syscall.MS_REC | syscall.MS_RDONLY)
	path, _ := u.session.GetPath(underlayDir)

	err := system.Points.AddBind(mount.LayerTag, path, u.session.FinalPath(), flags)
	if err != nil {
		return err
	}
	err = system.Points.AddRemount(mount.LayerTag, u.session.FinalPath(), flags)
	if err != nil {
		return err
	}

	return u.session.Update()
}

func (u *Underlay) duplicateDir(dir string, system *mount.System, existingPath string) error {
	binds := 0
	path := filepath.Clean(u.session.RootFsPath() + dir)
	files, err := ioutil.ReadDir(path)
	if err != nil {
		// directory doesn't exists, nothing to duplicate
		return nil
	}
	for _, file := range files {
		dst := filepath.Join(underlayDir+dir, file.Name())
		src := filepath.Join(path, file.Name())

		// no error means entry is already created
		if _, err := u.session.GetPath(dst); err == nil {
			continue
		}
		if file.IsDir() {
			if err := u.session.AddDir(dst); err != nil {
				return fmt.Errorf("can't add directory %s to underlay: %s", dst, err)
			}
			dst, _ = u.session.GetPath(dst)
			if err := system.Points.AddBind(mount.PreLayerTag, src, dst, syscall.MS_BIND); err != nil {
				return fmt.Errorf("can't add bind mount point: %s", err)
			}
			binds++
		} else if file.Mode()&os.ModeSymlink != 0 {
			tgt, err := os.Readlink(src)
			if err != nil {
				return fmt.Errorf("can't read symlink information for %s: %s", src, err)
			}
			if err := u.session.AddSymlink(dst, tgt); err != nil {
				return fmt.Errorf("can't add symlink: %s", err)
			}
		} else {
			if err := u.session.AddFile(dst, nil); err != nil {
				return fmt.Errorf("can't add directory %s to underlay: %s", dst, err)
			}
			dst, _ = u.session.GetPath(dst)
			if err := system.Points.AddBind(mount.PreLayerTag, src, dst, syscall.MS_BIND); err != nil {
				return fmt.Errorf("can't add bind mount point: %s", err)
			}
			binds++
		}
	}
	if binds > 50 && existingPath != "" {
		sylog.Warningf("underlay of %s required more than 50 (%d) bind mounts", existingPath, binds)
	}
	return nil
}
