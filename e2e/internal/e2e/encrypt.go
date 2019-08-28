// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/util/bin"
	"github.com/sylabs/singularity/pkg/util/crypt"
)

const (
	// Passphrase used for passphrase-based encryption tests
	Passphrase = "e2e-passphrase"
)

// CheckCryptsetupVersion checks the version of cryptsetup and returns
// an error if the version is not compatible; nil otherwise
func CheckCryptsetupVersion() error {
	cryptsetup, err := bin.Cryptsetup()
	if err != nil {
		return err
	}

	cmd := exec.Command(cryptsetup, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run cryptsetup --version: %s", err)
	}

	if !strings.Contains(string(out), "cryptsetup 2.") {
		return fmt.Errorf("incompatible cryptsetup version")
	}

	return nil
}

// GeneratePemFiles creates a new PEM file for testing purposes.
func GeneratePemFiles(t *testing.T, basedir string) (string, string) {
	// Temporary file to save the PEM public file. The caller is in charge of cleanup
	tempPemPubFile, err := ioutil.TempFile(basedir, "pem-pub-")
	if err != nil {
		t.Fatalf("failed to create temporary file: %s", err)
	}
	tempPemPubFile.Close()

	// Temporary file to save the PEM file. The caller is in charge of cleanup
	tempPemPrivFile, err := ioutil.TempFile(basedir, "pem-priv-")
	if err != nil {
		t.Fatalf("failed to create temporary file: %s", err)
	}
	tempPemPrivFile.Close()

	rsaKey, err := crypt.GenerateRSAKey(2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %s", err)
	}

	err = crypt.SavePublicPEM(tempPemPubFile.Name(), rsaKey)
	if err != nil {
		t.Fatalf("failed to generate PEM public file: %s", err)
	}

	err = crypt.SavePrivatePEM(tempPemPrivFile.Name(), rsaKey)
	if err != nil {
		t.Fatalf("failed to generate PEM private file: %s", err)
	}

	return tempPemPubFile.Name(), tempPemPrivFile.Name()
}
