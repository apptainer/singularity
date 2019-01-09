// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build singularity_runtime

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/opencontainers/runtime-tools/generate"
	"github.com/sylabs/singularity/internal/pkg/libexec"
	"github.com/sylabs/singularity/internal/pkg/util/nvidiautils"
	"github.com/sylabs/singularity/pkg/build/types"

	ocitypes "github.com/containers/image/types"
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/build"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	ociclient "github.com/sylabs/singularity/internal/pkg/client/oci"
	"github.com/sylabs/singularity/internal/pkg/instance"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config/oci"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/singularity"
	"github.com/sylabs/singularity/internal/pkg/security"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/env"
	"github.com/sylabs/singularity/internal/pkg/util/exec"
	"github.com/sylabs/singularity/internal/pkg/util/uri"
	"github.com/sylabs/singularity/internal/pkg/util/user"
	library "github.com/sylabs/singularity/pkg/client/library"
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
		cmd.Flags().AddFlag(actionFlags.Lookup("ipc"))
		cmd.Flags().AddFlag(actionFlags.Lookup("net"))
		cmd.Flags().AddFlag(actionFlags.Lookup("network"))
		cmd.Flags().AddFlag(actionFlags.Lookup("network-args"))
		cmd.Flags().AddFlag(actionFlags.Lookup("dns"))
		cmd.Flags().AddFlag(actionFlags.Lookup("nv"))
		cmd.Flags().AddFlag(actionFlags.Lookup("overlay"))
		cmd.Flags().AddFlag(actionFlags.Lookup("pid"))
		cmd.Flags().AddFlag(actionFlags.Lookup("uts"))
		cmd.Flags().AddFlag(actionFlags.Lookup("pwd"))
		cmd.Flags().AddFlag(actionFlags.Lookup("scratch"))
		cmd.Flags().AddFlag(actionFlags.Lookup("userns"))
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
		cmd.Flags().AddFlag(actionFlags.Lookup("containlibs"))
		cmd.Flags().AddFlag(actionFlags.Lookup("no-nv"))
		cmd.Flags().AddFlag(actionFlags.Lookup("tmpdir"))
		cmd.Flags().AddFlag(actionFlags.Lookup("nohttps"))
		cmd.Flags().AddFlag(actionFlags.Lookup("docker-username"))
		cmd.Flags().AddFlag(actionFlags.Lookup("docker-password"))
		cmd.Flags().AddFlag(actionFlags.Lookup("docker-login"))
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

func handleOCI(cmd *cobra.Command, u string) (string, error) {
	authConf, err := makeDockerCredentials(cmd)
	if err != nil {
		sylog.Fatalf("While creating Docker credentials: %v", err)
	}

	sysCtx := &ocitypes.SystemContext{
		OCIInsecureSkipTLSVerify:    noHTTPS,
		DockerInsecureSkipTLSVerify: noHTTPS,
		DockerAuthConfig:            authConf,
	}

	sum, err := ociclient.ImageSHA(u, sysCtx)
	if err != nil {
		return "", fmt.Errorf("failed to get SHA of %v: %v", u, err)
	}

	name := uri.GetName(u)
	imgabs := cache.OciTempImage(sum, name)

	if exists, err := cache.OciTempExists(sum, name); err != nil {
		return "", fmt.Errorf("unable to check if %v exists: %v", imgabs, err)
	} else if !exists {
		sylog.Infof("Converting OCI blobs to SIF format")
		b, err := build.NewBuild(u, imgabs, "sif", "", "", types.Options{TmpDir: tmpDir, NoTest: true, NoHTTPS: noHTTPS, DockerAuthConfig: authConf})
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

	imageName := uri.GetName(u)
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
	imageName := uri.GetName(u)
	imagePath := cache.ShubImage("hash", imageName)

	libexec.PullShubImage(imagePath, u, true, noHTTPS)

	return imagePath, nil
}

func handleNet(u string) (string, error) {
	refParts := strings.Split(u, "/")
	imageName := refParts[len(refParts)-1]
	imagePath := cache.NetImage("hash", imageName)

	exists, err := cache.NetImageExists("hash", imageName)
	if err != nil {
		return "", fmt.Errorf("unable to check if %v exists: %v", imagePath, err)
	}
	if !exists {
		sylog.Infof("Downloading network image")
		libexec.PullNetImage(imagePath, u, true)
	} else {
		sylog.Infof("Use image from cache")
	}

	return imagePath, nil
}

func replaceURIWithImage(cmd *cobra.Command, args []string) {
	// If args[0] is not transport:ref (ex. instance://...) formatted return, not a URI
	t, _ := uri.Split(args[0])
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
		image, err = handleOCI(cmd, args[0])
	case uri.HTTP:
		image, err = handleNet(args[0])
	case uri.HTTPS:
		image, err = handleNet(args[0])
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

	syscall.Umask(0022)

	starter := buildcfg.LIBEXECDIR + "/singularity/bin/starter-suid"

	engineConfig := singularity.NewConfig()

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
	if os.Getuid() == 0 && uidParam != "" {
		u, err := strconv.ParseUint(uidParam, 10, 32)
		if err != nil {
			sylog.Fatalf("failed to parse provided UID")
		}
		targetUID = int(u)
		uid = uint32(targetUID)

		engineConfig.SetTargetUID(targetUID)
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

		engineConfig.SetTargetGID(targetGID)
	} else if gidParam != "" {
		sylog.Warningf("gid security feature requires root privileges")
	}

	if strings.HasPrefix(image, "instance://") {
		instanceName := instance.ExtractName(image)
		file, err := instance.Get(instanceName)
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
	engineConfig.SetAllowSUID(AllowSUID)
	engineConfig.SetKeepPrivs(KeepPrivs)
	engineConfig.SetNoPrivs(NoPrivs)
	engineConfig.SetSecurity(Security)
	engineConfig.SetShell(ShellPath)
	engineConfig.SetLibrariesPath(ContainLibsPath)

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

		_, err := instance.Get(name)
		if err == nil {
			sylog.Fatalf("instance %s already exists", name)
		}

		if IsBoot {
			UtsNamespace = true
			NetNamespace = true
			if Hostname == "" {
				engineConfig.SetHostname(name)
			}
			engineConfig.SetDropCaps("CAP_SYS_BOOT,CAP_SYS_RAWIO")
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

	if engineConfig.GetInstance() {
		stdout, stderr, err := instance.SetLogFile(name, int(uid))
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
