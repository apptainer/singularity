// Copyright (c) 2019, Sylabs Inc. All rights reserved.
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

	"github.com/opencontainers/runtime-tools/generate"
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/instance"
	"github.com/sylabs/singularity/internal/pkg/plugin"
	"github.com/sylabs/singularity/internal/pkg/runtime/engine/config"
	"github.com/sylabs/singularity/internal/pkg/runtime/engine/config/oci"
	"github.com/sylabs/singularity/internal/pkg/security"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/env"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/internal/pkg/util/starter"
	"github.com/sylabs/singularity/internal/pkg/util/user"
	imgutil "github.com/sylabs/singularity/pkg/image"
	"github.com/sylabs/singularity/pkg/image/unpacker"
	singularityConfig "github.com/sylabs/singularity/pkg/runtime/engines/singularity/config"
	"github.com/sylabs/singularity/pkg/util/crypt"
	"github.com/sylabs/singularity/pkg/util/namespaces"
	"github.com/sylabs/singularity/pkg/util/nvidia"
)

// EnsureRootPriv ensures that a command is executed with root privileges.
func EnsureRootPriv(cmd *cobra.Command, args []string) {
	if os.Geteuid() != 0 {
		sylog.Fatalf("%q command requires root privileges", cmd.CommandPath())
	}
}

func convertImage(filename string, unsquashfsPath string) (string, error) {
	img, err := imgutil.Init(filename, false)
	if err != nil {
		return "", fmt.Errorf("could not open image %s: %s", filename, err)
	}
	defer img.File.Close()

	if !img.HasRootFs() {
		return "", fmt.Errorf("no root filesystem found in %s", filename)
	}

	// squashfs only
	if img.Partitions[0].Type != imgutil.SQUASHFS {
		return "", fmt.Errorf("not a squashfs root filesystem")
	}

	// create a reader for rootfs partition
	reader, err := imgutil.NewPartitionReader(img, "", 0)
	if err != nil {
		return "", fmt.Errorf("could not extract root filesystem: %s", err)
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
	dir, err := ioutil.TempDir(tmpdir, "rootfs-")
	if err != nil {
		return "", fmt.Errorf("could not create temporary sandbox: %s", err)
	}

	// extract root filesystem
	if err := s.ExtractAll(reader, dir); err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("root filesystem extraction failed: %s", err)
	}

	return dir, err
}

// TODO: Let's stick this in another file so that that CLI is just CLI
func execStarter(cobraCmd *cobra.Command, image string, args []string, name string) {
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

	syscall.Umask(0022)

	engineConfig := singularityConfig.NewConfig()

	configurationFile := buildcfg.SINGULARITY_CONF_FILE
	if err := config.Parser(configurationFile, engineConfig.File); err != nil {
		sylog.Fatalf("Unable to parse singularity.conf file: %s", err)
	}

	ociConfig := &oci.Config{}
	generator := generate.Generator{Config: &ociConfig.Spec}

	engineConfig.OciConfig = ociConfig

	generator.SetProcessArgs(args)

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

	useSuid := true

	// singularity was compiled with '--without-suid' option
	if buildcfg.SINGULARITY_SUID_INSTALL == 0 {
		useSuid = false
	}

	// use non privileged starter binary:
	// - if we are the root user
	// - if we are already running inside a user namespace
	// - if user namespace is requested
	// - if 'allow setuid = no' is set in singularity.conf
	if uid == 0 || insideUserNs || UserNamespace || !engineConfig.File.AllowSetuid {
		useSuid = false
		if buildcfg.SINGULARITY_SUID_INSTALL == 1 && !engineConfig.File.AllowSetuid {
			sylog.Verbosef("'allow setuid' set to 'no' by configuration, fallback to user namespace")
			UserNamespace = true
		}
	}

	if !NoNvidia && (Nvidia || engineConfig.File.AlwaysUseNv) {
		userPath := os.Getenv("USER_PATH")

		if engineConfig.File.AlwaysUseNv {
			sylog.Verbosef("'always use nv = yes' found in singularity.conf")
			sylog.Verbosef("binding nvidia files into container")
		}

		libs, bins, err := nvidia.Paths(buildcfg.NVIDIALIBS_FILE, userPath)
		if err != nil {
			sylog.Warningf("Unable to capture NVIDIA bind points: %v", err)
		} else {
			if len(bins) == 0 {
				sylog.Infof("Could not find any NVIDIA binaries on this host!")
			} else {
				if IsWritable {
					sylog.Warningf("NVIDIA binaries may not be bound with --writable")
				}
				binaries := make([]string, len(bins))
				for i, binary := range bins {
					usrBinBinary := filepath.Join("/usr/bin", filepath.Base(binary))
					binaries[i] = strings.Join([]string{binary, usrBinBinary}, ":")
				}
				engineConfig.SetFilesPath(binaries)
			}
			if len(libs) == 0 {
				sylog.Warningf("Could not find any NVIDIA libraries on this host!")
				sylog.Warningf("You may need to edit %v/nvliblist.conf", buildcfg.SINGULARITY_CONFDIR)
			} else {
				engineConfig.SetLibrariesPath(libs)
			}
		}
		// bind persistenced socket if found
		engineConfig.AppendFilesPath(nvidia.IpcsPath(userPath)...)
	}

	// early check for key material before we start engine so we can fail fast if missing
	// we do not need this check when joining a running instance, just for starting a container
	if !engineConfig.GetInstanceJoin() {
		sylog.Debugf("Checking for encrypted system partition")
		img, err := imgutil.Init(engineConfig.GetImage(), false)
		if err != nil {
			sylog.Fatalf("could not open image %s: %s", engineConfig.GetImage(), err)
		}

		if !img.HasRootFs() {
			sylog.Fatalf("no root filesystem found in %s", engineConfig.GetImage())
		}

		// ensure we have decryption material
		if img.Partitions[0].Type == imgutil.ENCRYPTSQUASHFS {
			sylog.Debugf("Encrypted container filesystem detected")

			keyInfo, err := getEncryptionMaterial(cobraCmd)
			if err != nil {
				sylog.Fatalf("While handling encryption material: %v", err)
			}

			plaintextKey, err := crypt.PlaintextKey(keyInfo, engineConfig.GetImage())
			if err != nil {
				sylog.Fatalf("Cannot retrieve key from image %s: %+v", engineConfig.GetImage(), err)
			}

			engineConfig.SetEncryptionKey(plaintextKey)
		}

		// don't defer this call as in all cases it won't be
		// called before execing starter, so it would leak the
		// image file descriptor to the container process
		img.File.Close()
	}

	engineConfig.SetBindPath(BindPaths)
	if len(FuseMount) > 0 {
		/* If --fusemount is given, imply --pid */
		PidNamespace = true
		engineConfig.SetFuseMount(FuseMount)
	}
	engineConfig.SetNetwork(Network)
	engineConfig.SetDNS(DNS)
	engineConfig.SetNetworkArgs(NetworkArgs)
	engineConfig.SetOverlayImage(OverlayPath)
	engineConfig.SetWritableImage(IsWritable)
	engineConfig.SetNoHome(NoHome)
	engineConfig.SetNv(Nvidia)
	engineConfig.SetAddCaps(AddCaps)
	engineConfig.SetDropCaps(DropCaps)

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

	if !homeFlag.Changed && IsFakeroot {
		engineConfig.SetCustomHome(true)
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
	if !UserNamespace && uid != 0 && buildcfg.SINGULARITY_SUID_INSTALL == 0 {
		sylog.Verbosef("Unprivileged installation: using user namespace")
		UserNamespace = true
		useSuid = false
	}

	if UserNamespace {
		generator.AddOrReplaceLinuxNamespace("user", "")

		if !IsFakeroot {
			generator.AddLinuxUIDMapping(uid, uid, 1)
			generator.AddLinuxGIDMapping(gid, gid, 1)
		}
	}

	// Copy and cache environment
	environment := os.Environ()

	// Clean environment
	env.SetContainerEnv(&generator, environment, IsCleanEnv, engineConfig.GetHomeDest())

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
		unsquashfsPath := ""
		if engineConfig.File.MksquashfsPath != "" {
			d := filepath.Dir(engineConfig.File.MksquashfsPath)
			unsquashfsPath = filepath.Join(d, "unsquashfs")
		}
		sylog.Verbosef("User namespace requested, convert image %s to sandbox", image)
		sylog.Infof("Convert SIF file to sandbox...")
		dir, err := convertImage(image, unsquashfsPath)
		if err != nil {
			sylog.Fatalf("while extracting %s: %s", image, err)
		}
		engineConfig.SetImage(dir)
		engineConfig.SetDeleteImage(true)
		generator.AddProcessEnv("SINGULARITY_CONTAINER", dir)

		// if '--disable-cache' flag, then remove original SIF after converting to sandbox
		if disableCache {
			sylog.Debugf("Removing tmp image: %s", image)
			err := os.Remove(image)
			if err != nil {
				sylog.Errorf("unable to remove tmp image: %s: %v", image, err)
			}
		}
	}

	for _, m := range plugin.EngineConfigMutators() {
		sylog.Debugf("Running runtime mutator from plugin %s", m.PluginName)
		m.Mutate(engineConfig)
	}

	cfg := &config.Common{
		EngineName:   singularityConfig.Name,
		ContainerID:  name,
		EngineConfig: engineConfig,
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
