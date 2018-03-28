/*
Copyright (c) 2018, Sylabs, Inc. All rights reserved.
This software is licensed under a 3-clause BSD license.  Please
consult LICENSE file distributed with the sources of this project regarding
your rights to use or distribute this software.
*/

package main

import (
	//	"github.com/singularityware/singularity/internal/pkg/cli"
	//	"github.com/singularityware/singularity/pkg/build"
	"github.com/singularityware/singularity/pkg/signing"
)

func main() {
	signing.Sign([]byte("6d02c8a7d6b41cb821e23d7ca345da7da907e6325e303e4741d773080c075ebe96405432087f6c74682d73c479dc393a"))
}
