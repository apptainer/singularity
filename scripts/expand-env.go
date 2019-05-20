// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// expand-env is a simple program that reads stdin, replaces all
// occurrences of @VAR@ by the corresponding value of VAR in the current
// environment, and writes the result to stdout.
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

func main() {
	stdin, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "E: Cannot read stdin. Abort.\n")
		os.Exit(1)
	}

	env := os.Environ()

	replacements := make([]string, 0, 2*len(env))

	for _, e := range os.Environ() {
		values := strings.SplitN(e, "=", 2)

		// This should never happen, but just in case
		if len(values) == 1 {
			values = append(values, "")
		}

		replacements = append(replacements, "@"+values[0]+"@", values[1])
	}

	strings.NewReplacer(replacements...).WriteString(os.Stdout, string(stdin))
}
