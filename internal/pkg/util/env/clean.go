// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package env

import (
	"strings"

	"github.com/opencontainers/runtime-tools/generate"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

var alwaysPassKeys = map[string]struct{}{
	"term":        {},
	"http_proxy":  {},
	"https_proxy": {},
	"no_proxy":    {},
	"all_proxy":   {},
	"ftp_proxy":   {},
}

var alwaysOmitKeys = map[string]struct{}{
	"path":            {},
	"ld_library_path": {},
}

// SetContainerEnv cleans environment variables before running the container.
func SetContainerEnv(g *generate.Generator, hostEnvs []string, cleanEnv bool, homeDest string) {
	for _, env := range hostEnvs {
		e := strings.SplitN(env, "=", 2)
		if len(e) != 2 {
			sylog.Verbosef("Can't process environment variable %s", env)
			continue
		}
		if strings.HasPrefix(e[0], "SINGULARITY_") {
			sylog.Verbosef("Not forwarding %s from user to container environment", e[0])
			continue
		}

		switch e[0] {
		case "SINGULARITYENV_PREPEND_PATH":
			g.AddProcessEnv("SING_USER_DEFINED_PREPEND_PATH", e[1])
		case "SINGULARITYENV_APPEND_PATH":
			g.AddProcessEnv("SING_USER_DEFINED_APPEND_PATH", e[1])
		case "SINGULARITYENV_PATH":
			g.AddProcessEnv("SING_USER_DEFINED_PATH", e[1])
		default:
			// transpose host env variables into config
			if key := keyToAdd(e[0], cleanEnv); key != "" {
				g.AddProcessEnv(key, e[1])
			}
		}
	}

	sylog.Verbosef("HOME=%s", homeDest)
	g.AddProcessEnv("HOME", homeDest)
	g.AddProcessEnv("PATH", "/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin")

	if cleanEnv {
		g.AddProcessEnv("LANG", "C")
	}
}

// keyToAdd processes given key and returns a new non-empty key
// if the environment variable should be added to the container.
func keyToAdd(key string, cleanEnv bool) string {
	const envPrefix = "SINGULARITYENV_"

	if strings.HasPrefix(key, envPrefix) {
		return strings.TrimPrefix(key, envPrefix)
	}
	keyLow := strings.ToLower(key)
	if _, ok := alwaysPassKeys[keyLow]; ok {
		return key
	}
	if cleanEnv {
		return ""
	}
	if _, ok := alwaysOmitKeys[keyLow]; ok {
		return ""
	}
	return key
}
