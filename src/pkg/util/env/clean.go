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

// CleanEnv cleans environment variables before running the container
func CleanEnv(g *generate.Generator, NoHome bool, IsCleanEnv bool, HomeDest string, environment []string) {
	for _, env := range environment {
		e := strings.SplitN(env, "=", 2)
		if len(e) != 2 {
			sylog.Verbosef("Can't process environment variable %s", env)
			continue
		}

		// Transpose environment
		if strings.HasPrefix(e[0], "SINGULARITYENV_") {
			e[0] = strings.TrimPrefix(e[0], "SINGULARITYENV_")
		} else if IsCleanEnv && (e[0] != "HOME" &&
			e[0] != "TERM" &&
			e[0] != "LANG" &&
			e[0] != "http_proxy" &&
			e[0] != "https_proxy" &&
			e[0] != "no_proxy" &&
			e[0] != "all_proxy" &&
			e[0] != "ftp_proxy" &&
			e[0] != "SINGULARITY_CONTAINER" &&
			e[0] != "SINGULARITY_NAME") {
			continue
		}

		if e[0] == "HOME" {
			if !NoHome {
				g.AddProcessEnv(e[0], HomeDest)
			} else {
				g.AddProcessEnv(e[0], "/")
			}
		} else {
			g.AddProcessEnv(e[0], e[1])
		}
	}
}
