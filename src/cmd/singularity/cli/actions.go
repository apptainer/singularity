// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"encoding/json"
	"os"
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
		command := addArgs(execCommand, args[1:])
		execWrapper(cmd, args[0], command)
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
		command := addArgs(shellCommand, args[1:])
		execWrapper(cmd, args[0], command)
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
		command := addArgs(runCommand, args[1:])
		execWrapper(cmd, args[0], command)
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

	ociConfig := &oci.Config{}
	generator := generate.NewFromSpec(&ociConfig.Spec)

	generator.SetProcessArgs(args)

	engineConfig.SetImage(image)
	engineConfig.SetBindPath(BindPaths)
	engineConfig.SetOverlayImage(OverlayPath)
	engineConfig.SetWritableImage(IsWritable)
	engineConfig.SetNoHome(NoHome)

	if IsContained || IsContainAll {
		engineConfig.SetContain(true)

		if IsContainAll {
			PidNamespace = true
			IpcNamespace = true
			IsCleanEnv = true
		}
	}

	engineConfig.SetScratchDir(ScratchPath)
	engineConfig.SetWorkdir(WorkdirPath)

	homedir := strings.SplitN(HomePath, ":", 2)
	if len(homedir) == 2 {
		engineConfig.SetHome(homedir[1])
	} else {
		engineConfig.SetHome(homedir[0])
	}
	engineConfig.SetHomeDir(HomePath)

	if IsFakeroot {
		UserNamespace = true
	}

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

		uid := uint32(os.Getuid())
		gid := uint32(os.Getgid())

		if IsFakeroot {
			generator.AddLinuxUIDMapping(uid, 0, 1)
			generator.AddLinuxGIDMapping(gid, 0, 1)
		} else {
			generator.AddLinuxUIDMapping(uid, uid, 1)
			generator.AddLinuxGIDMapping(gid, gid, 1)
		}
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
			if e[0] == "HOME" {
				if !NoHome {
					generator.AddProcessEnv(e[0], engineConfig.GetHome())
				} else {
					generator.AddProcessEnv(e[0], "/")
				}
			} else {
				generator.AddProcessEnv(e[0], e[1])
			}
		}
	}

	if pwd, err := os.Getwd(); err == nil {
		if PwdPath != "" {
			generator.SetProcessCwd(PwdPath)
		} else {
			generator.SetProcessCwd(pwd)
		}
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

func addArgs(command string, args []string) []string {
	argString := strings.Join(args, " ")
	payload := strings.Replace(command, `"$@"`, argString, -1)
	return []string{`/bin/sh`, "-c", payload}
}

const (
	execCommand = `for script in /.singularity.d/env/*.sh; do
    if [ -f "$script" ]; then
        . "$script"
    fi
done

exec "$@"
`

	runCommand = `for script in /.singularity.d/env/*.sh; do
    if [ -f "$script" ]; then
        . "$script"
    fi
done

if test -n "${SINGULARITY_APPNAME:-}"; then

    if test -x "/scif/apps/${SINGULARITY_APPNAME:-}/scif/runscript"; then
        exec "/scif/apps/${SINGULARITY_APPNAME:-}/scif/runscript" "$@"
    else
        echo "No Singularity runscript for contained app: ${SINGULARITY_APPNAME:-}"
        exit 1
    fi

elif test -x "/.singularity.d/runscript"; then
    exec "/.singularity.d/runscript" "$@"
else
    echo "No Singularity runscript found, executing /bin/sh"
    exec /bin/sh "$@"
fi
`

	shellCommand = `for script in /.singularity.d/env/*.sh; do
	if [ -f "$script" ]; then
        . "$script"
    fi
done

if test -n "$SINGULARITY_SHELL" -a -x "$SINGULARITY_SHELL"; then
    exec $SINGULARITY_SHELL "$@"

    echo "ERROR: Failed running shell as defined by '\$SINGULARITY_SHELL'" 1>&2
    exit 1

elif test -x /bin/bash; then
    SHELL=/bin/bash
    PS1="Singularity $SINGULARITY_CONTAINER:\\w> "
    export SHELL PS1
    exec /bin/bash --norc "$@"
elif test -x /bin/sh; then
    SHELL=/bin/sh
    export SHELL
    exec /bin/sh "$@"
else
    echo "ERROR: /bin/sh does not exist in container" 1>&2
fi
exit 1
`
)
