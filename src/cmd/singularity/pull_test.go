// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"os"
	"os/exec"
	"testing"

	"github.com/singularityware/singularity/src/pkg/test"
)

type pullOpts struct {
	force   bool
	library string
}

func imagePull(opts pullOpts, sourceSpec string) ([]byte, error) {
	var argv []string
	argv = append(argv, "pull")
	if opts.force {
		argv = append(argv, "--force")
	}
	if opts.library != "" {
		argv = append(argv, "--library", opts.library)
	}
	argv = append(argv, sourceSpec)

	return exec.Command(cmdPath, argv...).CombinedOutput()
}

func TestPullForce(t *testing.T) {
	test.EnsurePrivilege(t)

	imagePath := "./alpine_3.7.sif"
	defer os.Remove(imagePath)
	if b, err := imagePull(pullOpts{}, "library://dtrudg/linux/alpine:3.7"); err != nil {
		t.Log(string(b))
		t.Fatalf("unexpected failure: %v", err)
	}
	imageVerify(t, imagePath, false)

	if b, err := imagePull(pullOpts{}, "library://dtrudg/linux/alpine:3.7"); err == nil {
		t.Log(string(b))
		t.Fatalf("unexpected success")
	}

	if b, err := imagePull(pullOpts{force: true}, "library://dtrudg/linux/alpine:3.7"); err != nil {
		t.Log(string(b))
		t.Fatalf("unexpected failure: %v", err)
	}
	imageVerify(t, imagePath, false)
}

func TestPullNonExistent(t *testing.T) {
	test.EnsurePrivilege(t)

	if b, err := imagePull(pullOpts{}, "library://this_should_not/exist"); err == nil {
		t.Log(string(b))
		t.Fatalf("unexpected success")
	}
}
