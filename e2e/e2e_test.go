// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import (
	"io/ioutil"
	"log"
	"os"
	"testing"

	// This import will execute a CGO section with the help of a C
	// constructor section "init". As we always require to run e2e
	// tests as root, the C part is responsible of finding the original
	// user who executes tests; it will also create a dedicated pid
	// and mount namespace for e2e tests, and will finally restore
	// identity to the original user but will retain privileges for
	// Privileged method enabling the execution of a function with root
	// privileges when required
	_ "github.com/sylabs/singularity/e2e/internal/e2e/init"
)

func TestE2E(t *testing.T) {
	targetCoverageFilePath := os.Getenv("SINGULARITY_E2E_COVERAGE")
	if targetCoverageFilePath != "" {
		logFile, err := os.OpenFile(targetCoverageFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("failed to create log file: %s", err)
		}
		defer logFile.Close()
		log.SetOutput(logFile)
		log.Println("List of commands called by E2E")
	} else {
		log.SetOutput(ioutil.Discard)
	}

	RunE2ETests(t)
}
