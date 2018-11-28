// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"os"
	"os/exec"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

type startOpts struct {
	add_caps       string
	allow_setuid   bool
	apply_cgroups  string
	bind           string
	boot           bool
	cleanenv       bool
	contain        bool
	containall     bool
	dns            string
	drop_caps      string
	home           string
	hostname       string
	keep_privs     bool
	net            bool
	network        string
	network_args   string
	no_home        bool
	no_privs       bool
	nv             bool
	overlay        string
	scratch        string
	security       string
	userns         bool
	uts            bool
	workdir        string
	writable       bool
	writable_tmpfs bool
}

type listOpts struct {
	json      bool
	user      string
	container string
}

type stopOpts struct {
	all     bool
	force   bool
	signal  string
	timeout string
	user    string
}

func startInstance(image string, instance string, opts startOpts) ([]byte, error) {
	args := []string{"instance", "start"}
	if opts.add_caps != "" {
		args = append(args, "--add-caps", opts.add_caps)
	}
	if opts.allow_setuid {
		args = append(args, "--allow-setuid")
	}
	if opts.apply_cgroups != "" {
		args = append(args, "--apply-cgroups", opts.apply_cgroups)
	}
	if opts.bind != "" {
		args = append(args, "--bind", opts.bind)
	}
	if opts.boot {
		args = append(args, "--boot")
	}
	if opts.cleanenv {
		args = append(args, "--cleanenv")
	}
	if opts.contain {
		args = append(args, "--contain")
	}
	if opts.containall {
		args = append(args, "--containall")
	}
	if opts.dns != "" {
		args = append(args, "--dns", opts.dns)
	}
	if opts.drop_caps != "" {
		args = append(args, "--drop-caps", opts.drop_caps)
	}
	if opts.home != "" {
		args = append(args, "--home", opts.home)
	}
	if opts.hostname != "" {
		args = append(args, "--hostname", opts.hostname)
	}
	if opts.keep_privs {
		args = append(args, "--keep-privs")
	}
	if opts.net {
		args = append(args, "--net")
	}
	if opts.network != "" {
		args = append(args, "--network", opts.network)
	}
	if opts.network_args != "" {
		args = append(args, "--network-args", opts.network_args)
	}
	if opts.no_home {
		args = append(args, "--no-home")
	}
	if opts.no_privs {
		args = append(args, "--no-privs")
	}
	if opts.nv {
		args = append(args, "--nv")
	}
	if opts.overlay != "" {
		args = append(args, "--overlay", opts.overlay)
	}
	if opts.scratch != "" {
		args = append(args, "--scratch", opts.scratch)
	}
	if opts.security != "" {
		args = append(args, "--security", opts.security)
	}
	if opts.userns {
		args = append(args, "--userns")
	}
	if opts.uts {
		args = append(args, "--uts")
	}
	if opts.workdir != "" {
		args = append(args, "--workdir", opts.workdir)
	}
	if opts.writable {
		args = append(args, "--writable")
	}
	if opts.writable_tmpfs {
		args = append(args, "--writable-tmpfs")
	}
	args = append(args, image, instance)
	cmd := exec.Command(cmdPath, args...)
	return cmd.CombinedOutput()
}

func listInstance(opts listOpts) ([]byte, error) {
	args := []string{"instance", "list"}
	if opts.json {
		args = append(args, "--json")
	}
	if opts.user != "" {
		args = append(args, "--user", opts.user)
	}
	if opts.container != "" {
		args = append(args, opts.container)
	}
	cmd := exec.Command(cmdPath, args...)
	return cmd.CombinedOutput()
}

func stopInstance(instance string, opts stopOpts) ([]byte, error) {
	args := []string{"instance", "stop"}
	if opts.all {
		args = append(args, "--all")
	}
	if opts.force {
		args = append(args, "--force")
	}
	if opts.signal != "" {
		args = append(args, "--signal", opts.signal)
	}
	if opts.timeout != "" {
		args = append(args, "--timeout", opts.timeout)
	}
	if opts.user != "" {
		args = append(args, "--user", opts.user)
	}
	args = append(args, instance)
	cmd := exec.Command(cmdPath, args...)
	return cmd.CombinedOutput()
}

// TestInstance tests singularity instance cmd
// start, list, stop
func TestInstance(t *testing.T) {
	var definition = "../../examples/busybox/Singularity"
	var imagePath = "./instance_tests.sif"

	opts := buildOpts{
		force:   true,
		sandbox: false,
	}
	if b, err := imageBuild(opts, imagePath, definition); err != nil {
		t.Log(string(b))
		t.Fatalf("unexpected failure: %v", err)
	}
	imageVerify(t, imagePath, true)
	defer os.RemoveAll(imagePath)

	t.Run("StartListStop", test.WithoutPrivilege(func(t *testing.T) {
		var defaultInstance = "www"

		startInstanceOutput, err := startInstance(imagePath, defaultInstance, startOpts{})
		if err != nil {
			t.Fatalf("Error starting instance from an image: %v. Output follows.\n%s", err, string(startInstanceOutput))
		}

		listInstanceOutput, err := listInstance(listOpts{})
		if err != nil {
			t.Fatalf("Error listing instances: %v. Output follows.\n%s", err, string(listInstanceOutput))
		}

		stopInstanceOutput, err := stopInstance(defaultInstance, stopOpts{})
		if err != nil {
			t.Fatalf("Error stopping instance by name: %v. Output follows.\n%s", err, string(stopInstanceOutput))
		}
	}))
}
