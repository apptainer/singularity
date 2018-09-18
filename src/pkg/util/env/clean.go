// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package env

import (
	"strings"

	"github.com/opencontainers/runtime-tools/generate"
	"github.com/singularityware/singularity/src/pkg/sylog"
)

const (
	envPrefix = "SINGULARITYENV_"
)

var alwaysPassKeys = map[string]bool{
	"TERM":        true,
	"http_proxy":  true,
	"https_proxy": true,
	"no_proxy":    true,
	"all_proxy":   true,
	"ftp_proxy":   true,
}

// SetContainerEnv cleans environment variables before running the container
func SetContainerEnv(g *generate.Generator, env []string, cleanEnv bool, homeDest string) {
	for _, env := range env {
		e := strings.SplitN(env, "=", 2)
		if len(e) != 2 {
			sylog.Verbosef("Can't process environment variable %s", env)
			continue
		}

		// Transpose host env variables into config
		if addKey, ok := addIfReq(e[0], cleanEnv); ok {
			g.AddProcessEnv(addKey, e[1])
		}
	}

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
