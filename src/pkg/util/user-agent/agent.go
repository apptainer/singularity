// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package useragent

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/singularityware/singularity/src/pkg/buildcfg"
)

// Value contains the Singularity user agent.
//
// For example, "Singularity/3.0.0 (linux amd64) Go/1.10.3".
var Value string

func singularityVersion() string {
	product := strings.Title(buildcfg.PACKAGE_NAME)
	version := strings.Split(buildcfg.PACKAGE_VERSION, "-")[0]
	return fmt.Sprintf("%v/%v", product, version)
}

func goVersion() string {
	version := strings.TrimPrefix(runtime.Version(), "go")
	return fmt.Sprintf("Go/%v", version)
}

func init() {
	Value = fmt.Sprintf("%v (%v %v) %v",
		singularityVersion(),
		strings.Title(runtime.GOOS),
		runtime.GOARCH,
		goVersion())
}
