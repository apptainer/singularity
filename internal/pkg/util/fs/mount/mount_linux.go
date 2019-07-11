// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package mount

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/sylabs/singularity/pkg/util/fs/proc"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

type mountError string

func (e mountError) Error() string { return string(e) }

const (
	// ErrMountExists indicates a duplicated mount being asked for
	ErrMountExists = mountError("destination is already in the mount point list")
)

var mountFlags = []struct {
	option string
	flag   uintptr
}{
	{"acl", 0},
	{"async", syscall.MS_ASYNC},
	{"atime", 0},
	{"bind", syscall.MS_BIND},
	{"defaults", 0},
	{"dev", 0},
	{"diratime", 0},
	{"dirsync", syscall.MS_DIRSYNC},
	{"exec", 0},
	{"iversion", 0},
	{"lazytime", 0},
	{"loud", 0},
	{"mand", syscall.MS_MANDLOCK},
	{"noacl", 0},
	{"noatime", syscall.MS_NOATIME},
	{"nodev", syscall.MS_NODEV},
	{"nodiratime", syscall.MS_NODIRATIME},
	{"noexec", syscall.MS_NOEXEC},
	{"noiversion", 0},
	{"nolazytime", 0},
	{"nomand", 0},
	{"norelatime", 0},
	{"nostrictatime", 0},
	{"nosuid", syscall.MS_NOSUID},
	{"private", syscall.MS_PRIVATE},
	{"rbind", syscall.MS_BIND | syscall.MS_REC},
	{"rprivate", syscall.MS_PRIVATE | syscall.MS_REC},
	{"rslave", syscall.MS_SLAVE | syscall.MS_REC},
	{"rshared", syscall.MS_SHARED | syscall.MS_REC},
	{"runbindable", syscall.MS_UNBINDABLE | syscall.MS_REC},
	{"relatime", syscall.MS_RELATIME},
	{"remount", syscall.MS_REMOUNT},
	{"ro", syscall.MS_RDONLY},
	{"rw", 0},
	{"shared", syscall.MS_SHARED},
	{"slave", syscall.MS_SLAVE},
	{"silent", syscall.MS_SILENT},
	{"strictatime", syscall.MS_STRICTATIME},
	{"suid", 0},
	{"sync", syscall.MS_SYNCHRONOUS},
	{"unbindable", syscall.MS_UNBINDABLE},
}

type fsContext struct {
	context bool
}

// AuthorizedTag defines the tag type
type AuthorizedTag string

const (
	// SessionTag defines tag for session directory
	SessionTag AuthorizedTag = "sessiondir"
	// RootfsTag defines tag for container root filesystem
	RootfsTag = "rootfs"
	// PreLayerTag defines tag to prepare overlay/underlay layer
	PreLayerTag = "prelayer"
	// LayerTag defines tag for overlay/underlay final mount point
	LayerTag = "layer"
	// SharedTag defines tag for shared mount point between master
	// and container processes
	SharedTag = "shared"
	// DevTag defines tag for dev related mount point
	DevTag = "dev"
	// HostfsTag defines tag for host filesystem mount point
	HostfsTag = "hostfs"
	// BindsTag defines tag for bind path
	BindsTag = "binds"
	// KernelTag defines tag for kernel related mount point (proc, sysfs)
	KernelTag = "kernel"
	// HomeTag defines tag for home directory mount point
	HomeTag = "home"
	// TmpTag defines tag for temporary filesystem mount points (/tmp, /var/tmp)
	TmpTag = "tmp"
	// ScratchTag defines tag for scratch mount points
	ScratchTag = "scratch"
	// CwdTag defines tag for current working directory mount point
	CwdTag = "cwd"
	// FilesTag defines tag for file mount points (passwd, group ...)
	FilesTag = "files"
	// UserbindsTag defines tag for user bind mount points
	UserbindsTag = "userbinds"
	// OtherTag defines tag for other mount points that can't be classified
	OtherTag = "other"
	// FinalTag defines tag for mount points to mount/remount at the end of mount process
	FinalTag = "final"
)

var authorizedTags = map[AuthorizedTag]struct {
	multiPoint bool
	order      int
}{
	SessionTag:   {false, 0},
	RootfsTag:    {false, 1},
	PreLayerTag:  {true, 2},
	LayerTag:     {false, 3},
	SharedTag:    {true, 4},
	DevTag:       {true, 5},
	HostfsTag:    {true, 6},
	BindsTag:     {true, 7},
	KernelTag:    {true, 8},
	HomeTag:      {false, 9},
	TmpTag:       {true, 10},
	ScratchTag:   {true, 11},
	CwdTag:       {false, 12},
	FilesTag:     {true, 13},
	UserbindsTag: {true, 14},
	OtherTag:     {true, 15},
	FinalTag:     {true, 16},
}

var authorizedImage = map[string]fsContext{
	"encryptfs": {true},
	"ext3":      {true},
	"squashfs":  {true},
}

var authorizedFS = map[string]fsContext{
	"overlay": {true},
	"tmpfs":   {true},
	"ramfs":   {true},
	"devpts":  {true},
	"sysfs":   {false},
	"proc":    {false},
	"mqueue":  {false},
	"cgroup":  {false},
}

var internalOptions = []string{"loop", "offset", "sizelimit", "key"}

// Point describes a mount point
type Point struct {
	specs.Mount
	InternalOptions []string `json:"internalOptions"`
}

// Points defines and stores a set of mount points by tag
type Points struct {
	context string
	points  map[AuthorizedTag][]Point
}

// ConvertOptions converts an options string into a pair of mount flags and mount options
func ConvertOptions(options []string) (uintptr, []string) {
	var flags uintptr
	finalOpt := []string{}
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
			finalOpt = append(finalOpt, optionTrim)
		}
		isFlag = false
	}
	return flags, finalOpt
}

// ConvertSpec converts an OCI Mount spec into an importable mount points list
func ConvertSpec(mounts []specs.Mount) (map[AuthorizedTag][]Point, error) {
	points := make(map[AuthorizedTag][]Point)
	for _, m := range mounts {
		var tag AuthorizedTag
		var options []string
		var propagationOption string
		var err error
		source := m.Source
		mountType := m.Type

		if mountType != "" && mountType != "bind" && mountType != "none" {
			if _, ok := authorizedFS[mountType]; !ok {
				return points, fmt.Errorf("%s filesystem type is not authorized", mountType)
			}
			if has, err := proc.HasFilesystem(mountType); err != nil || !has {
				return points, fmt.Errorf("%s filesystem not supported", mountType)
			}
			tag = KernelTag
		} else {
			source, err = filepath.Abs(m.Source)
			if err != nil {
				return points, fmt.Errorf("failed to determine absolute path for %s: %s", m.Source, err)
			}
			tag = UserbindsTag
			mountType = ""
		}

		for _, opt := range m.Options {
			switch opt {
			case "shared",
				"rshared",
				"slave",
				"rslave",
				"private",
				"rprivate",
				"unbindable",
				"runbindable":
				propagationOption = opt
			default:
				options = append(options, opt)
			}
		}

		points[tag] = append(points[tag], Point{
			Mount: specs.Mount{
				Source:      source,
				Destination: m.Destination,
				Type:        mountType,
				Options:     options,
			},
		})

		if len(options) > 1 && tag == UserbindsTag {
			options = append(options, "remount")
			points[tag] = append(points[tag], Point{
				Mount: specs.Mount{
					Source:      "",
					Destination: m.Destination,
					Type:        "",
					Options:     options,
				},
			})
		}
		if propagationOption != "" {
			points[tag] = append(points[tag], Point{
				Mount: specs.Mount{
					Source:      "",
					Destination: m.Destination,
					Type:        "",
					Options:     []string{propagationOption},
				},
			})
		}
	}
	return points, nil
}

// GetTagList returns authorized tags in right order
func GetTagList() []AuthorizedTag {
	tagList := make([]AuthorizedTag, len(authorizedTags))
	for k, tag := range authorizedTags {
		tagList[tag.order] = k
	}
	return tagList
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

// GetKey returns key value for image options
func GetKey(options []string) (string, error) {
	for _, opt := range options {
		if strings.HasPrefix(opt, "key=") {
			return strings.TrimPrefix(opt, "key="), nil
		}
	}
	return "", fmt.Errorf("key option not found")
}

// HasRemountFlag checks if remount flag is set or not.
func HasRemountFlag(flags uintptr) bool {
	return flags&syscall.MS_REMOUNT != 0
}

// HasPropagationFlag checks if a propagation flag is set or not.
func HasPropagationFlag(flags uintptr) bool {
	return flags&getPropagationFlags() != 0
}

func getPropagationFlags() uintptr {
	return syscall.MS_UNBINDABLE | syscall.MS_SHARED | syscall.MS_PRIVATE | syscall.MS_SLAVE
}

func (p *Points) init() {
	if p.points == nil {
		p.points = make(map[AuthorizedTag][]Point)
	}
}

func (p *Points) add(tag AuthorizedTag, source string, dest string, fstype string, flags uintptr, options string) error {
	var bind = false

	p.init()

	mountOpts := []string{}
	internalOpts := []string{}

	if dest == "" {
		return fmt.Errorf("mount point must contain a destination")
	}
	if !strings.HasPrefix(dest, "/") {
		return fmt.Errorf("destination must be an absolute path")
	}
	if _, ok := authorizedTags[tag]; !ok {
		return fmt.Errorf("tag %s is not a recognized tag", tag)
	}
	if !HasRemountFlag(flags) && !HasPropagationFlag(flags) {
		present := false
		for _, point := range p.points[tag] {
			if point.Destination == dest {
				present = true
				break
			}
		}
		if present {
			return ErrMountExists
		}

		if len(p.points[tag]) == 1 && !authorizedTags[tag].multiPoint {
			return fmt.Errorf("tag %s allow only one mount point", tag)
		}
	}
	for i := len(mountFlags) - 1; i >= 0; i-- {
		flag := mountFlags[i].flag
		if flag != 0 && flag == (flags&flag) {
			if bind && flag&syscall.MS_BIND != 0 {
				continue
			}
			mountOpts = append(mountOpts, mountFlags[i].option)
			if flag&syscall.MS_BIND != 0 {
				bind = true
			}
		}
	}
	setContext := true
	for _, option := range strings.Split(options, ",") {
		o := strings.TrimSpace(option)
		if o != "" {
			keyVal := strings.SplitN(o, "=", 2)
			if keyVal[0] == "context" {
				setContext = false
			}
			isInternal := false
			for _, internal := range internalOptions {
				if keyVal[0] == internal {
					isInternal = true
				}
			}
			if isInternal {
				internalOpts = append(internalOpts, o)
			} else {
				mountOpts = append(mountOpts, o)
			}
		}
	}
	if fstype != "" && setContext {
		setContext = authorizedFS[fstype].context
	}
	if !bind && setContext && p.context != "" {
		context := fmt.Sprintf("context=%q", p.context)
		mountOpts = append(mountOpts, context)
	}
	p.points[tag] = append(p.points[tag], Point{
		Mount: specs.Mount{
			Source:      source,
			Destination: dest,
			Type:        fstype,
			Options:     mountOpts,
		},
		InternalOptions: internalOpts,
	})
	return nil
}

// GetAll returns all registered mount points
func (p *Points) GetAll() map[AuthorizedTag][]Point {
	p.init()
	return p.points
}

// GetByDest returns registered mount points with the matched destination
func (p *Points) GetByDest(dest string) []Point {
	p.init()
	mounts := []Point{}
	for tag := range p.points {
		for _, point := range p.points[tag] {
			if point.Destination == dest {
				mounts = append(mounts, point)
			}
		}
	}
	return mounts
}

// GetBySource returns registered mount points with the matched source
func (p *Points) GetBySource(source string) []Point {
	p.init()
	mounts := []Point{}
	for tag := range p.points {
		for _, point := range p.points[tag] {
			if point.Source == source {
				mounts = append(mounts, point)
			}
		}
	}
	return mounts
}

// GetByTag returns mount points attached to a tag
func (p *Points) GetByTag(tag AuthorizedTag) []Point {
	p.init()
	return p.points[tag]
}

// RemoveAll removes all mounts points from list
func (p *Points) RemoveAll() {
	p.init()
	for tag := range p.points {
		p.points[tag] = nil
	}
}

// RemoveByDest removes mount points identified by destination
func (p *Points) RemoveByDest(dest string) {
	p.init()
	for tag := range p.points {
		for d := len(p.points[tag]) - 1; d >= 0; d-- {
			if p.points[tag][d].Destination == dest {
				p.points[tag] = append(p.points[tag][:d], p.points[tag][d+1:]...)
			}
		}
	}
}

// RemoveBySource removes mount points identified by source
func (p *Points) RemoveBySource(source string) {
	p.init()
	for tag := range p.points {
		for d := len(p.points[tag]) - 1; d >= 0; d-- {
			if p.points[tag][d].Source == source {
				p.points[tag] = append(p.points[tag][:d], p.points[tag][d+1:]...)
			}
		}
	}
}

// RemoveByTag removes mount points attached to a tag
func (p *Points) RemoveByTag(tag AuthorizedTag) {
	p.init()
	p.points[tag] = nil
}

// Import imports a mount point list
func (p *Points) Import(points map[AuthorizedTag][]Point) error {
	for tag := range points {
		for _, point := range points[tag] {
			var err error
			var offset uint64
			var sizelimit uint64
			var key string

			flags, options := ConvertOptions(point.Options)
			// check if this is a mount point to remount
			if HasRemountFlag(flags) {
				if err = p.AddRemount(tag, point.Destination, flags); err == nil {
					continue
				}
			}
			if HasPropagationFlag(flags) {
				if err = p.AddPropagation(tag, point.Destination, flags); err == nil {
					continue
				}
			}
			// check if this is a bind mount point
			if flags&syscall.MS_BIND != 0 {
				if err = p.AddBind(tag, point.Source, point.Destination, flags); err == nil {
					continue
				} else {
					return err
				}
			}

			for _, option := range point.InternalOptions {
				if strings.HasPrefix(option, "offset=") {
					fmt.Sscanf(option, "offset=%d", &offset)
				}
				if strings.HasPrefix(option, "sizelimit=") {
					fmt.Sscanf(option, "sizelimit=%d", &sizelimit)
				}
				if strings.HasPrefix(option, "key=") {
					key = strings.TrimPrefix(option, "key=")
				}
			}

			// check if this is an image mount point
			if err = p.AddImage(tag, point.Source, point.Destination, point.Type, flags, offset, sizelimit, key); err == nil {
				continue
			}

			// check if this is a filesystem or overlay mount point
			if point.Type != "overlay" {
				if err = p.AddFSWithSource(tag, point.Source, point.Destination, point.Type, flags, strings.Join(options, ",")); err == nil {
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
				if err = p.AddOverlay(tag, point.Destination, flags, lowerdir, upperdir, workdir); err == nil {
					continue
				}
			}
			p.RemoveAll()
			return fmt.Errorf("can't import mount points list: %s", err)
		}
	}
	return nil
}

// ImportFromSpec converts an OCI Mount spec into a mount point list
// and imports it
func (p *Points) ImportFromSpec(mounts []specs.Mount) error {
	points, err := ConvertSpec(mounts)
	if err != nil {
		return err
	}
	return p.Import(points)
}

// AddImage adds an image mount point
func (p *Points) AddImage(tag AuthorizedTag, source string, dest string, fstype string, flags uintptr, offset uint64, sizelimit uint64, key string) error {
	options := ""
	if source == "" {
		return fmt.Errorf("an image mount point must contain a source")
	}
	if !strings.HasPrefix(source, "/") {
		return fmt.Errorf("source must be an absolute path")
	}
	if flags&(syscall.MS_BIND|syscall.MS_REMOUNT|syscall.MS_REC) != 0 {
		return fmt.Errorf("ms_bind, ms_rec or ms_remount are not valid flags for image mount points")
	}
	if _, ok := authorizedImage[fstype]; !ok {
		return fmt.Errorf("mount %s image is not authorized", fstype)
	}
	if sizelimit == 0 {
		return fmt.Errorf("invalid image size, zero length")
	}
	options = fmt.Sprintf("loop,offset=%d,sizelimit=%d,key=%s,errors=remount-ro", offset, sizelimit, key)
	return p.add(tag, source, dest, fstype, flags, options)
}

// GetAllImages returns a list of all registered image mount points
func (p *Points) GetAllImages() []Point {
	p.init()
	images := []Point{}
	for tag := range p.points {
		for _, point := range p.points[tag] {
			if _, ok := authorizedImage[point.Type]; ok {
				images = append(images, point)
			}
		}
	}
	return images
}

// AddBind adds a bind mount point
func (p *Points) AddBind(tag AuthorizedTag, source string, dest string, flags uintptr) error {
	bindFlags := flags | syscall.MS_BIND
	options := ""

	if source == "" {
		return fmt.Errorf("a bind mount point must contain a source")
	}
	if !strings.HasPrefix(source, "/") {
		return fmt.Errorf("source must be an absolute path")
	}
	return p.add(tag, source, dest, "", bindFlags, options)
}

// GetAllBinds returns a list of all registered bind mount points
func (p *Points) GetAllBinds() []Point {
	p.init()
	binds := []Point{}
	for tag := range p.points {
		for _, point := range p.points[tag] {
			for _, option := range point.Options {
				if option == "bind" || option == "rbind" {
					binds = append(binds, point)
					break
				}
			}
		}
	}
	return binds
}

// AddOverlay adds an overlay mount point
func (p *Points) AddOverlay(tag AuthorizedTag, dest string, flags uintptr, lowerdir string, upperdir string, workdir string) error {
	if flags&(syscall.MS_BIND|syscall.MS_REMOUNT|syscall.MS_REC) != 0 {
		return fmt.Errorf("ms_bind, ms_rec or ms_remount are not valid flags for overlay mount points")
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
	return p.add(tag, "overlay", dest, "overlay", flags, options)
}

// GetAllOverlays returns a list of all registered overlay mount points
func (p *Points) GetAllOverlays() []Point {
	p.init()
	fs := []Point{}
	for tag := range p.points {
		for _, point := range p.points[tag] {
			if point.Type == "overlay" {
				fs = append(fs, point)
			}
		}
	}
	return fs
}

// AddFS adds a filesystem mount point
func (p *Points) AddFS(tag AuthorizedTag, dest string, fstype string, flags uintptr, options string) error {
	return p.AddFSWithSource(tag, fstype, dest, fstype, flags, options)
}

// AddFSWithSource adds a filesystem mount point
func (p *Points) AddFSWithSource(tag AuthorizedTag, source string, dest string, fstype string, flags uintptr, options string) error {
	if flags&(syscall.MS_BIND|syscall.MS_REMOUNT|syscall.MS_REC) != 0 {
		return fmt.Errorf("ms_bind, ms_rec or ms_remount are not valid flags for fs mount points")
	}
	if _, ok := authorizedFS[fstype]; !ok {
		return fmt.Errorf("mount %s file system is not authorized", fstype)
	}
	return p.add(tag, source, dest, fstype, flags, options)
}

// GetAllFS returns a list of all registered filesystem mount points
func (p *Points) GetAllFS() []Point {
	p.init()
	fs := []Point{}
	for tag := range p.points {
		for _, point := range p.points[tag] {
			for fstype := range authorizedFS {
				if fstype == point.Type && point.Type != "overlay" {
					fs = append(fs, point)
				}
			}
		}
	}
	return fs
}

// AddRemount adds a mount point to remount
func (p *Points) AddRemount(tag AuthorizedTag, dest string, flags uintptr) error {
	remountFlags := (flags &^ getPropagationFlags()) | syscall.MS_REMOUNT
	return p.add(tag, "", dest, "", remountFlags, "")
}

// AddPropagation adds a mount propagation for mount point
func (p *Points) AddPropagation(tag AuthorizedTag, dest string, flags uintptr) error {
	finalFlags := flags & getPropagationFlags()
	if !HasPropagationFlag(finalFlags) {
		return fmt.Errorf("no mount propagation flag found")
	}
	if flags&syscall.MS_REC != 0 {
		finalFlags |= syscall.MS_REC
	}
	return p.add(tag, "", dest, "", finalFlags, "")
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
