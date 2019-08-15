/*
 * umoci: Umoci Modifies Open Containers' Images
 * Copyright (C) 2016, 2017, 2018 SUSE LLC.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package convert

import (
	"path/filepath"
	"runtime"
	"strings"

	"github.com/apex/log"
	igen "github.com/openSUSE/umoci/oci/config/generate"
	"github.com/openSUSE/umoci/third_party/user"
	ispec "github.com/opencontainers/image-spec/specs-go/v1"
	rspec "github.com/opencontainers/runtime-spec/specs-go"
	rgen "github.com/opencontainers/runtime-tools/generate"
	"github.com/pkg/errors"
)

// Annotations described by the OCI image-spec document (these represent fields
// in an image configuration that do not have a native representation in the
// runtime-spec).
const (
	authorAnnotation       = "org.opencontainers.image.author"
	createdAnnotation      = "org.opencontainers.image.created"
	stopSignalAnnotation   = "org.opencontainers.image.stopSignal"
	exposedPortsAnnotation = "org.opencontainers.image.exposedPorts"
)

// ToRuntimeSpec converts the given OCI image configuration to a runtime
// configuration appropriate for use, which is templated on the default
// configuration specified by the OCI runtime-tools. It is equivalent to
// MutateRuntimeSpec("runtime-tools/generate".New(), image).Spec().
func ToRuntimeSpec(rootfs string, image ispec.Image) (rspec.Spec, error) {
	g, err := rgen.New(runtime.GOOS)
	if err != nil {
		return rspec.Spec{}, err
	}
	if err := MutateRuntimeSpec(g, rootfs, image); err != nil {
		return rspec.Spec{}, err
	}
	return *g.Spec(), nil
}

// parseEnv splits a given environment variable (of the form name=value) into
// (name, value). An error is returned if there is no "=" in the line or if the
// name is empty.
func parseEnv(env string) (string, string, error) {
	parts := strings.SplitN(env, "=", 2)
	if len(parts) != 2 {
		return "", "", errors.Errorf("environment variable must contain '=': %s", env)
	}

	name, value := parts[0], parts[1]
	if name == "" {
		return "", "", errors.Errorf("environment variable must have non-empty name: %s", env)
	}
	return name, value, nil
}

// MutateRuntimeSpec mutates a given runtime specification generator with the
// image configuration provided. It returns the original generator, and does
// not modify any fields directly (to allow for chaining).
func MutateRuntimeSpec(g rgen.Generator, rootfs string, image ispec.Image) error {
	ig, err := igen.NewFromImage(image)
	if err != nil {
		return errors.Wrap(err, "creating image generator")
	}

	if ig.OS() != "linux" {
		return errors.Errorf("unsupported OS: %s", image.OS)
	}

	// FIXME: We need to figure out if we're modifying an incompatible runtime spec.
	//g.SetVersion(rspec.Version)
	// TODO: We stopped including the OS and Architecture information in the runtime-spec.
	//       Make sure we fix that once https://github.com/opencontainers/image-spec/pull/711
	//       is resolved.

	// Set verbatim fields
	g.SetProcessTerminal(true)
	g.SetRootPath(filepath.Base(rootfs))
	g.SetRootReadonly(false)

	g.SetProcessCwd("/")
	if ig.ConfigWorkingDir() != "" {
		g.SetProcessCwd(ig.ConfigWorkingDir())
	}

	for _, env := range ig.ConfigEnv() {
		name, value, err := parseEnv(env)
		if err != nil {
			return errors.Wrap(err, "parsing image.Config.Env")
		}
		g.AddProcessEnv(name, value)
	}

	args := []string{}
	args = append(args, ig.ConfigEntrypoint()...)
	args = append(args, ig.ConfigCmd()...)
	if len(args) > 0 {
		g.SetProcessArgs(args)
	}

	// Set annotations fields
	for key, value := range ig.ConfigLabels() {
		g.AddAnnotation(key, value)
	}
	g.AddAnnotation(authorAnnotation, ig.Author())
	g.AddAnnotation(createdAnnotation, ig.Created().Format(igen.ISO8601))
	g.AddAnnotation(stopSignalAnnotation, image.Config.StopSignal)

	// Set parsed fields
	// Get the *actual* uid and gid of the user. If the image doesn't contain
	// an /etc/passwd or /etc/group file then GetExecUserPath will just do a
	// numerical parsing.
	var passwdPath, groupPath string
	if rootfs != "" {
		passwdPath = filepath.Join(rootfs, "/etc/passwd")
		groupPath = filepath.Join(rootfs, "/etc/group")
	}
	execUser, err := user.GetExecUserPath(ig.ConfigUser(), nil, passwdPath, groupPath)
	if err != nil {
		// We only log an error if were not given a rootfs, and we set execUser
		// to the "default" (root:root).
		if rootfs != "" {
			return errors.Wrapf(err, "cannot parse user spec: '%s'", ig.ConfigUser())
		}
		log.Warnf("could not parse user spec '%s' without a rootfs -- defaulting to root:root", ig.ConfigUser())
		execUser = new(user.ExecUser)
	}

	g.SetProcessUID(uint32(execUser.Uid))
	g.SetProcessGID(uint32(execUser.Gid))
	g.ClearProcessAdditionalGids()

	for _, gid := range execUser.Sgids {
		g.AddProcessAdditionalGid(uint32(gid))
	}
	if execUser.Home != "" {
		g.AddProcessEnv("HOME", execUser.Home)
	}

	// Set optional fields
	ports := ig.ConfigExposedPortsArray()
	g.AddAnnotation(exposedPortsAnnotation, strings.Join(ports, ","))

	for vol := range ig.ConfigVolumes() {
		// XXX: This is _fine_ but might cause some issues in the future.
		g.AddMount(rspec.Mount{
			Destination: vol,
			Type:        "tmpfs",
			Source:      "none",
			Options:     []string{"rw", "nosuid", "nodev", "noexec", "relatime"},
		})
	}

	// Remove all seccomp rules.
	g.Spec().Linux.Seccomp = nil

	return nil
}
