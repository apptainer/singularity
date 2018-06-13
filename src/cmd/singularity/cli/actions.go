// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"encoding/json"
	"os"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/singularityware/singularity/src/docs"
	"github.com/singularityware/singularity/src/pkg/buildcfg"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/exec"
	"github.com/singularityware/singularity/src/runtime/engines/common/config"
	ociConfig "github.com/singularityware/singularity/src/runtime/engines/common/oci/config"
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
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("bind"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("contain"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("containall"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("cleanenv"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("home"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("ipc"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("net"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("nv"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("overlay"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("pid"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("uts"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("pwd"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("scratch"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("userns"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("workdir"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("hostname"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("fakeroot"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("keep-privs"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("no-privs"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("add-caps"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("drop-caps"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("allow-setuid"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("writable"))
	}

	SingularityCmd.AddCommand(ExecCmd)
	SingularityCmd.AddCommand(ShellCmd)
	SingularityCmd.AddCommand(RunCmd)

}

// ExecCmd represents the exec command
var ExecCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		a := append([]string{"/.singularity.d/actions/exec"}, args[1:]...)
		execWrapper(cmd, args[0], a)
	},

	Use:     docs.ExecUse,
	Short:   docs.ExecShort,
	Long:    docs.ExecLong,
	Example: docs.ExecExamples,
}

// ShellCmd represents the shell command
var ShellCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		a := append([]string{"/.singularity.d/actions/shell"}, args[1:]...)
		execWrapper(cmd, args[0], a)
	},

	Use:     docs.ShellUse,
	Short:   docs.ShellShort,
	Long:    docs.ShellLong,
	Example: docs.ShellExamples,
}

// RunCmd represents the run command
var RunCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		a := append([]string{"/.singularity.d/actions/run"}, args[1:]...)
		execWrapper(cmd, args[0], a)
	},

	Use:     docs.RunUse,
	Short:   docs.RunShort,
	Long:    docs.RunLong,
	Example: docs.RunExamples,
}

// TODO: Let's stick this in another file so that that CLI is just CLI
func execWrapper(cobraCmd *cobra.Command, image string, args []string) {
	lvl := "0"

	wrapper := buildcfg.SBINDIR + "/wrapper-suid"

	engineConfig := singularity.NewConfig()
	oci := &ociConfig.RuntimeOciConfig{}
	_ = ociConfig.DefaultRuntimeOciConfig(oci) // must call this to initialize fields in RuntimeOciConfig

	oci.Root.SetPath(image)
	oci.Process.SetArgs(args)
	oci.Process.SetNoNewPrivileges(true)
	engineConfig.SetImage(image)
	engineConfig.SetBindPath(BindPaths)

	oci.RuntimeOciSpec.Linux = &specs.Linux{}
	namespaces := []specs.LinuxNamespace{}
	if NetNamespace {
		namespaces = append(namespaces, specs.LinuxNamespace{Type: specs.NetworkNamespace})
	}
	if UtsNamespace {
		namespaces = append(namespaces, specs.LinuxNamespace{Type: specs.UTSNamespace})
	}
	if PidNamespace {
		namespaces = append(namespaces, specs.LinuxNamespace{Type: specs.PIDNamespace})
	}
	if IpcNamespace {
		namespaces = append(namespaces, specs.LinuxNamespace{Type: specs.IPCNamespace})
	}
	if UserNamespace {
		namespaces = append(namespaces, specs.LinuxNamespace{Type: specs.UserNamespace})
		wrapper = buildcfg.SBINDIR + "/wrapper"
	}
	oci.RuntimeOciSpec.Linux.Namespaces = namespaces

	if verbose {
		lvl = "2"
	}
	if debug {
		lvl = "5"
	}

	oci.Process.SetEnv(os.Environ())

	if pwd, err := os.Getwd(); err == nil {
		oci.Process.SetCwd(pwd)
	} else {
		sylog.Warningf("can't determine current working directory: %s", err)
	}

	Env := []string{"SINGULARITY_MESSAGELEVEL=" + lvl, "SRUNTIME=singularity"}
	progname := "singularity " + args[0]

	cfg := &config.Common{
		EngineName:   "singularity",
		ContainerID:  "new",
		OciConfig:    oci,
		EngineConfig: engineConfig,
	}

	configData, err := json.Marshal(cfg)
	if err != nil {
		sylog.Fatalf("CLI Failed to marshal CommonEngineConfig: %s\n", err)
	}

	if err := exec.Pipe(wrapper, []string{progname}, Env, configData); err != nil {
		sylog.Fatalf("%s", err)
	}
}
