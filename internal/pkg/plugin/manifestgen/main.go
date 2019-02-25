// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/sylabs/singularity/internal/pkg/plugin"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

// expected args: <path_plugin> <path_manifest>

func main() {
	args := os.Args[1:]
	if len(args) != 2 {
		os.Exit(1)
	}

	fmt.Println(args)
	pl, err := plugin.Initialize(args[0])
	if err != nil {
		sylog.Fatalf("While initializing %s as plugin: %s", args[0], err)
	}

	manifest, err := json.Marshal(pl.Manifest)
	if err != nil {
		sylog.Fatalf("While marshalling manifest to json: %s", err)
	}

	fmt.Println(string(manifest))
	if err := ioutil.WriteFile(args[1], manifest, 0644); err != nil {
		sylog.Fatalf("While writing manifest to file: %s", err)
	}
}
