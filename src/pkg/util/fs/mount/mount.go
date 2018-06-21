// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package mount

import (
	"fmt"
	"strings"
	"syscall"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

var mountFlags = []struct {
	option string
	flag   uintptr
}{
	{"acl", 0},
	{"async", 0},
	{"atime", 0},
	{"bind", syscall.MS_BIND},
	{"defaults", 0},
	{"dev", 0},
	{"diratime", 0},
	{"dirsync", 0},
	{"exec", 0},
	{"iversion", 0},
	{"lazytime", 0},
	{"loud", 0},
	{"mand", 0},
	{"noacl", 0},
	{"noatime", 0},
	{"nodev", syscall.MS_NODEV},
	{"nodiratime", 0},
	{"noexec", syscall.MS_NOEXEC},
	{"noiversion", 0},
	{"nolazytime", 0},
	{"nomand", 0},
	{"norelatime", 0},
	{"nostrictatime", 0},
	{"nosuid", syscall.MS_NOSUID},
	{"rbind", syscall.MS_BIND | syscall.MS_REC},
	{"relatime", 0},
	{"remount", syscall.MS_REMOUNT},
	{"ro", syscall.MS_RDONLY},
	{"rw", 0},
	{"silent", syscall.MS_SILENT},
	{"strictatime", 0},
	{"suid", 0},
	{"sync", 0},
}

type fsContext struct {
	context bool
}

var authorizedImage = map[string]fsContext{
	"ext3":     {true},
	"squashfs": {true},
}

var authorizedFS = map[string]fsContext{
	"overlay": {true},
	"tmpfs":   {true},
	"ramfs":   {true},
	"devpts":  {true},
	"sysfs":   {false},
	"proc":    {false},
	"mqueue":  {false},
}

// ConvertOptions converts an options string into a pair of mount flags and mount options
func ConvertOptions(options []string) (uintptr, []string) {
	var flags uintptr
	finalOptions := []string{}
	isFlag := false

	for _, option := range options {
		optionTrim := strings.TrimSpace(option)
		for _, flag := range mountFlags {
			if flag.option == optionTrim {
				flags |= flag.flag
				isFlag = true
				break
			}
		}
		if !isFlag {
			finalOptions = append(finalOptions, optionTrim)
		}
		isFlag = false
	}
	return flags, finalOptions
}

// Points defines and stores a set of mount points
type Points struct {
	Context string
	points  []specs.Mount
}

func (p *Points) add(source string, dest string, fstype string, flags uintptr, options string) error {
	var bind = false

	mountOptions := []string{}

	if dest == "" {
		return fmt.Errorf("mount point must contain a destination")
	}
	if !strings.HasPrefix(dest, "/") {
		return fmt.Errorf("destination must be an absolute path")
	}
	for i := len(mountFlags) - 1; i >= 0; i-- {
		flag := mountFlags[i].flag
		if flag != 0 && flag == (flags&flag) {
			if bind && flag&syscall.MS_BIND != 0 {
				continue
			}
			mountOptions = append(mountOptions, mountFlags[i].option)
			if flag&syscall.MS_BIND != 0 {
				bind = true
			}
		}
	}
	setContext := true
	for _, option := range strings.Split(options, ",") {
		o := strings.TrimSpace(option)
		if o != "" {
			if strings.HasPrefix(o, "context=") {
				setContext = false
			}
			mountOptions = append(mountOptions, o)
		}
	}
	if fstype != "" && setContext {
		setContext = authorizedFS[fstype].context
	}
	if setContext {
		context := fmt.Sprintf("context=%s", p.Context)
		mountOptions = append(mountOptions, context)
	}
	p.points = append(p.points, specs.Mount{
		Source:      source,
		Destination: dest,
		Type:        fstype,
		Options:     mountOptions,
	})
	return nil
}

// GetAll returns all registered mount points
func (p *Points) GetAll() []specs.Mount {
	return p.points
}

// GetByDest returns registered mount points with the matched destination
func (p *Points) GetByDest(dest string) []specs.Mount {
	mounts := []specs.Mount{}
	for _, point := range p.points {
		if point.Destination == dest {
			mounts = append(mounts, point)
			break
		}
	}
	return mounts
}

// GetBySource returns registered mount points with the matched source
func (p *Points) GetBySource(source string) []specs.Mount {
	mounts := []specs.Mount{}
	for _, point := range p.points {
		if point.Source == source {
			mounts = append(mounts, point)
			break
		}
	}
	return mounts
}

// RemoveByDest removes mount points identified by destination
func (p *Points) RemoveByDest(dest string) {
	for i := len(p.points) - 1; i >= 0; i-- {
		if p.points[i].Destination == dest {
			p.points = append(p.points[:i], p.points[i+1:]...)
		}
	}
}

// RemoveBySource removes mount points identified by source
func (p *Points) RemoveBySource(source string) {
	for i := len(p.points) - 1; i >= 0; i-- {
		if p.points[i].Source == source {
			p.points = append(p.points[:i], p.points[i+1:]...)
		}
	}
}

// RemoveAll removes all mounts points from list
func (p *Points) RemoveAll() {
	p.points = nil
}

// Import imports a mount point list
func (p *Points) Import(points []specs.Mount) error {
	for _, point := range points {
		var err error
		var offset uint64
		var sizelimit uint64

		flags, options := ConvertOptions(point.Options)
		// check if this is a mount point to remount
		if flags&syscall.MS_REMOUNT != 0 {
			if err = p.AddRemount(point.Destination, flags); err == nil {
				continue
			}
		}
		// check if this is a bind mount point
		if flags&syscall.MS_BIND != 0 {
			if err = p.AddBind(point.Source, point.Destination, flags); err == nil {
				continue
			}
		}
		// check if this is an image mount point
		for _, option := range options {
			if strings.HasPrefix(option, "offset=") {
				fmt.Sscanf(option, "offset=%d", &offset)
			}
			if strings.HasPrefix(option, "sizelimit=") {
				fmt.Sscanf(option, "sizelimit=%d", &sizelimit)
			}
		}
		if err = p.AddImage(point.Source, point.Destination, point.Type, flags, offset, sizelimit); err == nil {
			continue
		}
		// check if this is a filesystem or overlay mount point
		if point.Type != "overlay" {
			if err = p.AddFS(point.Destination, point.Type, flags, strings.Join(options, ",")); err == nil {
				continue
			}
		} else {
			lowerdir := ""
			upperdir := ""
			workdir := ""
			for _, option := range options {
				if strings.HasPrefix(option, "lowerdir=") {
					fmt.Sscanf(option, "lowerdir=%s", &lowerdir)
				} else if strings.HasPrefix(option, "upperdir=") {
					fmt.Sscanf(option, "upperdir=%s", &upperdir)
				} else if strings.HasPrefix(option, "workdir=") {
					fmt.Sscanf(option, "workdir=%s", &workdir)
				}
			}
			if err = p.AddOverlay(point.Destination, flags, lowerdir, upperdir, workdir); err == nil {
				continue
			}
		}
		p.RemoveAll()
		return fmt.Errorf("can't import mount points list: %s", err)
	}
	return nil
}

// AddImage adds an image mount point
func (p *Points) AddImage(source string, dest string, fstype string, flags uintptr, offset uint64, sizelimit uint64) error {
	if source == "" {
		return fmt.Errorf("an image mount point must contain a source")
	}
	if !strings.HasPrefix(source, "/") {
		return fmt.Errorf("source must be an absolute path")
	}
	if flags&(syscall.MS_BIND|syscall.MS_REMOUNT|syscall.MS_REC) != 0 {
		return fmt.Errorf("MS_BIND, MS_REC or MS_REMOUNT are not valid flags for image mount points")
	}
	if _, ok := authorizedImage[fstype]; !ok {
		return fmt.Errorf("mount %s image is not authorized", fstype)
	}
	if sizelimit == 0 {
		return fmt.Errorf("invalid image size, zero length")
	}
	options := fmt.Sprintf("loop,offset=%d,sizelimit=%d,errors=remount-ro", offset, sizelimit)
	return p.add(source, dest, fstype, flags, options)
}

// GetAllImages returns a list of all registered image mount points
func (p *Points) GetAllImages() []specs.Mount {
	images := []specs.Mount{}
	for _, point := range p.points {
		for fs := range authorizedImage {
			if fs == point.Type {
				images = append(images, point)
				break
			}
		}
	}
	return images
}

// AddBind adds a bind mount point
func (p *Points) AddBind(source string, dest string, flags uintptr) error {
	bindFlags := flags | syscall.MS_BIND

	if source == "" {
		return fmt.Errorf("a bind mount point must contain a source")
	}
	if !strings.HasPrefix(source, "/") {
		return fmt.Errorf("source must be an absolute path")
	}
	if err := p.add(source, dest, "", bindFlags, ""); err != nil {
		return err
	}
	return nil
}

// GetAllBinds returns a list of all registered bind mount points
func (p *Points) GetAllBinds() []specs.Mount {
	binds := []specs.Mount{}
	for _, point := range p.points {
		for _, option := range point.Options {
			if option == "bind" || option == "rbind" {
				binds = append(binds, point)
				break
			}
		}
	}
	return binds
}

// AddOverlay adds an overlay mount point
func (p *Points) AddOverlay(dest string, flags uintptr, lowerdir string, upperdir string, workdir string) error {
	if flags&(syscall.MS_BIND|syscall.MS_REMOUNT|syscall.MS_REC) != 0 {
		return fmt.Errorf("MS_BIND, MS_REC or MS_REMOUNT are not valid flags for overlay mount points")
	}
	if lowerdir == "" {
		return fmt.Errorf("overlay mount point %s should have at least lowerdir option", dest)
	}
	if !strings.HasPrefix(lowerdir, "/") {
		return fmt.Errorf("lowerdir may contain only an absolute paths")
	}
	options := ""
	if upperdir != "" {
		if !strings.HasPrefix(upperdir, "/") {
			return fmt.Errorf("upperdir must be an absolute path")
		}
		if workdir == "" {
			return fmt.Errorf("overlay mount point %s should have workdir option set when used in conjunction with upperdir", dest)
		}
		if !strings.HasPrefix(workdir, "/") {
			return fmt.Errorf("workdir must be an absolute path")
		}
		options = fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lowerdir, upperdir, workdir)
	} else {
		options = fmt.Sprintf("lowerdir=%s", lowerdir)
	}
	return p.add("overlay", dest, "overlay", flags, options)
}

// GetAllOverlays returns a list of all registered overlay mount points
func (p *Points) GetAllOverlays() []specs.Mount {
	fs := []specs.Mount{}
	for _, point := range p.points {
		if point.Type == "overlay" {
			fs = append(fs, point)
		}
	}
	return fs
}

// AddFS adds a filesystem mount point
func (p *Points) AddFS(dest string, fstype string, flags uintptr, options string) error {
	if flags&(syscall.MS_BIND|syscall.MS_REMOUNT|syscall.MS_REC) != 0 {
		return fmt.Errorf("MS_BIND, MS_REC or MS_REMOUNT are not valid flags for FS mount points")
	}
	if _, ok := authorizedFS[fstype]; !ok {
		return fmt.Errorf("mount %s file system is not authorized", fstype)
	}
	return p.add(fstype, dest, fstype, flags, options)
}

// GetAllFS returns a list of all registered filesystem mount points
func (p *Points) GetAllFS() []specs.Mount {
	fs := []specs.Mount{}
	for _, point := range p.points {
		for fstype := range authorizedFS {
			if fstype == point.Type && point.Type != "overlay" {
				fs = append(fs, point)
			}
		}
	}
	return fs
}

// AddRemount adds a mount point to remount
func (p *Points) AddRemount(dest string, flags uintptr) error {
	remountFlags := flags | syscall.MS_REMOUNT
	return p.add("", dest, "", remountFlags, "")
}
