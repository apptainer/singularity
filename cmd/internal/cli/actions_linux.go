// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/opencontainers/runtime-tools/generate"
	"github.com/sylabs/singularity/internal/pkg/util/nvidiautils"
	"github.com/sylabs/singularity/pkg/image"
	"github.com/sylabs/singularity/pkg/image/unpacker"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/instance"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config/oci"
	singularityConfig "github.com/sylabs/singularity/internal/pkg/runtime/engines/singularity/config"
	"github.com/sylabs/singularity/internal/pkg/security"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/env"
	"github.com/sylabs/singularity/internal/pkg/util/exec"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/internal/pkg/util/user"
)

func convertImage(filename string, unsquashfsPath string) (string, error) {
	img, err := image.Init(filename, false)
	if err != nil {
		return "", fmt.Errorf("could not open image %s: %s", filename, err)
	}
	defer img.File.Close()

	// squashfs only
	if img.Partitions[0].Type != image.SQUASHFS {
		return "", fmt.Errorf("not a squashfs root filesystem")
	}

	// create a reader for rootfs partition
	reader, err := image.NewPartitionReader(img, "", 0)
	if err != nil {
		return "", fmt.Errorf("could not extract root filesystem: %s", err)
	}
	s := unpacker.NewSquashfs()
	if !s.HasUnsquashfs() && unsquashfsPath != "" {
		s.UnsquashfsPath = unsquashfsPath
	}

	// keep compatibility with v2
	tmpdir := os.Getenv("SINGULARITY_LOCALCACHEDIR")
	if tmpdir == "" {
		tmpdir = os.Getenv("SINGULARITY_CACHEDIR")
	}
	if tmpdir == "" {
		pw, err := user.GetPwUID(uint32(os.Getuid()))
		if err != nil {
			return "", fmt.Errorf("could not find current user information: %s", err)
		}
		tmpdir = filepath.Join(pw.Dir, ".singularity", "tmp")
		if !fs.IsDir(tmpdir) {
			if err := os.Mkdir(tmpdir, 0755); err != nil {
				return "", fmt.Errorf("could not create directory %s: %s", tmpdir, err)
			}
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

	starter := buildcfg.LIBEXECDIR + "/singularity/bin/starter-suid"

	engineConfig := singularityConfig.NewConfig()

	configurationFile := buildcfg.SYSCONFDIR + "/singularity/singularity.conf"
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
		instanceName := instance.ExtractName(image)
		file, err := instance.Get(instanceName, instance.SingSubDir)
		if err != nil {
			sylog.Fatalf("%s", err)
		}
		if !file.Privileged {
			UserNamespace = true
		}
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
			sylog.Verbosef("'always use nv = yes' found in singularity.conf")
			sylog.Verbosef("binding nvidia files into container")
		}

		libs, bins, err := nvidiautils.GetNvidiaPath(buildcfg.SINGULARITY_CONFDIR, userPath)
		if err != nil {
			sylog.Infof("Unable to capture nvidia bind points: %v", err)
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

	if !engineConfig.File.AllowSetuid || IsFakeroot {
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
		procname = instance.ProcName(name, pwd.Name)
	} else {
		generator.SetProcessArgs(args)
		procname = "Singularity runtime parent"
	}

	if NetNamespace {
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
	if !UserNamespace {
		if _, err := os.Stat(starter); os.IsNotExist(err) {
			sylog.Verbosef("starter-suid not found, using user namespace")
			UserNamespace = true
		}
	}

	if UserNamespace {
		generator.AddOrReplaceLinuxNamespace("user", "")
		starter = buildcfg.LIBEXECDIR + "/singularity/bin/starter"

		if IsFakeroot {
			generator.AddLinuxUIDMapping(uid, 0, 1)
			generator.AddLinuxGIDMapping(gid, 0, 1)
		} else {
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
		sylog.Verbosef("User namespace requested, convert image %s to sandbox", image)
		dir, err := convertImage(image, unsquashfsPath)
		if err != nil {
			sylog.Fatalf("while extracting %s: %s", image, err)
		}
		engineConfig.SetImage(dir)
		engineConfig.SetDeleteImage(true)
		generator.AddProcessEnv("SINGULARITY_CONTAINER", dir)
	}

	cfg := &config.Common{
		EngineName:   singularityConfig.Name,
		ContainerID:  name,
		EngineConfig: engineConfig,
	}

	configData, err := json.Marshal(cfg)
	if err != nil {
		sylog.Fatalf("CLI Failed to marshal CommonEngineConfig: %s\n", err)
	}

	if engineConfig.GetInstance() {
		stdout, stderr, err := instance.SetLogFile(name, int(uid), instance.SingSubDir)
		if err != nil {
			sylog.Fatalf("failed to create instance log files: %s", err)
		}

		start, err := stderr.Seek(0, os.SEEK_END)
		if err != nil {
			sylog.Warningf("failed to get standard error stream offset: %s", err)
		}

		cmd, err := exec.PipeCommand(starter, []string{procname}, Env, configData)
		cmd.Stdout = stdout
		cmd.Stderr = stderr

		cmdErr := cmd.Run()

		if sylog.GetLevel() != 0 {
			// starter can exit a bit before all errors has been reported
			// by instance process, wait a bit to catch all errors
			time.Sleep(100 * time.Millisecond)

			end, err := stderr.Seek(0, os.SEEK_END)
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
		if err := exec.Pipe(starter, []string{procname}, Env, configData); err != nil {
			sylog.Fatalf("%s", err)
		}
	}
}
