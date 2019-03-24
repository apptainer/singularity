// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package useragent

import (
	"regexp"
	"testing"
)

func TestSingularityVersion(t *testing.T) {
	InitValue("singularity", "3.0.0-alpha.1-303-gaed8d30-dirty")

	re := regexp.MustCompile(`Singularity/[[:digit:]]+(.[[:digit:]]+){2} \(Linux [[:alnum:]]+\) Go/[[:digit:]]+(.[[:digit:]]+){1,2}`)
	if !re.MatchString(Value()) {
		t.Fatalf("user agent did not match regexp")
	}
}
