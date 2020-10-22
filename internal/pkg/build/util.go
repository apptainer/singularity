// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	ocitypes "github.com/containers/image/v5/types"
	"github.com/sylabs/singularity/internal/pkg/cache"
	"github.com/sylabs/singularity/internal/pkg/util/env"
	"github.com/sylabs/singularity/pkg/build/types"
	buildtypes "github.com/sylabs/singularity/pkg/build/types"
	"github.com/sylabs/singularity/pkg/sylog"
	"golang.org/x/sys/unix"
)

// ConvertOciToSIf will convert an OCI source into a SIF using the build routines
func ConvertOciToSIF(ctx context.Context, imgCache *cache.Handle, image, cachedImgPath, tmpDir string, noHTTPS, noCleanUp bool, authConf *ocitypes.DockerAuthConfig) error {
	if imgCache == nil {
		return fmt.Errorf("image cache is undefined")
	}

	b, err := NewBuild(
		image,
		Config{
			Dest:      cachedImgPath,
			Format:    "sif",
			NoCleanUp: noCleanUp,
			Opts: buildtypes.Options{
				TmpDir:           tmpDir,
				NoCache:          imgCache.IsDisabled(),
				NoTest:           true,
				NoHTTPS:          noHTTPS,
				DockerAuthConfig: authConf,
				ImgCache:         imgCache,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("unable to create new build: %v", err)
	}

	return b.Full(ctx)
}

func createStageFile(source string, b *types.Bundle, warnMsg string) (string, error) {
	dest := filepath.Join(b.RootfsPath, source)
	if err := unix.Access(dest, unix.R_OK); err != nil {
		sylog.Warningf("%s: while accessing to %s: %s", warnMsg, dest, err)
		return "", nil
	}

	sessionFile := filepath.Join(b.TmpDir, filepath.Base(source))
	stageFile, err := os.Create(sessionFile)
	if err != nil {
		return "", fmt.Errorf("failed to create staging %s file: %s", sessionFile, err)
	}
	defer stageFile.Close()

	content, err := ioutil.ReadFile(source)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %s", source, err)
	}

	// Append an extra blank line to the end of the staged file. This is a trick to fix #5250
	// where a yum install of the `setup` package can fail
	//
	// When /etc/hosts on the host system is unmodified from the distro 'setup' package, yum
	// will try to rename & replace it if the 'setup' package is reinstalled / upgraded. This will
	// fail as it is bind mounted, and cannot be renamed.
	//
	// Adding a newline means the staged file is now different than the one in the 'setup' package
	// and yum will leave the file alone, as it considers it modified.
	content = append(content, []byte("\n")...)

	if _, err := stageFile.Write(content); err != nil {
		return "", fmt.Errorf("failed to copy %s content to %s: %s", source, sessionFile, err)
	}

	return sessionFile, nil
}

func createScript(path string, content []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0755)
	if err != nil {
		return fmt.Errorf("failed to create script: %s", err)
	}

	if _, err := f.Write(content); err != nil {
		f.Close()
		return fmt.Errorf("failed to write script: %s", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close script: %s", err)
	}

	return nil
}

func getSectionScriptArgs(name string, script string, s types.Script) ([]string, error) {
	args := []string{"/bin/sh", "-ex"}
	// trim potential trailing comment from args and append to args list
	sectionParams := strings.Fields(strings.Split(s.Args, "#")[0])

	commandOption := false

	// look for -c option, we assume that everything after is part of -c
	// arguments and we just inject script path as the last arguments of -c
	for i, param := range sectionParams {
		if param == "-c" {
			if len(sectionParams)-1 < i+1 {
				return nil, fmt.Errorf("bad %s section '-c' parameter: missing arguments", name)
			}
			// replace shell "[args...]" arguments list by single
			// argument "shell [args...] script"
			shellArgs := strings.Join(sectionParams[i+1:], " ")
			sectionParams = append(sectionParams[0:i+1], shellArgs+" "+script)
			commandOption = true
			break
		}
	}

	args = append(args, sectionParams...)
	if !commandOption {
		args = append(args, script)
	}

	return args, nil
}

func currentEnvNoSingularity() []string {
	envs := make([]string, 0)

	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, env.SingularityPrefix) {
			envs = append(envs, e)
		}
	}

	return envs
}
