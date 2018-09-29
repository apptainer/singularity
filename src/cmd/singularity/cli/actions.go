// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/opencontainers/runtime-tools/generate"
	"github.com/sylabs/singularity/src/pkg/libexec"
	"github.com/sylabs/singularity/src/pkg/util/arrayhelper"
	"github.com/sylabs/singularity/src/pkg/util/nvidiautils"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/src/docs"
	"github.com/sylabs/singularity/src/pkg/build"
	"github.com/sylabs/singularity/src/pkg/buildcfg"
	"github.com/sylabs/singularity/src/pkg/client/cache"
	library "github.com/sylabs/singularity/src/pkg/client/library"
	ociclient "github.com/sylabs/singularity/src/pkg/client/oci"
	"github.com/sylabs/singularity/src/pkg/instance"
	"github.com/sylabs/singularity/src/pkg/security"
	"github.com/sylabs/singularity/src/pkg/sylog"
	"github.com/sylabs/singularity/src/pkg/util/env"
	"github.com/sylabs/singularity/src/pkg/util/exec"
	"github.com/sylabs/singularity/src/pkg/util/uri"
	"github.com/sylabs/singularity/src/pkg/util/user"
	"github.com/sylabs/singularity/src/runtime/engines/config"
	"github.com/sylabs/singularity/src/runtime/engines/config/oci"
	"github.com/sylabs/singularity/src/runtime/engines/singularity"
)

func init() {
	actionCmds := []*cobra.Command{
		ExecCmd,
		ShellCmd,
		RunCmd,
		TestCmd,
	}

	// TODO : the next n lines of code are repeating too much but I don't
	// know how to shorten them tonight
	for _, cmd := range actionCmds {
		cmd.Flags().AddFlag(actionFlags.Lookup("bind"))
		cmd.Flags().AddFlag(actionFlags.Lookup("contain"))
		cmd.Flags().AddFlag(actionFlags.Lookup("containall"))
		cmd.Flags().AddFlag(actionFlags.Lookup("cleanenv"))
		cmd.Flags().AddFlag(actionFlags.Lookup("home"))
		cmd.Flags().AddFlag(actionFlags.Lookup("network"))
		cmd.Flags().AddFlag(actionFlags.Lookup("network-args"))
		cmd.Flags().AddFlag(actionFlags.Lookup("dns"))
		cmd.Flags().AddFlag(actionFlags.Lookup("nv"))
		cmd.Flags().AddFlag(actionFlags.Lookup("overlay"))
		cmd.Flags().AddFlag(actionFlags.Lookup("pwd"))
		cmd.Flags().AddFlag(actionFlags.Lookup("scratch"))
		cmd.Flags().AddFlag(actionFlags.Lookup("workdir"))
		cmd.Flags().AddFlag(actionFlags.Lookup("hostname"))
		cmd.Flags().AddFlag(actionFlags.Lookup("fakeroot"))
		cmd.Flags().AddFlag(actionFlags.Lookup("keep-privs"))
		cmd.Flags().AddFlag(actionFlags.Lookup("no-privs"))
		cmd.Flags().AddFlag(actionFlags.Lookup("add-caps"))
		cmd.Flags().AddFlag(actionFlags.Lookup("drop-caps"))
		cmd.Flags().AddFlag(actionFlags.Lookup("allow-setuid"))
		cmd.Flags().AddFlag(actionFlags.Lookup("writable"))
		cmd.Flags().AddFlag(actionFlags.Lookup("writable-tmpfs"))
		cmd.Flags().AddFlag(actionFlags.Lookup("no-home"))
		cmd.Flags().AddFlag(actionFlags.Lookup("no-init"))
		cmd.Flags().AddFlag(actionFlags.Lookup("security"))
		cmd.Flags().AddFlag(actionFlags.Lookup("apply-cgroups"))
		cmd.Flags().AddFlag(actionFlags.Lookup("app"))
		cmd.Flags().AddFlag(actionFlags.Lookup("namespace"))
		if cmd == ShellCmd {
			cmd.Flags().AddFlag(actionFlags.Lookup("shell"))
		}
		cmd.Flags().SetInterspersed(false)
	}

	SingularityCmd.AddCommand(ExecCmd)
	SingularityCmd.AddCommand(ShellCmd)
	SingularityCmd.AddCommand(RunCmd)
	SingularityCmd.AddCommand(TestCmd)
}

func handleOCI(u string) (string, error) {
	sum, err := ociclient.ImageSHA(u)
	if err != nil {
		return "", fmt.Errorf("failed to get SHA of %v: %v", u, err)
	}

	name := uri.NameFromURI(u)
	imgabs := cache.OciTempImage(sum, name)

	if exists, err := cache.OciTempExists(sum, name); err != nil {
		return "", fmt.Errorf("unable to check if %v exists: %v", imgabs, err)
	} else if !exists {
		sylog.Infof("Converting OCI blobs to SIF format")
		b, err := build.NewBuild(u, imgabs, "sif", false, false, nil, true, "", "")
		if err != nil {
			return "", fmt.Errorf("unable to create new build: %v", err)
		}

		if err := b.Full(); err != nil {
			return "", fmt.Errorf("unable to build: %v", err)
		}

		sylog.Infof("Image cached as SIF at %s", imgabs)
	}

	return imgabs, nil
}

func handleLibrary(u string) (string, error) {
	libraryImage, err := library.GetImage("https://library.sylabs.io", authToken, u)
	if err != nil {
		return "", err
	}

	imageName := uri.NameFromURI(u)
	imagePath := cache.LibraryImage(libraryImage.Hash, imageName)

	if exists, err := cache.LibraryImageExists(libraryImage.Hash, imageName); err != nil {
		return "", fmt.Errorf("unable to check if %v exists: %v", imagePath, err)
	} else if !exists {
		sylog.Infof("Downloading library image")
		libexec.PullLibraryImage(imagePath, u, "https://library.sylabs.io", false, authToken)
	}

	return imagePath, nil
}

func handleShub(u string) (string, error) {
	imageName := uri.NameFromURI(u)
	imagePath := cache.ShubImage("hash", imageName)

	libexec.PullShubImage(imagePath, u, true)

	return imagePath, nil
}

func replaceURIWithImage(cmd *cobra.Command, args []string) {
	// If args[0] is not transport:ref (ex. intance://...) formatted return, not a URI
	t, _ := uri.SplitURI(args[0])
	if t == "instance" || t == "" {
		return
	}

	var image string
	var err error

	switch t {
	case uri.Library:
		sylabsToken(cmd, args) // Fetch Auth Token for library access

		image, err = handleLibrary(args[0])
	case uri.Shub:
		image, err = handleShub(args[0])
	case ociclient.IsSupported(t):
		image, err = handleOCI(args[0])
	default:
		sylog.Fatalf("Unsupported transport type: %s", t)
	}

	if err != nil {
		sylog.Fatalf("Unable to handle %s uri: %v", args[0], err)
	}

	args[0] = image
	return
}

// ExecCmd represents the exec command
var ExecCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	TraverseChildren:      true,
	Args:                  cobra.MinimumNArgs(2),
	PreRun:                replaceURIWithImage,
	Run: func(cmd *cobra.Command, args []string) {
		a := append([]string{"/.singularity.d/actions/exec"}, args[1:]...)
		execStarter(cmd, args[0], a, "")
	},

	Use:     docs.ExecUse,
	Short:   docs.ExecShort,
	Long:    docs.ExecLong,
	Example: docs.ExecExamples,
}

// ShellCmd represents the shell command
var ShellCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	TraverseChildren:      true,
	Args:                  cobra.MinimumNArgs(1),
	PreRun:                replaceURIWithImage,
	Run: func(cmd *cobra.Command, args []string) {
		a := []string{"/.singularity.d/actions/shell"}
		execStarter(cmd, args[0], a, "")
	},

	Use:     docs.ShellUse,
	Short:   docs.ShellShort,
	Long:    docs.ShellLong,
	Example: docs.ShellExamples,
}

// RunCmd represents the run command
var RunCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	TraverseChildren:      true,
	Args:                  cobra.MinimumNArgs(1),
	PreRun:                replaceURIWithImage,
	Run: func(cmd *cobra.Command, args []string) {
		a := append([]string{"/.singularity.d/actions/run"}, args[1:]...)
		execStarter(cmd, args[0], a, "")
	},

	Use:     docs.RunUse,
	Short:   docs.RunShort,
	Long:    docs.RunLong,
	Example: docs.RunExamples,
}

// TestCmd represents the test command
var TestCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	TraverseChildren:      true,
	Args:                  cobra.MinimumNArgs(1),
	PreRun:                replaceURIWithImage,
	Run: func(cmd *cobra.Command, args []string) {
		a := append([]string{"/.singularity.d/test"}, args[1:]...)
		execStarter(cmd, args[0], a, "")
	},

	Use:     docs.RunTestUse,
	Short:   docs.RunTestShort,
	Long:    docs.RunTestLong,
	Example: docs.RunTestExample,
}

// TODO: Let's stick this in another file so that that CLI is just CLI
func execStarter(cobraCmd *cobra.Command, image string, args []string, name string) {
	targetUID := 0
	targetGID := make([]int, 0)

	procname := ""

	uid := uint32(os.Getuid())
	gid := uint32(os.Getgid())

	starter := buildcfg.SBINDIR + "/starter-suid"

	engineConfig := singularity.NewConfig()

	ociConfig := &oci.Config{}
	generator := generate.Generator{Config: &ociConfig.Spec}

	engineConfig.OciConfig = ociConfig

	generator.SetProcessArgs(args)

	uidParam := security.GetParam(Security, "uid")
	gidParam := security.GetParam(Security, "gid")

	if os.Getuid() == 0 && uidParam != "" {
		u, err := strconv.ParseUint(uidParam, 10, 32)
		if err != nil {
			sylog.Fatalf("failed to parse provided UID")
		}
		targetUID = int(u)
		uid = uint32(targetUID)
	} else if uidParam != "" {
		sylog.Warningf("uid security feature requires root privileges")
	}
	if os.Getuid() == 0 && gidParam != "" {
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
	} else if gidParam != "" {
		sylog.Warningf("gid security feature requires root privileges")
	}

	// temporary check for development
	// TODO: a real URI handler
	if strings.HasPrefix(image, "instance://") {
		instanceName := instance.ExtractName(image)
		file, err := instance.Get(instanceName)
		if err != nil {
			sylog.Fatalf("%s", err)
		}
		if !file.Privileged {
			Namespace = append(Namespace, "user")
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

	if Nvidia {
		NvidiaBindPaths, err := nvidiautils.GetNvidiaBindPath(buildcfg.SINGULARITY_CONFDIR)
		if err != nil {
			sylog.Infof("Unable to capture nvidia bind points: %v", err)
		} else {
			if len(NvidiaBindPaths) == 0 {
				sylog.Warningf("Could not find any NVIDIA libraries on this host!")
				sylog.Warningf("You may need to edit %v/nvliblist.conf", buildcfg.SINGULARITY_CONFDIR)
			} else {
				BindPaths = append(BindPaths, NvidiaBindPaths...)
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
	engineConfig.SetAllowSUID(AllowSUID)
	engineConfig.SetKeepPrivs(KeepPrivs)
	engineConfig.SetNoPrivs(NoPrivs)
	engineConfig.SetSecurity(Security)
	engineConfig.SetShell(ShellPath)

	if ShellPath != "" {
		generator.AddProcessEnv("SINGULARITY_SHELL", ShellPath)
	}

	if os.Getuid() != 0 && CgroupsPath != "" {
		sylog.Warningf("--apply-cgroups requires root privileges")
	} else {
		engineConfig.SetCgroupsPath(CgroupsPath)
	}

	if IsWritable && IsWritableTmpfs {
		sylog.Warningf("Disabling --writable-tmpfs flag, mutually exclusive with --writable")
		engineConfig.SetWritableTmpfs(false)
	} else {
		engineConfig.SetWritableTmpfs(IsWritableTmpfs)
	}

	homeFlag := cobraCmd.Flag("home")
	engineConfig.SetCustomHome(homeFlag.Changed)

	if Hostname != "" {
		Namespace = append(Namespace, "uts")
		engineConfig.SetHostname(Hostname)
	}

	if IsContained || IsContainAll || IsBoot {
		engineConfig.SetContain(true)

		if IsContainAll {
			Namespace = append(Namespace, "pid")
			Namespace = append(Namespace, "ipc")
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
		Namespace = append(Namespace, "user")
	}

	/* if name submitted, run as instance */
	if name != "" {
		Namespace = append(Namespace, "pid")
		Namespace = append(Namespace, "ipc")
		engineConfig.SetInstance(true)
		engineConfig.SetBootInstance(IsBoot)

		_, err := instance.Get(name)
		if err == nil {
			sylog.Fatalf("instance %s already exists", name)
		}
		if err := instance.SetLogFile(name); err != nil {
			sylog.Fatalf("failed to create instance log files: %s", err)
		}

		if IsBoot {
			Namespace = append(Namespace, "uts")
			Namespace = append(Namespace, "network")
			if Hostname == "" {
				engineConfig.SetHostname(name)
			}
			engineConfig.SetDropCaps("CAP_SYS_BOOT,CAP_SYS_RAWIO")
			generator.SetProcessArgs([]string{"/sbin/init"})
		}
		pwd, err := user.GetPwUID(uid)
		if err != nil {
			sylog.Fatalf("failed to retrieve user information for UID %d: %s", uid, err)
		}
		procname = instance.ProcName(name, pwd.Name)
	} else {
		generator.SetProcessArgs(args)
		procname = "Singularity runtime parent"
	}

	if userns := arrayhelper.IsIn(Namespace, "user"); !userns {
		if _, err := os.Stat(starter); os.IsNotExist(err) {
			sylog.Verbosef("starter-suid not found, using user namespace")
			Namespace = append(Namespace, "user")
		}
	}

	Namespace = arrayhelper.Unique(Namespace)
	fmt.Printf("%v\n", Namespace)

	for _, ns := range Namespace {
		switch ns {
		case "uts", "pid", "network", "ipc":
			generator.AddOrReplaceLinuxNamespace(ns, "")
		case "user":
			generator.AddOrReplaceLinuxNamespace("user", "")
			starter = buildcfg.SBINDIR + "/starter"
			if IsFakeroot {
				generator.AddLinuxUIDMapping(uid, 0, 1)
				generator.AddLinuxGIDMapping(gid, 0, 1)
			} else {
				generator.AddLinuxUIDMapping(uid, uid, 1)
				generator.AddLinuxGIDMapping(gid, gid, 1)
			}
		default:
			sylog.Warningf("Unknown --namespace argument \"%s\". Doing nothing.", ns)
		}
	}

	// Copy and cache environment
	environment := os.Environ()

	// Clean environment
	env.SetContainerEnv(&generator, environment, IsCleanEnv, engineConfig.GetHomeDest())

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

	Env := []string{sylog.GetEnvVar(), "SRUNTIME=singularity"}

	generator.AddProcessEnv("SINGULARITY_APPNAME", AppName)

	cfg := &config.Common{
		EngineName:   singularity.Name,
		ContainerID:  name,
		EngineConfig: engineConfig,
	}

	configData, err := json.Marshal(cfg)
	if err != nil {
		sylog.Fatalf("CLI Failed to marshal CommonEngineConfig: %s\n", err)
	}

	runtime.LockOSThread()

	if len(targetGID) > 0 {
		gid := int(targetGID[0])
		gids := targetGID[1:]

		if err := syscall.Setgroups(gids); err != nil {
			sylog.Fatalf("failed to reset groups: %s", err)
		}
		if err := syscall.Setresgid(gid, gid, gid); err != nil {
			sylog.Fatalf("failed to set GID %d: %s", gid, err)
		}
	}

	if targetUID != 0 {
		uid := int(targetUID)
		if err := syscall.Setresuid(uid, uid, uid); err != nil {
			sylog.Fatalf("failed to set UID %d: %s", uid, err)
		}
	}

	if err := exec.Pipe(starter, []string{procname}, Env, configData); err != nil {
		sylog.Fatalf("%s", err)
	}
}
