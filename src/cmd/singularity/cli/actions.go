// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/opencontainers/runtime-tools/generate"

	"github.com/singularityware/singularity/src/docs"
	"github.com/singularityware/singularity/src/pkg/buildcfg"
	"github.com/singularityware/singularity/src/pkg/instance"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/exec"
	"github.com/singularityware/singularity/src/pkg/util/user"
	"github.com/singularityware/singularity/src/runtime/engines/config"
	"github.com/singularityware/singularity/src/runtime/engines/config/oci"
	"github.com/singularityware/singularity/src/runtime/engines/singularity"
	"github.com/spf13/cobra"
)

func init() {
	actionCmds := []*cobra.Command{
		ExecCmd,
		ShellCmd,
		RunCmd,
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
		//cmd.Flags().AddFlag(actionFlags.Lookup("writable"))
		cmd.Flags().AddFlag(actionFlags.Lookup("no-home"))
		cmd.Flags().AddFlag(actionFlags.Lookup("no-init"))
		cmd.Flags().SetInterspersed(false)
	}

	SingularityCmd.AddCommand(ExecCmd)
	SingularityCmd.AddCommand(ShellCmd)
	SingularityCmd.AddCommand(RunCmd)

}

// ExecCmd represents the exec command
var ExecCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	TraverseChildren:      true,
	Args:                  cobra.MinimumNArgs(2),
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
	Run: func(cmd *cobra.Command, args []string) {
		a := append([]string{"/.singularity.d/actions/run"}, args[1:]...)
		execStarter(cmd, args[0], a, "")
	},

	Use:     docs.RunUse,
	Short:   docs.RunShort,
	Long:    docs.RunLong,
	Example: docs.RunExamples,
}

// TODO: Let's stick this in another file so that that CLI is just CLI
func execStarter(cobraCmd *cobra.Command, image string, args []string, name string) {
	procname := ""

	uid := uint32(os.Getuid())
	gid := uint32(os.Getgid())

	starter := buildcfg.SBINDIR + "/starter-suid"

	engineConfig := singularity.NewConfig()

	ociConfig := &oci.Config{}
	generator := generate.Generator{Config: &ociConfig.Spec}

	engineConfig.OciConfig = ociConfig

	generator.SetProcessArgs(args)

	// temporary check for development
	// TODO: a real URI handler
	// if image == "" { // if image arg is blank check for an env var
	// 	if ev := os.Getenv("SINGULARITY_IMAGE"); ev != "" {
	// 		image = ev
	// 	}
	// }
	if strings.HasPrefix(image, "instance://") {
		instanceName := instance.ExtractName(image)
		file, err := instance.Get(instanceName)
		if err != nil {
			sylog.Fatalf("%s", err)
		}
		if !file.Privileged {
			UserNamespace = true
		}
		engineConfig.SetImage(image)
		engineConfig.SetInstanceJoin(true)
	} else {
		abspath, err := filepath.Abs(image)
		if err != nil {
			sylog.Fatalf("Failed to determine image absolute path for %s: %s", image, err)
		}
		engineConfig.SetImage(abspath)
	}

	engineConfig.SetBindPath(BindPaths, "SINGULARITY_BINDPATH")
	engineConfig.SetOverlayImage(OverlayPath, "SINGULARITY_OVERLAYIMAGE")
	engineConfig.SetWritableImage(IsWritable, "SINGULARITY_WRITABLE")
	engineConfig.SetNoHome(NoHome, "SINGULARITY_NOHOME")
	engineConfig.SetNv(Nvidia, "SINGULARITY_NV")
	engineConfig.SetAddCaps(AddCaps, "SINGULARITY_ADDCAPS")
	engineConfig.SetDropCaps(DropCaps, "SINGULARITY_DROPCAPS")
	engineConfig.SetAllowSUID(AllowSUID, "SINGULARITY_SUID")
	engineConfig.SetKeepPrivs(KeepPrivs, "SINGULARITY_KEEPPRIVS")
	engineConfig.SetNoPrivs(NoPrivs, "SINGULARITY_NOPRIVS")

	homeFlag := cobraCmd.Flag("home")
	engineConfig.SetCustomHome(homeFlag.Changed)

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
	engineConfig.SetHomeSource(HomeSpec)
	engineConfig.SetHomeDest(HomeSpec)

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
		if err := instance.SetLogFile(name); err != nil {
			sylog.Fatalf("failed to create instance log files: %s", err)
		}

		if IsBoot {
			UtsNamespace = true
			NetNamespace = true
			if Hostname == "" {
				engineConfig.SetHostname(name)
			}
			engineConfig.SetDropCaps("CAP_SYS_BOOT,CAP_SYS_RAWIO", "")
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
	if UserNamespace {
		generator.AddOrReplaceLinuxNamespace("user", "")
		starter = buildcfg.SBINDIR + "/starter"

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
	for _, env := range environment {
		e := strings.SplitN(env, "=", 2)
		if len(e) != 2 {
			sylog.Verbosef("can't process environment variable %s", env)
			continue
		}

		// Transpose environment
		if strings.HasPrefix(e[0], "SINGULARITYENV_") {
			e[0] = strings.TrimPrefix(e[0], "SINGULARITYENV_")
		} else if IsCleanEnv {
			continue
		}

		if e[0] == "HOME" {
			if !NoHome {
				generator.AddProcessEnv(e[0], engineConfig.GetHomeDest())
			} else {
				generator.AddProcessEnv(e[0], "/")
			}
		} else {
			generator.AddProcessEnv(e[0], e[1])
		}
	}

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

	cfg := &config.Common{
		EngineName:   singularity.Name,
		ContainerID:  name,
		EngineConfig: engineConfig,
	}

	configData, err := json.Marshal(cfg)
	if err != nil {
		sylog.Fatalf("CLI Failed to marshal CommonEngineConfig: %s\n", err)
	}

	if err := exec.Pipe(starter, []string{procname}, Env, configData); err != nil {
		sylog.Fatalf("%s", err)
	}
}
