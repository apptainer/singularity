// Copyright (c) 2019-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/instance"
	"github.com/sylabs/singularity/internal/pkg/plugin"
	"github.com/sylabs/singularity/internal/pkg/runtime/engine/config/oci"
	"github.com/sylabs/singularity/internal/pkg/runtime/engine/config/oci/generate"
	"github.com/sylabs/singularity/internal/pkg/security"
	"github.com/sylabs/singularity/internal/pkg/util/env"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/internal/pkg/util/shell/interpreter"
	"github.com/sylabs/singularity/internal/pkg/util/starter"
	"github.com/sylabs/singularity/internal/pkg/util/user"
	imgutil "github.com/sylabs/singularity/pkg/image"
	"github.com/sylabs/singularity/pkg/image/unpacker"
	clicallback "github.com/sylabs/singularity/pkg/plugin/callback/cli"
	singularitycallback "github.com/sylabs/singularity/pkg/plugin/callback/runtime/engine/singularity"
	"github.com/sylabs/singularity/pkg/runtime/engine/config"
	singularityConfig "github.com/sylabs/singularity/pkg/runtime/engine/singularity/config"
	"github.com/sylabs/singularity/pkg/sylog"
	"github.com/sylabs/singularity/pkg/util/capabilities"
	"github.com/sylabs/singularity/pkg/util/crypt"
	"github.com/sylabs/singularity/pkg/util/fs/proc"
	"github.com/sylabs/singularity/pkg/util/gpu"
	"github.com/sylabs/singularity/pkg/util/namespaces"
	"github.com/sylabs/singularity/pkg/util/rlimit"
	"github.com/sylabs/singularity/pkg/util/singularityconf"
	"golang.org/x/sys/unix"
)

// convertImage extracts the image found at filename to directory dir within a temporary directory
// tempDir. If the unsquashfs binary is not located, the binary at unsquashfsPath is used. It is
// the caller's responsibility to remove tempDir when no longer needed.
func convertImage(filename string, unsquashfsPath string) (tempDir, imageDir string, err error) {
	img, err := imgutil.Init(filename, false)
	if err != nil {
		return "", "", fmt.Errorf("could not open image %s: %s", filename, err)
	}
	defer img.File.Close()

	part, err := img.GetRootFsPartition()
	if err != nil {
		return "", "", fmt.Errorf("while getting root filesystem in %s: %s", filename, err)
	}

	// Nice message if we have been given an older ext3 image, which cannot be extracted due to lack of privilege
	// to loopback mount.
	if part.Type == imgutil.EXT3 {
		sylog.Errorf("File %q is an ext3 format continer image.", filename)
		sylog.Errorf("Only SIF and squashfs images can be extracted in unprivileged mode.")
		sylog.Errorf("Use `singularity build` to convert this image to a SIF file using a setuid install of Singularity.")
	}

	// Only squashfs can be extracted
	if part.Type != imgutil.SQUASHFS {
		return "", "", fmt.Errorf("not a squashfs root filesystem")
	}

	// create a reader for rootfs partition
	reader, err := imgutil.NewPartitionReader(img, "", 0)
	if err != nil {
		return "", "", fmt.Errorf("could not extract root filesystem: %s", err)
	}
	s := unpacker.NewSquashfs()
	if !s.HasUnsquashfs() && unsquashfsPath != "" {
		s.UnsquashfsPath = unsquashfsPath
	}

	// keep compatibility with v2
	tmpdir := os.Getenv("SINGULARITY_TMPDIR")
	if tmpdir == "" {
		tmpdir = os.Getenv("SINGULARITY_LOCALCACHEDIR")
		if tmpdir == "" {
			tmpdir = os.Getenv("SINGULARITY_CACHEDIR")
		}
	}

	// create temporary sandbox
	tempDir, err = ioutil.TempDir(tmpdir, "rootfs-")
	if err != nil {
		return "", "", fmt.Errorf("could not create temporary sandbox: %s", err)
	}
	defer func() {
		if err != nil {
			os.RemoveAll(tempDir)
		}
	}()

	// create an inner dir to extract to, so we don't clobber the secure permissions on the tmpDir.
	imageDir = filepath.Join(tempDir, "root")
	if err := os.Mkdir(imageDir, 0755); err != nil {
		return "", "", fmt.Errorf("could not create root directory: %s", err)
	}

	// extract root filesystem
	if err := s.ExtractAll(reader, imageDir); err != nil {
		return "", "", fmt.Errorf("root filesystem extraction failed: %s", err)
	}

	return tempDir, imageDir, err
}

// checkHidepid checks if hidepid is set on /proc mount point, when this
// option is an instance started with setuid workflow could not even be
// joined later or stopped correctly.
func hidepidProc() bool {
	entries, err := proc.GetMountInfoEntry("/proc/self/mountinfo")
	if err != nil {
		sylog.Warningf("while reading /proc/self/mountinfo: %s", err)
		return false
	}
	for _, e := range entries {
		if e.Point == "/proc" {
			for _, o := range e.SuperOptions {
				if strings.HasPrefix(o, "hidepid=") {
					return true
				}
			}
		}
	}
	return false
}

// Set engine flags to disable mounts, to allow overriding them if they are set true
// in the singularity.conf
func setNoMountFlags(c *singularityConfig.EngineConfig) {
	for _, v := range NoMount {
		switch v {
		case "proc":
			c.SetNoProc(true)
		case "sys":
			c.SetNoSys(true)
		case "dev":
			c.SetNoDev(true)
		case "devpts":
			c.SetNoDevPts(true)
		case "home":
			c.SetNoHome(true)
		case "tmp":
			c.SetNoTmp(true)
		case "hostfs":
			c.SetNoHostfs(true)
		case "cwd":
			c.SetNoCwd(true)
		default:
			sylog.Warningf("Ignoring unknown mount type '%s'", v)
		}
	}
}

// TODO: Let's stick this in another file so that that CLI is just CLI
func execStarter(cobraCmd *cobra.Command, image string, args []string, name string) {
	var err error

	targetUID := 0
	targetGID := make([]int, 0)

	procname := ""

	uid := uint32(os.Getuid())
	gid := uint32(os.Getgid())
	insideUserNs, _ := namespaces.IsInsideUserNamespace(os.Getpid())

	// Are we running from a privileged account?
	isPrivileged := uid == 0
	checkPrivileges := func(cond bool, desc string, fn func()) {
		if !cond {
			return
		}

		if !isPrivileged {
			sylog.Fatalf("%s requires root privileges", desc)
		}

		fn()
	}

	engineConfig := singularityConfig.NewConfig()

	engineConfig.File = singularityconf.GetCurrentConfig()
	if engineConfig.File == nil {
		sylog.Fatalf("Unable to get singularity configuration")
	}

	ociConfig := &oci.Config{}
	generator := generate.New(&ociConfig.Spec)

	engineConfig.OciConfig = ociConfig

	generator.SetProcessArgs(args)

	currMask := syscall.Umask(0022)
	if !NoUmask {
		// Save the current umask, to be set for the process run in the container
		// https://github.com/hpcng/singularity/issues/5214
		sylog.Debugf("Saving umask %04o for propagation into container", currMask)
		engineConfig.SetUmask(currMask)
		engineConfig.SetRestoreUmask(true)
	}

	uidParam := security.GetParam(Security, "uid")
	gidParam := security.GetParam(Security, "gid")

	// handle target UID/GID for root user
	checkPrivileges(uidParam != "", "uid security feature", func() {
		u, err := strconv.ParseUint(uidParam, 10, 32)
		if err != nil {
			sylog.Fatalf("failed to parse provided UID")
		}
		targetUID = int(u)
		uid = uint32(targetUID)

		engineConfig.SetTargetUID(targetUID)
	})

	checkPrivileges(gidParam != "", "gid security feature", func() {
		gids := strings.Split(gidParam, ":")
		for _, id := range gids {
			g, err := strconv.ParseUint(id, 10, 32)
			if err != nil {
				sylog.Fatalf("failed to parse provided GID")
			}
			targetGID = append(targetGID, int(g))
		}
		if len(gids) > 0 {
			gid = uint32(targetGID[0])
		}

		engineConfig.SetTargetGID(targetGID)
	})

	if strings.HasPrefix(image, "instance://") {
		if name != "" {
			sylog.Fatalf("Starting an instance from another is not allowed")
		}
		instanceName := instance.ExtractName(image)
		file, err := instance.Get(instanceName, instance.SingSubDir)
		if err != nil {
			sylog.Fatalf("%s", err)
		}
		UserNamespace = file.UserNs
		generator.AddProcessEnv("SINGULARITY_CONTAINER", file.Image)
		generator.AddProcessEnv("SINGULARITY_NAME", filepath.Base(file.Image))
		engineConfig.SetImage(image)
		engineConfig.SetInstanceJoin(true)
	} else {
		abspath, err := filepath.Abs(image)
		generator.AddProcessEnv("SINGULARITY_CONTAINER", abspath)
		generator.AddProcessEnv("SINGULARITY_NAME", filepath.Base(abspath))
		if err != nil {
			sylog.Fatalf("Failed to determine image absolute path for %s: %s", image, err)
		}
		engineConfig.SetImage(abspath)
	}

	// privileged installation by default
	useSuid := true

	// singularity was compiled with '--without-suid' option
	if buildcfg.SINGULARITY_SUID_INSTALL == 0 {
		useSuid = false

		if !UserNamespace && uid != 0 {
			sylog.Verbosef("Unprivileged installation: using user namespace")
			UserNamespace = true
		}
	}

	// use non privileged starter binary:
	// - if running as root
	// - if already running inside a user namespace
	// - if user namespace is requested
	// - if running as user and 'allow setuid = no' is set in singularity.conf
	if uid == 0 || insideUserNs || UserNamespace || !engineConfig.File.AllowSetuid {
		useSuid = false

		// fallback to user namespace:
		// - for non root user with setuid installation and 'allow setuid = no'
		// - for root user without effective capability CAP_SYS_ADMIN
		if uid != 0 && buildcfg.SINGULARITY_SUID_INSTALL == 1 && !engineConfig.File.AllowSetuid {
			sylog.Verbosef("'allow setuid' set to 'no' by configuration, fallback to user namespace")
			UserNamespace = true
		} else if uid == 0 && !UserNamespace {
			caps, err := capabilities.GetProcessEffective()
			if err != nil {
				sylog.Fatalf("Could not get process effective capabilities: %s", err)
			}
			if caps&uint64(1<<unix.CAP_SYS_ADMIN) == 0 {
				sylog.Verbosef("Effective capability CAP_SYS_ADMIN is missing, fallback to user namespace")
				UserNamespace = true
			}
		}
	}

	var libs, bins, ipcs []string
	var gpuConfFile, gpuPlatform string
	userPath := os.Getenv("USER_PATH")

	if !NoNvidia && (Nvidia || engineConfig.File.AlwaysUseNv) {
		gpuPlatform = "nv"
		gpuConfFile = filepath.Join(buildcfg.SINGULARITY_CONFDIR, "nvliblist.conf")

		if engineConfig.File.AlwaysUseNv {
			Nvidia = true
			sylog.Verbosef("'always use nv = yes' found in singularity.conf")
			sylog.Verbosef("binding nvidia files into container")
		}

		// bind persistenced socket if found
		ipcs = gpu.NvidiaIpcsPath(userPath)
		libs, bins, err = gpu.NvidiaPaths(gpuConfFile, userPath)

	} else if !NoRocm && (Rocm || engineConfig.File.AlwaysUseRocm) { // Mount rocm GPU
		gpuPlatform = "rocm"
		gpuConfFile = filepath.Join(buildcfg.SINGULARITY_CONFDIR, "rocmliblist.conf")

		if engineConfig.File.AlwaysUseRocm {
			Rocm = true
			sylog.Verbosef("'always use rocm = yes' found in singularity.conf")
			sylog.Verbosef("binding rocm files into container")
		}

		libs, bins, err = gpu.RocmPaths(gpuConfFile, userPath)
	}

	if Nvidia || Rocm {
		if err != nil {
			sylog.Warningf("Unable to capture %s bind points: %v", gpuPlatform, err)
		} else {
			files := make([]string, len(bins)+len(ipcs))

			if len(files) == 0 {
				sylog.Infof("Could not find any %s files on this host!", gpuPlatform)
			} else {
				if IsWritable {
					sylog.Warningf("%s files may not be bound with --writable", gpuPlatform)
				}
				for i, binary := range bins {
					usrBinBinary := filepath.Join("/usr/bin", filepath.Base(binary))
					files[i] = strings.Join([]string{binary, usrBinBinary}, ":")
				}
				for i, ipc := range ipcs {
					files[i+len(bins)] = ipc
				}
				engineConfig.SetFilesPath(files)
			}
			if len(libs) == 0 {
				sylog.Warningf("Could not find any %s libraries on this host!", gpuPlatform)
				sylog.Warningf("You may need to manually edit %s", gpuConfFile)
			} else {
				engineConfig.SetLibrariesPath(libs)
			}
		}
	}

	// early check for key material before we start engine so we can fail fast if missing
	// we do not need this check when joining a running instance, just for starting a container
	if !engineConfig.GetInstanceJoin() {
		sylog.Debugf("Checking for encrypted system partition")
		img, err := imgutil.Init(engineConfig.GetImage(), false)
		if err != nil {
			sylog.Fatalf("could not open image %s: %s", engineConfig.GetImage(), err)
		}

		part, err := img.GetRootFsPartition()
		if err != nil {
			sylog.Fatalf("while getting root filesystem in %s: %s", engineConfig.GetImage(), err)
		}

		// ensure we have decryption material
		if part.Type == imgutil.ENCRYPTSQUASHFS {
			sylog.Debugf("Encrypted container filesystem detected")

			keyInfo, err := getEncryptionMaterial(cobraCmd)
			if err != nil {
				sylog.Fatalf("Cannot load key for decryption: %v", err)
			}

			plaintextKey, err := crypt.PlaintextKey(keyInfo, engineConfig.GetImage())
			if err != nil {
				sylog.Errorf("Cannot decrypt %s: %v", engineConfig.GetImage(), err)
				sylog.Fatalf("Please check you are providing the correct key for decryption")
			}

			engineConfig.SetEncryptionKey(plaintextKey)
		}

		// don't defer this call as in all cases it won't be
		// called before execing starter, so it would leak the
		// image file descriptor to the container process
		img.File.Close()
	}

	binds, err := singularityConfig.ParseBindPath(strings.Join(BindPaths, ","))
	if err != nil {
		sylog.Fatalf("while parsing bind path: %s", err)
	}
	engineConfig.SetBindPath(binds)
	generator.AddProcessEnv("SINGULARITY_BIND", strings.Join(BindPaths, ","))

	if len(FuseMount) > 0 {
		/* If --fusemount is given, imply --pid */
		PidNamespace = true
		if err := engineConfig.SetFuseMount(FuseMount); err != nil {
			sylog.Fatalf("while setting fuse mount: %s", err)
		}
	}
	engineConfig.SetNetwork(Network)
	engineConfig.SetDNS(DNS)
	engineConfig.SetNetworkArgs(NetworkArgs)
	engineConfig.SetOverlayImage(OverlayPath)
	engineConfig.SetWritableImage(IsWritable)
	engineConfig.SetNoHome(NoHome)
	setNoMountFlags(engineConfig)
	engineConfig.SetNv(Nvidia)
	engineConfig.SetRocm(Rocm)
	engineConfig.SetAddCaps(AddCaps)
	engineConfig.SetDropCaps(DropCaps)
	engineConfig.SetConfigurationFile(configurationFile)

	checkPrivileges(AllowSUID, "--allow-setuid", func() {
		engineConfig.SetAllowSUID(AllowSUID)
	})

	checkPrivileges(KeepPrivs, "--keep-privs", func() {
		engineConfig.SetKeepPrivs(KeepPrivs)
	})

	engineConfig.SetNoPrivs(NoPrivs)
	engineConfig.SetSecurity(Security)
	engineConfig.SetShell(ShellPath)
	engineConfig.AppendLibrariesPath(ContainLibsPath...)
	engineConfig.SetFakeroot(IsFakeroot)

	if ShellPath != "" {
		generator.AddProcessEnv("SINGULARITY_SHELL", ShellPath)
	}

	checkPrivileges(CgroupsPath != "", "--apply-cgroups", func() {
		engineConfig.SetCgroupsPath(CgroupsPath)
	})

	if IsWritable && IsWritableTmpfs {
		sylog.Warningf("Disabling --writable-tmpfs flag, mutually exclusive with --writable")
		engineConfig.SetWritableTmpfs(false)
	} else {
		engineConfig.SetWritableTmpfs(IsWritableTmpfs)
	}

	homeFlag := cobraCmd.Flag("home")
	engineConfig.SetCustomHome(homeFlag.Changed)

	// If we have fakeroot & the home flag has not been used then we have the standard
	// /root location for the root user $HOME in the container.
	// This doesn't count as a SetCustomHome(true), as we are mounting from the real
	// user's standard $HOME -> /root and we want to respect --contain not mounting
	// the $HOME in this case.
	// See https://github.com/sylabs/singularity/pull/5227
	if !homeFlag.Changed && IsFakeroot {
		HomePath = fmt.Sprintf("%s:/root", HomePath)
	}

	// set home directory for the targeted UID if it exists on host system
	if !homeFlag.Changed && targetUID != 0 {
		if targetUID > 500 {
			if pwd, err := user.GetPwUID(uint32(targetUID)); err == nil {
				sylog.Debugf("Target UID requested, set home directory to %s", pwd.Dir)
				HomePath = pwd.Dir
				engineConfig.SetCustomHome(true)
			} else {
				sylog.Verbosef("Home directory for UID %d not found, home won't be mounted", targetUID)
				engineConfig.SetNoHome(true)
				HomePath = "/"
			}
		} else {
			sylog.Verbosef("System UID %d requested, home won't be mounted", targetUID)
			engineConfig.SetNoHome(true)
			HomePath = "/"
		}
	}

	if Hostname != "" {
		UtsNamespace = true
		engineConfig.SetHostname(Hostname)
	}

	checkPrivileges(IsBoot, "--boot", func() {})

	if IsContained || IsContainAll || IsBoot {
		engineConfig.SetContain(true)

		if IsContainAll {
			PidNamespace = true
			IpcNamespace = true
			IsCleanEnv = true
		}
	}

	engineConfig.SetScratchDir(ScratchPath)
	engineConfig.SetWorkdir(WorkdirPath)

	homeSlice := strings.Split(HomePath, ":")

	if len(homeSlice) > 2 || len(homeSlice) == 0 {
		sylog.Fatalf("home argument has incorrect number of elements: %v", len(homeSlice))
	}

	engineConfig.SetHomeSource(homeSlice[0])
	if len(homeSlice) == 1 {
		engineConfig.SetHomeDest(homeSlice[0])
	} else {
		engineConfig.SetHomeDest(homeSlice[1])
	}

	if IsFakeroot {
		UserNamespace = true
	}

	/* if name submitted, run as instance */
	if name != "" {
		PidNamespace = true
		IpcNamespace = true
		engineConfig.SetInstance(true)
		engineConfig.SetBootInstance(IsBoot)

		if useSuid && !UserNamespace && hidepidProc() {
			sylog.Fatalf("hidepid option set on /proc mount, require 'hidepid=0' to start instance with setuid workflow")
		}

		_, err := instance.Get(name, instance.SingSubDir)
		if err == nil {
			sylog.Fatalf("instance %s already exists", name)
		}

		if IsBoot {
			UtsNamespace = true
			NetNamespace = true
			if Hostname == "" {
				engineConfig.SetHostname(name)
			}
			if !KeepPrivs {
				engineConfig.SetDropCaps("CAP_SYS_BOOT,CAP_SYS_RAWIO")
			}
			generator.SetProcessArgs([]string{"/sbin/init"})
		}
		pwd, err := user.GetPwUID(uint32(os.Getuid()))
		if err != nil {
			sylog.Fatalf("failed to retrieve user information for UID %d: %s", os.Getuid(), err)
		}
		procname, err = instance.ProcName(name, pwd.Name)
		if err != nil {
			sylog.Fatalf("%s", err)
		}
	} else {
		generator.SetProcessArgs(args)
		procname = "Singularity runtime parent"
	}

	if NetNamespace {
		if IsFakeroot && Network != "none" {
			engineConfig.SetNetwork("fakeroot")

			// unprivileged installation could not use fakeroot
			// network because it requires a setuid installation
			// so we fallback to none
			if buildcfg.SINGULARITY_SUID_INSTALL == 0 || !engineConfig.File.AllowSetuid {
				sylog.Warningf(
					"fakeroot with unprivileged installation or 'allow setuid = no' " +
						"could not use 'fakeroot' network, fallback to 'none' network",
				)
				engineConfig.SetNetwork("none")
			}
		}
		generator.AddOrReplaceLinuxNamespace("network", "")
	}
	if UtsNamespace {
		generator.AddOrReplaceLinuxNamespace("uts", "")
	}
	if PidNamespace {
		generator.AddOrReplaceLinuxNamespace("pid", "")
		engineConfig.SetNoInit(NoInit)
	}
	if IpcNamespace {
		generator.AddOrReplaceLinuxNamespace("ipc", "")
	}
	if UserNamespace {
		generator.AddOrReplaceLinuxNamespace("user", "")

		if !IsFakeroot {
			generator.AddLinuxUIDMapping(uid, uid, 1)
			generator.AddLinuxGIDMapping(gid, gid, 1)
		}
	}

	if SingularityEnvFile != "" {
		currentEnv := append(
			os.Environ(),
			"SINGULARITY_IMAGE="+engineConfig.GetImage(),
			"PATH="+os.Getenv("USER_PATH"),
		)

		content, err := ioutil.ReadFile(SingularityEnvFile)
		if err != nil {
			sylog.Fatalf("Could not read %q environment file: %s", SingularityEnvFile, err)
		}

		env, err := interpreter.EvaluateEnv(content, args, currentEnv)
		if err != nil {
			sylog.Fatalf("While processing %s: %s", SingularityEnvFile, err)
		}
		// --env variables will take precedence over variables
		// defined by the environment file
		sylog.Debugf("Setting environment variables from file %s", SingularityEnvFile)
		SingularityEnv = append(env, SingularityEnv...)
	}

	// process --env and --env-file variables for injection
	// into the environment by prefixing them with SINGULARITYENV_
	for _, env := range SingularityEnv {
		e := strings.SplitN(env, "=", 2)
		if len(e) != 2 {
			sylog.Warningf("Ignore environment variable %q: '=' is missing", env)
			continue
		}
		os.Setenv("SINGULARITYENV_"+e[0], e[1])
	}

	// Copy and cache environment
	environment := os.Environ()

	// Clean environment
	singularityEnv := env.SetContainerEnv(generator, environment, IsCleanEnv, engineConfig.GetHomeDest())
	engineConfig.SetSingularityEnv(singularityEnv)

	if pwd, err := os.Getwd(); err == nil {
		engineConfig.SetCwd(pwd)
		if PwdPath != "" {
			generator.SetProcessCwd(PwdPath)
		} else {
			if engineConfig.GetContain() {
				generator.SetProcessCwd(engineConfig.GetHomeDest())
			} else {
				generator.SetProcessCwd(pwd)
			}
		}
	} else {
		sylog.Warningf("can't determine current working directory: %s", err)
	}

	// starter will force the loading of kernel overlay module
	loadOverlay := false
	if !UserNamespace && buildcfg.SINGULARITY_SUID_INSTALL == 1 {
		loadOverlay = true
	}

	generator.AddProcessEnv("SINGULARITY_APPNAME", AppName)

	// convert image file to sandbox if we are using user
	// namespace or if we are currently running inside a
	// user namespace
	if (UserNamespace || insideUserNs) && fs.IsFile(image) {
		convert := true

		if engineConfig.File.ImageDriver != "" {
			// load image driver plugins
			callbackType := (singularitycallback.RegisterImageDriver)(nil)
			callbacks, err := plugin.LoadCallbacks(callbackType)
			if err != nil {
				sylog.Debugf("Loading plugins callbacks '%T' failed: %s", callbackType, err)
			} else {
				for _, callback := range callbacks {
					if err := callback.(singularitycallback.RegisterImageDriver)(true); err != nil {
						sylog.Debugf("While registering image driver: %s", err)
					}
				}
			}
			driver := imgutil.GetDriver(engineConfig.File.ImageDriver)
			if driver != nil && driver.Features()&imgutil.ImageFeature != 0 {
				// the image driver indicates support for image so let's
				// proceed with the image driver without conversion
				convert = false
			}
		}

		if convert {
			unsquashfsPath := ""
			if engineConfig.File.MksquashfsPath != "" {
				d := filepath.Dir(engineConfig.File.MksquashfsPath)
				unsquashfsPath = filepath.Join(d, "unsquashfs")
			}
			sylog.Verbosef("User namespace requested, convert image %s to sandbox", image)
			sylog.Infof("Converting SIF file to temporary sandbox...")
			tempDir, imageDir, err := convertImage(image, unsquashfsPath)
			if err != nil {
				sylog.Fatalf("while extracting %s: %s", image, err)
			}
			engineConfig.SetImage(imageDir)
			engineConfig.SetDeleteTempDir(tempDir)
			generator.AddProcessEnv("SINGULARITY_CONTAINER", imageDir)

			// if '--disable-cache' flag, then remove original SIF after converting to sandbox
			if disableCache {
				sylog.Debugf("Removing tmp image: %s", image)
				err := os.Remove(image)
				if err != nil {
					sylog.Errorf("unable to remove tmp image: %s: %v", image, err)
				}
			}
		}
	}

	// setuid workflow set RLIMIT_STACK to its default value,
	// get the original value to restore it before executing
	// container process
	if useSuid {
		soft, hard, err := rlimit.Get("RLIMIT_STACK")
		if err != nil {
			sylog.Warningf("can't retrieve stack size limit: %s", err)
		}
		generator.AddProcessRlimits("RLIMIT_STACK", hard, soft)
	}

	cfg := &config.Common{
		EngineName:   singularityConfig.Name,
		ContainerID:  name,
		EngineConfig: engineConfig,
	}

	callbackType := (clicallback.SingularityEngineConfig)(nil)
	callbacks, err := plugin.LoadCallbacks(callbackType)
	if err != nil {
		sylog.Fatalf("While loading plugins callbacks '%T': %s", callbackType, err)
	}
	for _, c := range callbacks {
		c.(clicallback.SingularityEngineConfig)(cfg)
	}

	if engineConfig.GetInstance() {
		stdout, stderr, err := instance.SetLogFile(name, int(uid), instance.LogSubDir)
		if err != nil {
			sylog.Fatalf("failed to create instance log files: %s", err)
		}

		start, err := stderr.Seek(0, io.SeekEnd)
		if err != nil {
			sylog.Warningf("failed to get standard error stream offset: %s", err)
		}

		cmdErr := starter.Run(
			procname,
			cfg,
			starter.UseSuid(useSuid),
			starter.WithStdout(stdout),
			starter.WithStderr(stderr),
			starter.LoadOverlayModule(loadOverlay),
		)

		if sylog.GetLevel() != 0 {
			// starter can exit a bit before all errors has been reported
			// by instance process, wait a bit to catch all errors
			time.Sleep(100 * time.Millisecond)

			end, err := stderr.Seek(0, io.SeekEnd)
			if err != nil {
				sylog.Warningf("failed to get standard error stream offset: %s", err)
			}
			if end-start > 0 {
				output := make([]byte, end-start)
				stderr.ReadAt(output, start)
				fmt.Println(string(output))
			}
		}

		if cmdErr != nil {
			sylog.Fatalf("failed to start instance: %s", cmdErr)
		} else {
			sylog.Verbosef("you will find instance output here: %s", stdout.Name())
			sylog.Verbosef("you will find instance error here: %s", stderr.Name())
			sylog.Infof("instance started successfully")
		}
	} else {
		err := starter.Exec(
			procname,
			cfg,
			starter.UseSuid(useSuid),
			starter.LoadOverlayModule(loadOverlay),
		)
		sylog.Fatalf("%s", err)
	}
}
