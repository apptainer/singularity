// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package generate

import (
	"io"
	"io/ioutil"
	"reflect"
	"sync"
	"testing"

	"github.com/sylabs/singularity/pkg/util/capabilities"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity/internal/pkg/test"
)

func TestGenerate(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	g := New(nil)
	config := g.Config

	args := []string{"arg1", "arg2", "arg3"}
	g.SetProcessArgs(args)
	if !reflect.DeepEqual(args, config.Process.Args) {
		t.Fatalf("OCI process arguments are not identical")
	}

	prof := "test"
	g.SetProcessApparmorProfile(prof)
	if config.Process.ApparmorProfile != prof {
		t.Fatalf("wrong OCI app armor process: %s instead of %s", config.Process.ApparmorProfile, prof)
	}

	cwd := "/"
	g.SetProcessCwd(cwd)
	if config.Process.Cwd != cwd {
		t.Fatalf("wrong OCI process cwd: %s instead of %s", config.Process.Cwd, prof)
	}

	terminal := true
	g.SetProcessTerminal(terminal)
	if config.Process.Terminal != terminal {
		t.Fatalf("wrong OCI process terminal: %v instead of %v", config.Process.Terminal, terminal)
	}

	noNewPriv := true
	g.SetProcessNoNewPrivileges(noNewPriv)
	if config.Process.NoNewPrivileges != noNewPriv {
		t.Fatalf("wrong OCI process no new privs: %v instead of %v", config.Process.NoNewPrivileges, noNewPriv)
	}

	selinux := "test"
	g.SetProcessSelinuxLabel(selinux)
	if config.Process.SelinuxLabel != selinux {
		t.Fatalf("wrong OCI process selinux label: %v instead of %v", config.Process.SelinuxLabel, selinux)
	}

	root := "/tmp"
	g.SetRootPath(root)
	if config.Root.Path != root {
		t.Fatalf("wrong OCI root path: %s instead of %v", config.Root.Path, root)
	}

	g.SetupPrivileged(false)
	if config.Process.SelinuxLabel != selinux {
		t.Fatalf("wrong OCI process selinux label: %v instead of %v", config.Process.SelinuxLabel, selinux)
	}

	g.SetupPrivileged(true)
	if config.Process.SelinuxLabel != "" {
		t.Fatalf("wrong OCI process selinux label: %v instead of empty", config.Process.SelinuxLabel)
	}
	if len(g.Config.Process.Capabilities.Bounding) != len(capabilities.Map) {
		t.Fatalf("wrong OCI capabilities while privileged")
	}
	if config.Process.SelinuxLabel != "" || config.Process.ApparmorProfile != "" || config.Linux.Seccomp != nil {
		t.Fatalf("wrong OCI privileged configuration")
	}

	g.AddLinuxUIDMapping(1000, 0, 1)
	g.AddLinuxUIDMapping(1001, 1, 1)
	if len(config.Linux.UIDMappings) != 2 {
		t.Fatalf("wrong OCI uid mapping size: %d instead of 2", len(config.Linux.UIDMappings))
	}
	mapping := config.Linux.UIDMappings[1]
	if mapping.HostID != 1001 || mapping.ContainerID != 1 || mapping.Size != 1 {
		t.Fatalf("wrong OCI uid mapping: %v", mapping)
	}

	g.AddLinuxGIDMapping(1000, 0, 1)
	g.AddLinuxGIDMapping(1001, 1, 1)
	if len(config.Linux.GIDMappings) != 2 {
		t.Fatalf("wrong OCI gid mapping size: %d instead of 2", len(config.Linux.GIDMappings))
	}
	mapping = config.Linux.GIDMappings[1]
	if mapping.HostID != 1001 || mapping.ContainerID != 1 || mapping.Size != 1 {
		t.Fatalf("wrong OCI uid mapping: %v", mapping)
	}

	mnt := specs.Mount{
		Source:      "/etc2",
		Destination: "/etc",
		Type:        "bind",
	}
	g.AddMount(mnt)
	g.AddMount(mnt)
	if len(config.Mounts) != 2 {
		t.Fatalf("wrong OCI mount size: %d instead of 2", len(config.Mounts))
	}
	mount := config.Mounts[0]
	if mount.Destination != mnt.Destination || mount.Source != mnt.Source || mount.Type != mnt.Type {
		t.Fatalf("wrong OCI mount entry: %v", mount)
	}

	g.AddProcessEnv("FOO", "bar")
	if len(config.Process.Env) != 1 {
		t.Fatalf("wrong OCI process environment size: %d instead of 1", len(config.Process.Env))
	} else if config.Process.Env[0] != "FOO=bar" {
		t.Fatalf("wrong OCI process environment FOO value: %v instead of FOO=bar", config.Process.Env[0])
	}

	g.AddProcessEnv("FOO", "foo")
	if len(config.Process.Env) != 1 {
		t.Fatalf("wrong OCI process environment size: %d instead of 1", len(config.Process.Env))
	} else if config.Process.Env[0] != "FOO=foo" {
		t.Fatalf("wrong OCI process environment FOO value: %v instead of FOO=foo", config.Process.Env[0])
	}

	g.AddProcessEnv("FOO2", "bar2")
	if len(config.Process.Env) != 2 {
		t.Fatalf("wrong OCI process environment size: %d instead of 2", len(config.Process.Env))
	}

	g.RemoveProcessEnv("FOO2")
	if len(config.Process.Env) != 1 {
		t.Fatalf("wrong OCI process environment size: %d instead of 1", len(config.Process.Env))
	}

	g.AddOrReplaceLinuxNamespace("bad", "")
	if len(config.Linux.Namespaces) != 0 {
		t.Fatalf("wrong OCI process namespace size: %d instead of 0", len(config.Linux.Namespaces))
	}
	g.AddOrReplaceLinuxNamespace(specs.PIDNamespace, "")
	if len(config.Linux.Namespaces) != 1 {
		t.Fatalf("wrong OCI process namespace size: %d instead of 1", len(config.Linux.Namespaces))
	} else if config.Linux.Namespaces[0].Type != specs.PIDNamespace || config.Linux.Namespaces[0].Path != "" {
		t.Fatalf("wrong OCI process pid namespace entry: %v", config.Linux.Namespaces[0])
	}
	selfPid := "/proc/self/ns/pid"
	g.AddOrReplaceLinuxNamespace(specs.PIDNamespace, selfPid)
	if config.Linux.Namespaces[0].Type != specs.PIDNamespace || config.Linux.Namespaces[0].Path != selfPid {
		t.Fatalf("wrong OCI process pid namespace entry: %v", config.Linux.Namespaces[0])
	}
	g.AddOrReplaceLinuxNamespace(specs.UserNamespace, "")
	if len(config.Linux.Namespaces) != 2 {
		t.Fatalf("wrong OCI process namespace size: %d instead of 2", len(config.Linux.Namespaces))
	}

	g.AddProcessRlimits("A_LIMIT", 1024, 128)
	if len(config.Process.Rlimits) != 1 {
		t.Fatalf("wrong OCI process rlimit size: %d instead of 1", len(config.Process.Rlimits))
	}
	g.AddProcessRlimits("A_SEC_LIMIT", 2048, 1024)
	if len(config.Process.Rlimits) != 2 {
		t.Fatalf("wrong OCI process rlimit size: %d instead of 2", len(config.Process.Rlimits))
	}
	rlimit := config.Process.Rlimits[1]
	if rlimit.Type != "A_SEC_LIMIT" || rlimit.Hard != 2048 || rlimit.Soft != 1024 {
		t.Fatalf("wrong OCI process rlimit entry: %v", rlimit)
	}
}

var ociJSON = `{
	"ociVersion": "` + specs.Version + `",
	"root": {
		"path": "/"
	}
}`

func TestSave(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	var wg sync.WaitGroup

	g := New(nil)
	g.SetRootPath("/")
	g.Config.Linux = &specs.Linux{}

	r, w := io.Pipe()

	wg.Add(1)

	go func() {
		defer r.Close()
		defer wg.Done()

		d, err := ioutil.ReadAll(r)
		if err != nil {
			t.Fatalf("while reading pipe: %s", err)
		}
		content := string(d)
		if content != ociJSON {
			t.Errorf("bad OCI JSON output")
		}
	}()

	g.Save(w)
	w.Close()
	wg.Wait()

	path := "/a/fake/file"
	err := g.SaveToFile(path)
	if err == nil {
		t.Fatalf("unexpected success while writing OCI config to %s", path)
	}
}
