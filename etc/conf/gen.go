// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/singularityware/singularity/src/runtime/engines/common/config"
	"github.com/singularityware/singularity/src/runtime/engines/singularity"
)

func main() {
	Args := os.Args

	if len(Args) < 3 || len(Args) > 4 {
		fmt.Println("Usage: go run ... <template> [infile] <outfile>")
		os.Exit(1)
	}

	tmplPath := filepath.Clean(Args[1])
	outPath := filepath.Clean(Args[2])

	inPath := ""
	if len(Args) == 4 {
		inPath = filepath.Clean(Args[2])
		outPath = filepath.Clean(Args[3])
	}

	genConf(tmplPath, inPath, outPath)
}

// genConf produces a singularity.conf file at out. It retains set configurations from in (leave blank for default)
func genConf(tmpl, in, out string) {
	// Parse current singularity.conf file into c
	c := &singularity.FileConfig{}
	if err := config.Parser(in, c); err != nil {
		fmt.Printf("Unable to parse singularity.conf file: %s\n", err)
		os.Exit(1)
	}

	newOutFile, err := os.OpenFile(out, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Printf("Unable to create file %s: %v\n", out, err)
	}
	defer newOutFile.Close()

	if err := config.Generate(newOutFile, tmpl, c); err != nil {
		fmt.Printf("Unable to generate config file: %v\n", err)
		os.Exit(1)
	}
}
