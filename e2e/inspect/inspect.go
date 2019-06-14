// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// This test sets singularity image specific environment variables and
// verifies that they are properly set.

package singularityenv

import (
	"fmt"
	"os/exec"
	"testing"

	"github.com/buger/jsonparser"
	"github.com/sylabs/singularity/e2e/internal/e2e"
	//	"github.com/sylabs/singularity/internal/pkg/test"
)

type testingEnv struct {
	// base env for running tests
	CmdPath string `split_words:"true"`
}

var testenv testingEnv

func singularityInspect(t *testing.T) {
	argv := []string{"inspect", "--json", "--labels", "testdata/test.sif"}
	cmd := exec.Command("singularity", argv...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(out))
		panic(err)
	}

	//    fmt.Println("OUT: ", string(out))

	v, err := jsonparser.GetString(out, "attributes", "labels", "E2E")
	if err != nil {
		fmt.Println("ERROR: ", err)
	}
	fmt.Println(v)

	if v != "AWSOME" {
		t.Fatalf("Unexpected faulure: got: %s, expecting: %s", v, "AWSOME")
	}

	v, err = jsonparser.GetString(out, "attributes", "labels", "hi")
	if err != nil {
		fmt.Println("ERROR: ", err)
	}
	fmt.Println(v)

}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(t *testing.T) {
	e2e.LoadEnv(t, &testenv)

	// try to build from a non existen path
	t.Run("singularityEnv", singularityInspect)
}
