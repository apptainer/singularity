// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package env

import (
	"os"
	"os/user"
	"strings"

	"github.com/opencontainers/runtime-tools/generate"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

const (
	envPrefix = "SINGULARITYENV_"
)

var alwaysPassKeys = map[string]bool{
	"TERM":        true,
	"http_proxy":  true,
	"HTTP_PROXY":  true,
	"https_proxy": true,
	"HTTPS_PROXY": true,
	"no_proxy":    true,
	"NO_PROXY":    true,
	"all_proxy":   true,
	"ALL_PROXY":   true,
	"ftp_proxy":   true,
	"FTP_PROXY":   true,
}

// SetContainerEnv cleans environment variables before running the container
func SetContainerEnv(g *generate.Generator, env []string, cleanEnv bool, homeDest string) {
	// first deal with special variables that allow user to control $PATH at
	// runtime (meh... special cases)
	if prependPath := os.Getenv("SINGULARITYENV_PREPEND_PATH"); prependPath != "" {
		g.AddProcessEnv("SING_USER_DEFINED_PREPEND_PATH", prependPath)
	}

	if appendPath := os.Getenv("SINGULARITYENV_APPEND_PATH"); appendPath != "" {
		g.AddProcessEnv("SING_USER_DEFINED_APPEND_PATH", appendPath)
	}

	if userPath := os.Getenv("SINGULARITYENV_PATH"); userPath != "" {
		g.AddProcessEnv("SING_USER_DEFINED_PATH", userPath)
	}

	for _, env := range env {
		e := strings.SplitN(env, "=", 2)
		if len(e) != 2 {
			sylog.Verbosef("Can't process environment variable %s", env)
			continue
		}

		if e[0] == "SINGULARITYENV_PREPEND_PATH" ||
			e[0] == "SINGULARITYENV_APPEND_PATH" ||
			e[0] == "SINGULARITYENV_PATH" {
			sylog.Verbosef("Not adding special case PATH control variable %s to container environment", e[0])
			continue
		}

		// Transpose host env variables into config
		if addKey, ok := addIfReq(e[0], cleanEnv); ok {
			g.AddProcessEnv(addKey, e[1])
		}
	}

	if homeDest == "" {
		// Image buid typically runs as root
		usr, err := user.Current()
		homeDest = "/root"

		if err == nil {
			homeDest = usr.HomeDir
		}
	}

	sylog.Verbosef("HOME = %s", homeDest)
	g.AddProcessEnv("HOME", homeDest)
	g.AddProcessEnv("PATH", "/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin")

	// Set LANG env
	if cleanEnv {
		g.AddProcessEnv("LANG", "C")
	}
}

func addIfReq(key string, cleanEnv bool) (string, bool) {
	if strings.HasPrefix(key, envPrefix) {
		return strings.TrimPrefix(key, envPrefix), true
	} else if _, ok := alwaysPassKeys[key]; cleanEnv && !ok {
		return "", false
	}

	return key, true
}
