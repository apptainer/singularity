// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/opencontainers/runtime-tools/generate"
	"github.com/sylabs/singularity/internal/pkg/plugin"
	"github.com/sylabs/singularity/pkg/util/nvidia"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/instance"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config/oci"
	"github.com/sylabs/singularity/internal/pkg/security"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/env"
	"github.com/sylabs/singularity/internal/pkg/util/exec"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/internal/pkg/util/user"
	singularityConfig "github.com/sylabs/singularity/pkg/runtime/engines/singularity/config"
)

// EnsureRootPriv ensures that a command is executed with root privileges.
// To customize the output, arguments can be used to specify the context (e.g., "oci", "plugin"),
// where the first argument (string) will be displayed before the command itself.
func EnsureRootPriv(cmd *cobra.Command, args []string) {
	if os.Geteuid() != 0 {
		if len(args) >= 1 && len(args[0]) > 0 {
			// The first argument is the context
			sylog.Fatalf("command '%s %s' requires root privileges", args[0], cmd.Name())
		} else {
			sylog.Fatalf("command %s requires root privileges", cmd.Name())
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
	engineConfig.File, err = getFileConfig()
	if err != nil {
		sylog.Fatalf("%s", err)
	}

	ociConfig := &oci.Config{}
	generator := generate.Generator{Config: &ociConfig.Spec}

	engineConfig.OciConfig = ociConfig

	allowedSuid := engineConfig.File.AllowSetuid
	// users need to use setuid workflow if they haven't requested
	// a user namespace or if they want to use fakeroot feature
	requireSuid := (!UserNamespace || IsFakeroot) && !isPrivileged
	starterBinary, starterSuid := exec.LookStarterPath(requireSuid, allowedSuid)
	if !starterSuid && requireSuid {
		// setuid workflow is not allowed and/or usable, fallback with user namespace
		if !allowedSuid {
			sylog.Verbosef("Setuid workflow not allowed by 'allow setuid = no' in %s", singularityConfPath)
			if IsFakeroot {
				sylog.Fatalf("fakeroot feature requires 'allow setuid = yes' in %s", singularityConfPath)
			}
		}
		sylog.Verbosef("Fallback with user namespace")
		UserNamespace = true
	} else if starterSuid && IsFakeroot {
		// Set it to true to force image conversion with fakeroot
		// which can use sandbox image only. This restriction could
		// be removed in the future with FUSE's help
		UserNamespace = true
	}

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

	if !NoNvidia && (Nvidia || engineConfig.File.AlwaysUseNv) {
		userPath := os.Getenv("USER_PATH")

		if engineConfig.File.AlwaysUseNv {
			sylog.Verbosef("'always use nv = yes' found in %s", singularityConfPath)
			sylog.Verbosef("binding nvidia files into container")
		}

		libs, bins, err := nvidia.Paths(buildcfg.SINGULARITY_CONFDIR, userPath)
		if err != nil {
			sylog.Warningf("Unable to capture NVIDIA bind points: %v", err)
		} else {
			if len(bins) == 0 {
				sylog.Infof("Could not find any NVIDIA binaries on this host!")
			} else {
				if IsWritable {
					sylog.Warningf("NVIDIA binaries may not be bound with --writable")
				}
				for _, binary := range bins {
					usrBinBinary := filepath.Join("/usr/bin", filepath.Base(binary))
					bind := strings.Join([]string{binary, usrBinBinary}, ":")
					BindPaths = append(BindPaths, bind)
				}
			}
			if len(libs) == 0 {
				sylog.Warningf("Could not find any NVIDIA libraries on this host!")
				sylog.Warningf("You may need to edit %v/nvliblist.conf", buildcfg.SINGULARITY_CONFDIR)
			} else {
				ContainLibsPath = append(ContainLibsPath, libs...)
			}
		}
		// bind persistenced socket if found
		BindPaths = append(BindPaths, nvidia.IpcsPath(userPath)...)
	}

	engineConfig.SetBindPath(BindPaths)
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
	engineConfig.SetLibrariesPath(ContainLibsPath)
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

	// Copy and cache environment
	environment := os.Environ()

	// Clean environment
	env.SetContainerEnv(&generator, environment, IsCleanEnv, engineConfig.GetHomeDest())

	// force to use getwd syscall
	os.Unsetenv("PWD")

	if pwd, err := os.Getwd(); err == nil {
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

	Env := []string{sylog.GetEnvVar()}

	generator.AddProcessEnv("SINGULARITY_APPNAME", AppName)

	// convert image file to sandbox if image contains
	// a squashfs filesystem
	if UserNamespace && fs.IsFile(image) {
		unsquashfsPath := ""
		if engineConfig.File.MksquashfsPath != "" {
			d := filepath.Dir(engineConfig.File.MksquashfsPath)
			unsquashfsPath = filepath.Join(d, "unsquashfs")
		}
		sylog.Infof("Convert SIF file to sandbox...")
		sylog.Verbosef("User namespace requested, convert image %s to sandbox", image)
		dir, err := convertSquashfsToSandbox(image, unsquashfsPath)
		if err != nil {
			sylog.Fatalf("error while converting %s: %s", image, err)
		}
		engineConfig.SetImage(dir)
		engineConfig.SetDeleteImage(true)
		generator.AddProcessEnv("SINGULARITY_CONTAINER", dir)
	}

	plugin.FlagHookCallbacks(engineConfig)

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

		cmd, err := exec.StarterCommand(starterBinary, []string{procname}, Env, cfg)
		if err != nil {
			sylog.Warningf("failed to prepare command: %s", err)
		}

		cmd.Stdout = stdout
		cmd.Stderr = stderr

		cmdErr := cmd.Run()

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
		if err := exec.Starter(starterBinary, []string{procname}, Env, cfg); err != nil {
			sylog.Fatalf("%s", err)
		}
	}
}
