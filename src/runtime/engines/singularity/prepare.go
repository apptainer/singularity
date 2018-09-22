// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/sylabs/singularity/src/pkg/buildcfg"
	"github.com/sylabs/singularity/src/pkg/image"
	"github.com/sylabs/singularity/src/pkg/instance"
	"github.com/sylabs/singularity/src/pkg/syecl"
	"github.com/sylabs/singularity/src/pkg/sylog"
	"github.com/sylabs/singularity/src/pkg/util/capabilities"
	"github.com/sylabs/singularity/src/pkg/util/fs"
	"github.com/sylabs/singularity/src/pkg/util/mainthread"
	"github.com/sylabs/singularity/src/pkg/util/user"
	"github.com/sylabs/singularity/src/runtime/engines/config"
	"github.com/sylabs/singularity/src/runtime/engines/config/starter"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// prepareUserCaps is responsible for checking that user's requested
// capabilities are authorized
func (e *EngineOperations) prepareUserCaps() error {
	uid := os.Getuid()
	commonCaps := make([]string, 0)

	e.EngineConfig.OciConfig.SetProcessNoNewPrivileges(true)

	file, err := capabilities.Open(buildcfg.CAPABILITY_FILE, true)
	if err != nil {
		return err
	}
	defer file.Close()

	pw, err := user.GetPwUID(uint32(uid))
	if err != nil {
		return err
	}

	caps, _ := capabilities.Split(e.EngineConfig.GetAddCaps())
	caps = append(caps, e.EngineConfig.OciConfig.Process.Capabilities.Permitted...)

	authorizedCaps, _ := file.CheckUserCaps(pw.Name, caps)

	if len(authorizedCaps) > 0 {
		sylog.Debugf("User capabilities %v added", authorizedCaps)
		commonCaps = authorizedCaps
	}

	groups, err := os.Getgroups()
	for _, g := range groups {
		gr, err := user.GetGrGID(uint32(g))
		if err != nil {
			return err
		}
		authorizedCaps, _ := file.CheckGroupCaps(gr.Name, caps)
		if len(authorizedCaps) > 0 {
			sylog.Debugf("%s group capabilities %v added", gr.Name, authorizedCaps)
			commonCaps = append(commonCaps, authorizedCaps...)
		}
	}

	commonCaps = capabilities.RemoveDuplicated(commonCaps)

	caps, _ = capabilities.Split(e.EngineConfig.GetDropCaps())
	for _, cap := range caps {
		for i, c := range commonCaps {
			if c == cap {
				sylog.Debugf("Capability %s dropped", cap)
				commonCaps = append(commonCaps[:i], commonCaps[i+1:]...)
				break
			}
		}
	}

	e.EngineConfig.OciConfig.Process.Capabilities.Permitted = commonCaps
	e.EngineConfig.OciConfig.Process.Capabilities.Effective = commonCaps
	e.EngineConfig.OciConfig.Process.Capabilities.Inheritable = commonCaps
	e.EngineConfig.OciConfig.Process.Capabilities.Bounding = commonCaps
	e.EngineConfig.OciConfig.Process.Capabilities.Ambient = commonCaps

	return nil
}

// prepareRootCaps is responsible for setting root capabilities
// based on capability/configuration files and requested capabilities
func (e *EngineOperations) prepareRootCaps() error {
	commonCaps := make([]string, 0)
	defaultCapabilities := e.EngineConfig.File.RootDefaultCapabilities

	// is no-privs/keep-privs set on command line
	if e.EngineConfig.GetNoPrivs() {
		sylog.Debugf("--no-privs requested")
		defaultCapabilities = "no"
	} else if e.EngineConfig.GetKeepPrivs() {
		sylog.Debugf("--keep-privs requested")
		defaultCapabilities = "full"
	}

	sylog.Debugf("Root %s capabilities", defaultCapabilities)

	// set default capabilities based on configuration file directive
	switch defaultCapabilities {
	case "full":
		e.EngineConfig.OciConfig.SetupPrivileged(true)
		commonCaps = e.EngineConfig.OciConfig.Process.Capabilities.Permitted
	case "file":
		file, err := capabilities.Open(buildcfg.CAPABILITY_FILE, true)
		if err != nil {
			return err
		}
		defer file.Close()

		commonCaps = append(commonCaps, file.ListUserCaps("root")...)
		groups, err := os.Getgroups()
		for _, g := range groups {
			gr, err := user.GetGrGID(uint32(g))
			if err != nil {
				return err
			}
			caps := file.ListGroupCaps(gr.Name)
			commonCaps = append(commonCaps, caps...)
			sylog.Debugf("%s group capabilities %v added", gr.Name, caps)
		}
	}

	caps, _ := capabilities.Split(e.EngineConfig.GetAddCaps())
	for _, cap := range caps {
		found := false
		for _, c := range commonCaps {
			if c == cap {
				found = true
				break
			}
		}
		if !found {
			sylog.Debugf("Root capability %s added", cap)
			commonCaps = append(commonCaps, cap)
		}
	}

	commonCaps = capabilities.RemoveDuplicated(commonCaps)

	caps, _ = capabilities.Split(e.EngineConfig.GetDropCaps())
	for _, cap := range caps {
		for i, c := range commonCaps {
			if c == cap {
				sylog.Debugf("Root capability %s dropped", cap)
				commonCaps = append(commonCaps[:i], commonCaps[i+1:]...)
				break
			}
		}
	}

	e.EngineConfig.OciConfig.Process.Capabilities.Permitted = commonCaps
	e.EngineConfig.OciConfig.Process.Capabilities.Effective = commonCaps
	e.EngineConfig.OciConfig.Process.Capabilities.Inheritable = commonCaps
	e.EngineConfig.OciConfig.Process.Capabilities.Bounding = commonCaps
	e.EngineConfig.OciConfig.Process.Capabilities.Ambient = commonCaps

	return nil
}

func (e *EngineOperations) prepareFd() {
	fds := make([]int, 0)

	if e.EngineConfig.File.UserBindControl {
		for _, b := range e.EngineConfig.GetBindPath() {
			splitted := strings.Split(b, ":")

			src, err := filepath.Abs(splitted[0])
			if err != nil {
				continue
			}

			if !fs.IsDir(src) {
				continue
			}

			sylog.Debugf("Open file descriptor for %s", src)
			f, err := os.Open(src)
			if err != nil {
				continue
			}
			fds = append(fds, int(f.Fd()))
		}
	}

	if !e.EngineConfig.GetContain() {
		for _, bindpath := range e.EngineConfig.File.BindPath {
			splitted := strings.Split(bindpath, ":")
			src := splitted[0]

			if !fs.IsDir(src) {
				continue
			}

			sylog.Debugf("Open file descriptor for %s", src)
			f, err := os.Open(src)
			if err != nil {
				continue
			}
			fds = append(fds, int(f.Fd()))
		}
	}

	for _, path := range e.EngineConfig.File.AutofsBugPath {
		if !fs.IsDir(path) {
			continue
		}

		sylog.Debugf("Open file descriptor for %s", path)
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		fds = append(fds, int(f.Fd()))
	}

	e.EngineConfig.SetOpenFd(fds)
}

// prepareContainerConfig is responsible for getting and applying user supplied
// configuration for container creation
func (e *EngineOperations) prepareContainerConfig(starterConfig *starter.Config) error {
	// always set mount namespace
	e.EngineConfig.OciConfig.AddOrReplaceLinuxNamespace(specs.MountNamespace, "")

	// if PID namespace is not allowed remove it from namespaces
	if !e.EngineConfig.File.AllowPidNs && e.EngineConfig.OciConfig.Linux != nil {
		namespaces := e.EngineConfig.OciConfig.Linux.Namespaces
		for i, ns := range namespaces {
			if ns.Type == specs.PIDNamespace {
				sylog.Debugf("Not virtualizing PID namespace by configuration")
				e.EngineConfig.OciConfig.Linux.Namespaces = append(namespaces[:i], namespaces[i+1:]...)
				break
			}
		}
	}

	if os.Getuid() == 0 {
		if err := e.prepareRootCaps(); err != nil {
			return err
		}
	} else {
		if err := e.prepareUserCaps(); err != nil {
			return err
		}
	}

	if e.EngineConfig.File.MountSlave {
		starterConfig.SetMountPropagation("slave")
	} else {
		starterConfig.SetMountPropagation("private")
	}

	starterConfig.SetInstance(e.EngineConfig.GetInstance())

	starterConfig.SetNsFlagsFromSpec(e.EngineConfig.OciConfig.Linux.Namespaces)

	// user namespace ID mappings
	if e.EngineConfig.OciConfig.Linux != nil {
		starterConfig.AddUIDMappings(e.EngineConfig.OciConfig.Linux.UIDMappings)
		starterConfig.AddGIDMappings(e.EngineConfig.OciConfig.Linux.GIDMappings)
	}

	// open file descriptors (autofs bug path)
	e.prepareFd()

	return nil
}

// prepareInstanceJoinConfig is responsible for getting and applying configuration
// to join a running instance
func (e *EngineOperations) prepareInstanceJoinConfig(starterConfig *starter.Config) error {
	name := instance.ExtractName(e.EngineConfig.GetImage())
	file, err := instance.Get(name)
	if err != nil {
		return err
	}

	// check if SUID workflow is really used with a privileged instance
	if !file.PrivilegedPath() && starterConfig.GetIsSUID() {
		return fmt.Errorf("try to join unprivileged instance with SUID workflow")
	}

	instanceEngineConfig := NewConfig()

	// extract configuration from instance file
	instanceConfig := &config.Common{
		EngineConfig: instanceEngineConfig,
	}
	if err := json.Unmarshal(file.Config, instanceConfig); err != nil {
		return err
	}

	// set namespaces to join
	starterConfig.SetNsPathFromSpec(instanceEngineConfig.OciConfig.Linux.Namespaces)

	if e.EngineConfig.OciConfig.Process == nil {
		e.EngineConfig.OciConfig.Process = &specs.Process{}
	}
	if e.EngineConfig.OciConfig.Process.Capabilities == nil {
		e.EngineConfig.OciConfig.Process.Capabilities = &specs.LinuxCapabilities{}
	}

	// duplicate instance capabilities
	if instanceEngineConfig.OciConfig.Process != nil && instanceEngineConfig.OciConfig.Process.Capabilities != nil {
		e.EngineConfig.OciConfig.Process.Capabilities.Permitted = instanceEngineConfig.OciConfig.Process.Capabilities.Permitted
		e.EngineConfig.OciConfig.Process.Capabilities.Effective = instanceEngineConfig.OciConfig.Process.Capabilities.Effective
		e.EngineConfig.OciConfig.Process.Capabilities.Inheritable = instanceEngineConfig.OciConfig.Process.Capabilities.Inheritable
		e.EngineConfig.OciConfig.Process.Capabilities.Bounding = instanceEngineConfig.OciConfig.Process.Capabilities.Bounding
		e.EngineConfig.OciConfig.Process.Capabilities.Ambient = instanceEngineConfig.OciConfig.Process.Capabilities.Ambient
	}

	if os.Getuid() == 0 {
		if err := e.prepareRootCaps(); err != nil {
			return err
		}
	} else {
		if err := e.prepareUserCaps(); err != nil {
			return err
		}
	}

	e.EngineConfig.OciConfig.Process.NoNewPrivileges = instanceEngineConfig.OciConfig.Process.NoNewPrivileges

	return nil
}

// PrepareConfig checks and prepares the runtime engine config
func (e *EngineOperations) PrepareConfig(masterConn net.Conn, starterConfig *starter.Config) error {
	if e.CommonConfig.EngineName != Name {
		return fmt.Errorf("incorrect engine")
	}

	if !e.EngineConfig.File.AllowSetuid && starterConfig.GetIsSUID() {
		return fmt.Errorf("SUID workflow disabled by administrator")
	}

	// Save the current working directory to restore it in stage 2
	// for relative bind paths
	if pwd, err := os.Getwd(); err == nil {
		e.EngineConfig.SetCwd(pwd)
	} else {
		sylog.Warningf("can't determine current working directory")
		e.EngineConfig.SetCwd("/")
	}

	if e.EngineConfig.OciConfig.Process == nil {
		e.EngineConfig.OciConfig.Process = &specs.Process{}
	}
	if e.EngineConfig.OciConfig.Process.Capabilities == nil {
		e.EngineConfig.OciConfig.Process.Capabilities = &specs.LinuxCapabilities{}
	}

	if e.EngineConfig.GetInstanceJoin() {
		if err := e.prepareInstanceJoinConfig(starterConfig); err != nil {
			return err
		}
	} else {
		if err := e.prepareContainerConfig(starterConfig); err != nil {
			return err
		}
		if err := e.loadImages(); err != nil {
			return err
		}
	}

	starterConfig.SetNoNewPrivs(e.EngineConfig.OciConfig.Process.NoNewPrivileges)

	if e.EngineConfig.OciConfig.Process != nil && e.EngineConfig.OciConfig.Process.Capabilities != nil {
		starterConfig.SetCapabilities(capabilities.Permitted, e.EngineConfig.OciConfig.Process.Capabilities.Permitted)
		starterConfig.SetCapabilities(capabilities.Effective, e.EngineConfig.OciConfig.Process.Capabilities.Effective)
		starterConfig.SetCapabilities(capabilities.Inheritable, e.EngineConfig.OciConfig.Process.Capabilities.Inheritable)
		starterConfig.SetCapabilities(capabilities.Bounding, e.EngineConfig.OciConfig.Process.Capabilities.Bounding)
		starterConfig.SetCapabilities(capabilities.Ambient, e.EngineConfig.OciConfig.Process.Capabilities.Ambient)
	}

	return nil
}

func (e *EngineOperations) loadImages() error {
	images := make([]image.Image, 0)

	// load rootfs image
	writable := e.EngineConfig.GetWritableImage()
	img, err := e.loadImage(e.EngineConfig.GetImage(), writable)
	if err != nil {
		return err
	}

	if writable && !img.Writable {
		sylog.Warningf("Can't set writable flag on image, no write permissions")
		e.EngineConfig.SetWritableImage(false)
	}

	// sandbox are handled differently for security reasons
	if img.Type == image.SANDBOX {
		if img.Path == "/" {
			return fmt.Errorf("/ as sandbox is not authorized")
		}
		if err := os.Chdir(img.Source); err != nil {
			return err
		}
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to determine current working directory: %s", err)
		}
		if cwd != img.Path {
			return fmt.Errorf("path mismatch for sandbox %s != %s", cwd, img.Path)
		}
	}
	if img.Type == image.SIF {
		// query the ECL module, proceed if an ecl config file is found
		ecl, err := syecl.LoadConfig(buildcfg.ECL_FILE)
		if err == nil {
			if err = ecl.ValidateConfig(); err != nil {
				return err
			}
			_, err := ecl.ShouldRunFp(img.File)
			if err != nil {
				return err
			}
		}
	}
	img.RootFS = true
	images = append(images, *img)

	// load overlay images
	for _, overlayImg := range e.EngineConfig.GetOverlayImage() {
		writable := true

		splitted := strings.SplitN(overlayImg, ":", 2)
		if len(splitted) == 2 {
			if splitted[1] == "ro" {
				writable = false
			}
		}

		img, err := e.loadImage(splitted[0], writable)
		if err != nil {
			return fmt.Errorf("failed to open overlay image %s: %s", splitted[0], err)
		}
		images = append(images, *img)
	}

	e.EngineConfig.SetImageList(images)

	return nil
}

func (e *EngineOperations) loadImage(path string, writable bool) (*image.Image, error) {
	imgObject, err := image.Init(path, writable)
	if err != nil {
		return nil, err
	}

	link, err := mainthread.Readlink(imgObject.Source)
	if link != imgObject.Path {
		return nil, fmt.Errorf("resolved path %s doesn't match with opened path %s", imgObject.Path, link)
	}

	if len(e.EngineConfig.File.LimitContainerPaths) != 0 {
		if authorized, err := imgObject.AuthorizedPath(e.EngineConfig.File.LimitContainerPaths); err != nil {
			return nil, err
		} else if !authorized {
			return nil, fmt.Errorf("Singularity image is not in an allowed configured path")
		}
	}
	if len(e.EngineConfig.File.LimitContainerGroups) != 0 {
		if authorized, err := imgObject.AuthorizedGroup(e.EngineConfig.File.LimitContainerGroups); err != nil {
			return nil, err
		} else if !authorized {
			return nil, fmt.Errorf("Singularity image is not owned by required group(s)")
		}
	}
	if len(e.EngineConfig.File.LimitContainerOwners) != 0 {
		if authorized, err := imgObject.AuthorizedOwner(e.EngineConfig.File.LimitContainerOwners); err != nil {
			return nil, err
		} else if !authorized {
			return nil, fmt.Errorf("Singularity image is not owned by required user(s)")
		}
	}

	switch imgObject.Type {
	case image.SANDBOX:
		if !e.EngineConfig.File.AllowContainerDir {
			return nil, fmt.Errorf("configuration disallows users from running sandbox based containers")
		}
	case image.EXT3:
		if !e.EngineConfig.File.AllowContainerExtfs {
			return nil, fmt.Errorf("configuration disallows users from running extFS based containers")
		}
	case image.SQUASHFS:
		if !e.EngineConfig.File.AllowContainerSquashfs {
			return nil, fmt.Errorf("configuration disallows users from running squashFS based containers")
		}
	}
	return imgObject, nil
}
