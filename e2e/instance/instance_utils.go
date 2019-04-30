// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package instance

import (
	"strconv"

	"github.com/sylabs/singularity/e2e/internal/e2e"
)

const instanceStartPort = 11372

type startOpts struct {
	addCaps       string
	allowSetuid   bool
	applyCgroups  string
	bind          []string
	boot          bool
	cleanenv      bool
	contain       bool
	containall    bool
	dns           string
	dockerLogin   bool
	dropCaps      string
	home          string
	hostname      string
	keepPrivs     bool
	net           bool
	network       string
	networkArgs   []string
	noHome        bool
	noPrivs       bool
	nv            bool
	overlay       []string
	scratch       []string
	security      []string
	userns        bool
	uts           bool
	workdir       string
	writable      bool
	writableTmpfs bool
}

type listOpts struct {
	json bool
	user string
}

type stopOpts struct {
	all     bool
	force   bool
	signal  string
	timeout int
	user    string
}

type instance struct {
	Instance string `json:"instance"`
	Pid      int    `json:"pid"`
	Image    string `json:"img"`
}

type instanceList struct {
	Instances []instance `json:"instances"`
}

func startInstance(opts startOpts, containerPath, instanceName string, argv ...string) (stdout string, stderr string, err error) {
	args := []string{"instance", "start"}
	if opts.addCaps != "" {
		args = append(args, "--add-caps", opts.addCaps)
	}
	if opts.allowSetuid {
		args = append(args, "--allow-setuid")
	}
	if opts.applyCgroups != "" {
		args = append(args, "--apply-cgroups", opts.applyCgroups)
	}
	for _, bind := range opts.bind {
		args = append(args, "--bind", bind)
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
	if opts.dockerLogin {
		args = append(args, "--docker-login")
	}
	if opts.dropCaps != "" {
		args = append(args, "--drop-caps", opts.dropCaps)
	}
	if opts.home != "" {
		args = append(args, "--home", opts.home)
	}
	if opts.hostname != "" {
		args = append(args, "--hostname", opts.hostname)
	}
	if opts.keepPrivs {
		args = append(args, "--keep-privs")
	}
	if opts.net {
		args = append(args, "--net")
	}
	if opts.network != "" {
		args = append(args, "--network", opts.network)
	}
	for _, networkArgs := range opts.networkArgs {
		args = append(args, "--network-args", networkArgs)
	}
	if opts.noHome {
		args = append(args, "--no-home")
	}
	if opts.noPrivs {
		args = append(args, "--no-privs")
	}
	if opts.nv {
		args = append(args, "--nv")
	}
	for _, overlay := range opts.overlay {
		args = append(args, "--overlay", overlay)
	}
	for _, scratch := range opts.scratch {
		args = append(args, "--scratch", scratch)
	}
	for _, security := range opts.security {
		args = append(args, "--security", security)
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
	if opts.writableTmpfs {
		args = append(args, "--writable-tmpfs")
	}
	args = append(args, containerPath, instanceName)
	args = append(args, argv...)
	return e2e.GenericExec(testenv.CmdPath, args...)
}

func listInstance(opts listOpts) (stdout string, stderr string, err error) {
	args := []string{"instance", "list"}
	if opts.json {
		args = append(args, "--json")
	}
	if opts.user != "" {
		args = append(args, "--user", opts.user)
	}
	return e2e.GenericExec(testenv.CmdPath, args...)
}

func stopInstance(opts stopOpts, instance string) (stdout string, stderr string, err error) {
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
	if opts.timeout != 0 {
		args = append(args, "--timeout", strconv.Itoa(opts.timeout))
	}
	if opts.user != "" {
		args = append(args, "--user", opts.user)
	}
	if instance != "" {
		args = append(args, instance)
	}
	return e2e.GenericExec(testenv.CmdPath, args...)
}

func execInstance(instance string, execCmd ...string) (stdout string, stderr string, err error) {
	args := []string{"exec", "instance://" + instance}
	args = append(args, execCmd...)
	return e2e.GenericExec(testenv.CmdPath, args...)
}
