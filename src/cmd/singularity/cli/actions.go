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
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/exec"
	"github.com/singularityware/singularity/src/runtime/engines/common/config"
	"github.com/singularityware/singularity/src/runtime/engines/common/oci"
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

	wrapper := filepath.Join(buildcfg.SBINDIR, "/wrapper-suid")

	engineConfig := singularity.NewConfig()

	ociConfig := &oci.Config{}
	generator := generate.NewFromSpec(&ociConfig.Spec)

	generator.SetProcessArgs(args)

	engineConfig.SetImage(image)
	engineConfig.SetBindPath(BindPaths)

	if NetNamespace {
		generator.AddOrReplaceLinuxNamespace("network", "")
	}
	if UtsNamespace {
		generator.AddOrReplaceLinuxNamespace("uts", "")
	}
	if PidNamespace {
		generator.AddOrReplaceLinuxNamespace("pid", "")
	}
	if IpcNamespace {
		generator.AddOrReplaceLinuxNamespace("ipc", "")
	}
	if UserNamespace {
		generator.AddOrReplaceLinuxNamespace("user", "")
		wrapper = buildcfg.SBINDIR + "/wrapper"
	}

	if verbose {
		lvl = "2"
	}
	if debug {
		lvl = "5"
	}

	if !IsCleanEnv {
		for _, env := range os.Environ() {
			e := strings.SplitN(env, "=", 2)
			if len(e) != 2 {
				sylog.Verbosef("can't process environment variable %s", env)
				continue
			}
			generator.AddProcessEnv(e[0], e[1])
		}
	}

	if pwd, err := os.Getwd(); err == nil {
		generator.SetProcessCwd(pwd)
	} else {
		sylog.Warningf("can't determine current working directory: %s", err)
	}

	Env := []string{"SINGULARITY_MESSAGELEVEL=" + lvl, "SRUNTIME=singularity"}
	progname := "Singularity runtime parent"

	cfg := &config.Common{
		EngineName:   singularity.Name,
		ContainerID:  "new",
		OciConfig:    ociConfig,
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
