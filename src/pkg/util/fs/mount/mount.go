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

var internalOptions = []string{"loop", "offset", "sizelimit"}

type fsContext struct {
	context bool
}

const (
	// SessionTag defines tag for session directory
	SessionTag = "sessiondir"
	// RootfsTag defines tag for container root filesystem
	RootfsTag = "rootfs"
	// OverlayLowerDirTag defines tag for overlay lower directories
	OverlayLowerDirTag = "overlay-lowerdir"
	// OverlayTag defines tag for overlay mount point
	OverlayTag = "overlay"
	// UnderlayTag defines tag for underlay mount points
	UnderlayTag = "underlay"
	// HostfsTag defines tag for host filesystem mount point
	HostfsTag = "hostfs"
	// BindsTag defines tag for bind path
	BindsTag = "binds"
	// KernelTag defines tag for kernel related mount point (proc, sysfs)
	KernelTag = "kernel"
	// DevTag defines tag for dev related mount point
	DevTag = "dev"
	// HomeTag defines tag for home directory mount point
	HomeTag = "home"
	// UserbindsTag defines tag for user bind mount points
	UserbindsTag = "userbinds"
	// TmpTag defines tag for temporary filesystem mount points (/tmp, /var/tmp)
	TmpTag = "tmp"
	// ScratchTag defines tag for scratch mount points
	ScratchTag = "scratch"
	// CwdTag defines tag for current working directory mount point
	CwdTag = "cwd"
	// FilesTag defines tag for file mount points (passwd, group ...)
	FilesTag = "files"
	// CustomTag defines tag for custom mount points
	CustomTag = "custom"
)

var authorizedTags = []struct {
	name      string
	multiDest bool
}{
	{SessionTag, false},
	{RootfsTag, false},
	{OverlayLowerDirTag, true},
	{OverlayTag, false},
	{UnderlayTag, true},
	{HostfsTag, true},
	{BindsTag, true},
	{KernelTag, true},
	{DevTag, true},
	{HomeTag, false},
	{UserbindsTag, true},
	{TmpTag, true},
	{ScratchTag, true},
	{CwdTag, false},
	{FilesTag, true},
	{CustomTag, true},
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
// plus internal options
func ConvertOptions(options []string) (uintptr, []string, []string) {
	var flags uintptr
	finalOpt := []string{}
	internalOpt := []string{}
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
			isInternal := false
			for _, opt := range internalOptions {
				if strings.HasPrefix(optionTrim, opt+"=") {
					internalOpt = append(internalOpt, optionTrim)
					isInternal = true
					break
				}
			}
			if !isInternal {
				finalOpt = append(finalOpt, optionTrim)
			}
		}
		isFlag = false
	}
	return flags, finalOpt, internalOpt
}

// GetOffset return offset value for image options
func GetOffset(options []string) (uint64, error) {
	var offset uint64
	for _, opt := range options {
		if strings.HasPrefix(opt, "offset=") {
			fmt.Sscanf(opt, "offset=%d", &offset)
			return offset, nil
		}
	}
	return 0, fmt.Errorf("offset option not found")
}

// GetSizeLimit returns sizelimit value for image options
func GetSizeLimit(options []string) (uint64, error) {
	var size uint64
	for _, opt := range options {
		if strings.HasPrefix(opt, "sizelimit=") {
			fmt.Sscanf(opt, "sizelimit=%d", &size)
			return size, nil
		}
	}
	return 0, fmt.Errorf("sizelimit option not found")
}

// Points defines and stores a set of mount points
type Points struct {
	context string
	tags    map[string][]string
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
	if p.hasDest(dest) && (flags&syscall.MS_REMOUNT) == 0 {
		return fmt.Errorf("destination %s is already in the mount point list", dest)
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
		context := fmt.Sprintf("context=%s", p.context)
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

func (p *Points) hasDest(dest string) bool {
	for _, point := range p.points {
		if point.Destination == dest {
			return true
		}
	}
	return false
}

// GetTags returns all registered tags and associated mount point
func (p *Points) GetTags() map[string][]string {
	return p.tags
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

// Tag sets a tag on destination mount point
func (p *Points) Tag(dest string, tag string) error {
	if !p.hasDest(dest) {
		return fmt.Errorf("no destination %s found", dest)
	}
	for _, t := range authorizedTags {
		for _, d := range p.tags[t.name] {
			if d == dest {
				return fmt.Errorf("tag %s already contains destination %s", t.name, dest)
			}
		}
		if tag == t.name {
			if !t.multiDest && len(p.tags[tag]) == 1 {
				return fmt.Errorf("tag %s can't have more than one destination", tag)
			}
			p.tags[tag] = append(p.tags[tag], dest)
			return nil
		}
	}
	return fmt.Errorf("tag %s is not a recognized tag", tag)
}

// GetByTag returns mount points attached to a tag
func (p *Points) GetByTag(tag string) []specs.Mount {
	mounts := []specs.Mount{}
	for _, dest := range p.tags[tag] {
		mounts = append(mounts, p.GetByDest(dest)...)
	}
	return mounts
}

// RemoveByTag removes mount points attached to a tag
func (p *Points) RemoveByTag(tag string) {
	for _, dest := range p.tags[tag] {
		p.RemoveByDest(dest)
	}
}

// RemoveByDest removes mount points identified by destination
func (p *Points) RemoveByDest(dest string) {
	for i := len(p.points) - 1; i >= 0; i-- {
		if p.points[i].Destination == dest {
			for _, t := range authorizedTags {
				for d := len(p.tags[t.name]) - 1; d >= 0; d-- {
					if p.tags[t.name][d] == dest {
						p.tags[t.name] = append(p.tags[t.name][:d], p.tags[t.name][d+1:]...)
					}
				}
			}
			p.points = append(p.points[:i], p.points[i+1:]...)
		}
	}
}

// RemoveBySource removes mount points identified by source
func (p *Points) RemoveBySource(source string) {
	for i := len(p.points) - 1; i >= 0; i-- {
		if p.points[i].Source == source {
			for _, t := range authorizedTags {
				for d := len(p.tags[t.name]) - 1; d >= 0; d-- {
					if p.tags[t.name][d] == p.points[i].Destination {
						p.tags[t.name] = append(p.tags[t.name][:d], p.tags[t.name][d+1:]...)
					}
				}
			}
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

		flags, options, internal := ConvertOptions(point.Options)
		// check if this is a mount point to remount
		if flags&syscall.MS_REMOUNT != 0 {
			if err = p.AddRemount(point.Destination, flags); err == nil {
				continue
			}
		}
		for _, option := range internal {
			if strings.HasPrefix(option, "offset=") {
				fmt.Sscanf(option, "offset=%d", &offset)
			}
			if strings.HasPrefix(option, "sizelimit=") {
				fmt.Sscanf(option, "sizelimit=%d", &sizelimit)
			}
		}
		// check if this is a bind mount point
		if flags&syscall.MS_BIND != 0 {
			if err = p.AddBind(point.Source, point.Destination, flags); err == nil {
				continue
			}
		}
		// check if this is an image mount point
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

// ImportTags imports a tag list with associated destination mount point
func (p *Points) ImportTags(tags map[string][]string) error {
	for tag, dests := range tags {
		for _, dest := range dests {
			if err := p.Tag(dest, tag); err != nil {
				return err
			}
		}
	}
	return nil
}

// AddImage adds an image mount point
func (p *Points) AddImage(source string, dest string, fstype string, flags uintptr, offset uint64, sizelimit uint64) error {
	options := ""
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
	options = fmt.Sprintf("loop,offset=%d,sizelimit=%d,errors=remount-ro", offset, sizelimit)
	return p.add(source, dest, fstype, flags, options)
}

// GetAllImages returns a list of all registered image mount points
func (p *Points) GetAllImages() []specs.Mount {
	images := []specs.Mount{}
	for _, point := range p.points {
		if _, ok := authorizedImage[point.Type]; ok {
			images = append(images, point)
		}
	}
	return images
}

// AddBind adds a bind mount point
func (p *Points) AddBind(source string, dest string, flags uintptr) error {
	bindFlags := flags | syscall.MS_BIND
	options := ""

	if source == "" {
		return fmt.Errorf("a bind mount point must contain a source")
	}
	if !strings.HasPrefix(source, "/") {
		return fmt.Errorf("source must be an absolute path")
	}
	if err := p.add(source, dest, "", bindFlags, options); err != nil {
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

// SetContext sets SELinux mount context, once set it can't be modified
func (p *Points) SetContext(context string) error {
	if p.context == "" {
		p.context = context
		return nil
	}
	return fmt.Errorf("mount context has already been set")
}

// GetContext returns SELinux mount context
func (p *Points) GetContext() string {
	return p.context
}
