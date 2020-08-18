// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package env

import (
	"strings"

	"github.com/sylabs/singularity/internal/pkg/runtime/engine/config/oci/generate"
	"github.com/sylabs/singularity/pkg/sylog"
)

const (
	// DefaultPath defines default value for PATH environment variable.
	DefaultPath = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
	// SingularityPrefix defines the environment variable prefix SINGULARITY_.
	SingularityPrefix = "SINGULARITY_"
	// SingularityEnvPrefix defines the environment variable prefix SINGULARITYENV_.
	SingularityEnvPrefix = "SINGULARITYENV_"
)

var alwaysPassKeys = map[string]struct{}{
	"TERM":        {},
	"http_proxy":  {},
	"HTTP_PROXY":  {},
	"https_proxy": {},
	"HTTPS_PROXY": {},
	"no_proxy":    {},
	"NO_PROXY":    {},
	"all_proxy":   {},
	"ALL_PROXY":   {},
	"ftp_proxy":   {},
	"FTP_PROXY":   {},
}

// boolean value defines if the variable could be overridden
// with the SINGULARITYENV_ variant.
var alwaysOmitKeys = map[string]bool{
	"HOME":                false,
	"PATH":                false,
	"SINGULARITY_SHELL":   false,
	"SINGULARITY_APPNAME": false,
	"LD_LIBRARY_PATH":     true,
}

// SetContainerEnv cleans environment variables before running the container.
func SetContainerEnv(g *generate.Generator, hostEnvs []string, cleanEnv bool, homeDest string) map[string]string {
	singEnvKeys := make(map[string]string)

	// allow override with SINGULARITYENV_LANG
	if cleanEnv {
		g.AddProcessEnv("LANG", "C")
	}

	for _, env := range hostEnvs {
		e := strings.SplitN(env, "=", 2)
		if len(e) != 2 {
			sylog.Verbosef("Can't process environment variable %s", env)
			continue
		}
		if strings.HasPrefix(e[0], SingularityPrefix) {
			sylog.Verbosef("Not forwarding %s environment variable", e[0])
			continue
		} else if strings.HasPrefix(e[0], SingularityEnvPrefix) {
			key := e[0][len(SingularityEnvPrefix):]
			switch key {
			case "PREPEND_PATH":
				singEnvKeys["SING_USER_DEFINED_PREPEND_PATH"] = e[1]
			case "APPEND_PATH":
				singEnvKeys["SING_USER_DEFINED_APPEND_PATH"] = e[1]
			case "PATH":
				singEnvKeys["SING_USER_DEFINED_PATH"] = e[1]
			default:
				if key == "" {
					continue
				}
				if permitted, ok := alwaysOmitKeys[key]; ok && !permitted {
					sylog.Warningf("Overriding %s environment variable with %s is not permitted", key, e[0])
					continue
				}
				sylog.Verbosef("Forwarding %s as %s environment variable", e[0], key)
				singEnvKeys[key] = e[1]
				g.RemoveProcessEnv(key)
			}
		} else {
			// SINGULARITYENV_ prefixed environment variables will take
			// precedence over the non prefixed variables
			if _, ok := singEnvKeys[e[0]]; ok {
				sylog.Verbosef("Skipping %[1]s environment variable, overridden by %[2]s%[1]s", e[0], SingularityEnvPrefix)
			} else if addHostEnv(e[0], cleanEnv) {
				// transpose host env variables into config
				sylog.Debugf("Forwarding %s environment variable", e[0])
				g.AddProcessEnv(e[0], e[1])
			}
		}
	}

	sylog.Verbosef("Setting HOME=%s", homeDest)
	sylog.Verbosef("Setting PATH=%s", DefaultPath)
	g.AddProcessEnv("HOME", homeDest)
	g.AddProcessEnv("PATH", DefaultPath)

	return singEnvKeys
}

// addHostEnv processes given key and returns if the environment
// variable should be added to the container or not.
func addHostEnv(key string, cleanEnv bool) bool {
	if _, ok := alwaysPassKeys[key]; ok {
		return true
	}
	if _, ok := alwaysOmitKeys[key]; ok || cleanEnv {
		return false
	}
	return true
}
