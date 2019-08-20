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

package generate

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/opencontainers/go-digest"
	ispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// FIXME: Because we are not a part of upstream, we have to add some tests that
//        ensure that this set of getters and setters is complete. This should
//        be possible through some reflection.

// Generator allows you to generate a mutable OCI image-spec configuration
// which can be written to a file (and its digest computed). It is the
// recommended way of handling modification and generation of image-spec
// configuration blobs.
type Generator struct {
	image ispec.Image
}

// init makes sure everything has a "proper" zero value.
func (g *Generator) init() {
	if g.image.Config.ExposedPorts == nil {
		g.ClearConfigExposedPorts()
	}
	if g.image.Config.Env == nil {
		g.ClearConfigEnv()
	}
	if g.image.Config.Entrypoint == nil {
		g.ClearConfigEntrypoint()
	}
	if g.image.Config.Cmd == nil {
		g.ClearConfigCmd()
	}
	if g.image.Config.Volumes == nil {
		g.ClearConfigVolumes()
	}
	if g.image.Config.Labels == nil {
		g.ClearConfigLabels()
	}
	if g.image.RootFS.DiffIDs == nil {
		g.ClearRootfsDiffIDs()
	}
	if g.image.History == nil {
		g.ClearHistory()
	}
}

// New creates a new Generator with the inital template set to a default. It is
// not recommended to leave any of the options as their default values (they
// may change in the future without warning and may be invalid images).
func New() *Generator {
	// FIXME: Come up with some sane default.
	g := &Generator{
		image: ispec.Image{},
	}
	g.init()
	return g
}

// NewFromImage generates a new generator with the initial template being the
// given ispec.Image.
func NewFromImage(image ispec.Image) (*Generator, error) {
	g := &Generator{
		image: image,
	}

	g.init()
	return g, nil
}

// Image returns a copy of the current state of the generated image.
func (g *Generator) Image() ispec.Image {
	return g.image
}

// SetConfigUser sets the username or UID which the process in the container should run as.
func (g *Generator) SetConfigUser(user string) {
	g.image.Config.User = user
}

// ConfigUser returns the username or UID which the process in the container should run as.
func (g *Generator) ConfigUser() string {
	return g.image.Config.User
}

// ClearConfigExposedPorts clears the set of ports to expose from a container running this image.
func (g *Generator) ClearConfigExposedPorts() {
	g.image.Config.ExposedPorts = map[string]struct{}{}
}

// AddConfigExposedPort adds a port the set of ports to expose from a container running this image.
func (g *Generator) AddConfigExposedPort(port string) {
	g.image.Config.ExposedPorts[port] = struct{}{}
}

// RemoveConfigExposedPort removes a port the set of ports to expose from a container running this image.
func (g *Generator) RemoveConfigExposedPort(port string) {
	delete(g.image.Config.ExposedPorts, port)
}

// ConfigExposedPorts returns the set of ports to expose from a container running this image.
func (g *Generator) ConfigExposedPorts() map[string]struct{} {
	// We have to make a copy to preserve the privacy of g.image.Config.
	copy := map[string]struct{}{}
	for k, v := range g.image.Config.ExposedPorts {
		copy[k] = v
	}
	return copy
}

// ConfigExposedPortsArray returns a sorted array of ports to expose from a container running this image.
func (g *Generator) ConfigExposedPortsArray() []string {
	var ports []string
	for port := range g.image.Config.ExposedPorts {
		ports = append(ports, port)
	}
	sort.Strings(ports)
	return ports
}

// ClearConfigEnv clears the list of environment variables to be used in a container.
func (g *Generator) ClearConfigEnv() {
	g.image.Config.Env = []string{}
}

// AddConfigEnv appends to the list of environment variables to be used in a container.
func (g *Generator) AddConfigEnv(name, value string) {
	// If the key already exists in the environment set, we replace it.
	// This ensures we don't run into POSIX undefined territory.
	env := fmt.Sprintf("%s=%s", name, value)
	for idx := range g.image.Config.Env {
		if strings.HasPrefix(g.image.Config.Env[idx], name+"=") {
			g.image.Config.Env[idx] = env
			return
		}
	}
	g.image.Config.Env = append(g.image.Config.Env, env)
}

// ConfigEnv returns the list of environment variables to be used in a container.
func (g *Generator) ConfigEnv() []string {
	copy := []string{}
	for _, v := range g.image.Config.Env {
		copy = append(copy, v)
	}
	return copy
}

// ClearConfigEntrypoint clears the list of arguments to use as the command to execute when the container starts.
func (g *Generator) ClearConfigEntrypoint() {
	g.image.Config.Entrypoint = []string{}
}

// SetConfigEntrypoint sets the list of arguments to use as the command to execute when the container starts.
func (g *Generator) SetConfigEntrypoint(entrypoint []string) {
	copy := []string{}
	for _, v := range entrypoint {
		copy = append(copy, v)
	}
	g.image.Config.Entrypoint = copy
}

// ConfigEntrypoint returns the list of arguments to use as the command to execute when the container starts.
func (g *Generator) ConfigEntrypoint() []string {
	// We have to make a copy to preserve the privacy of g.image.Config.
	copy := []string{}
	for _, v := range g.image.Config.Entrypoint {
		copy = append(copy, v)
	}
	return copy
}

// ClearConfigCmd clears the list of default arguments to the entrypoint of the container.
func (g *Generator) ClearConfigCmd() {
	g.image.Config.Cmd = []string{}
}

// SetConfigCmd sets the list of default arguments to the entrypoint of the container.
func (g *Generator) SetConfigCmd(cmd []string) {
	copy := []string{}
	for _, v := range cmd {
		copy = append(copy, v)
	}
	g.image.Config.Cmd = copy
}

// ConfigCmd returns the list of default arguments to the entrypoint of the container.
func (g *Generator) ConfigCmd() []string {
	// We have to make a copy to preserve the privacy of g.image.Config.
	copy := []string{}
	for _, v := range g.image.Config.Cmd {
		copy = append(copy, v)
	}
	return copy
}

// ClearConfigVolumes clears the set of directories which should be created as data volumes in a container running this image.
func (g *Generator) ClearConfigVolumes() {
	g.image.Config.Volumes = map[string]struct{}{}
}

// AddConfigVolume adds a volume to the set of directories which should be created as data volumes in a container running this image.
func (g *Generator) AddConfigVolume(volume string) {
	g.image.Config.Volumes[volume] = struct{}{}
}

// RemoveConfigVolume removes a volume from the set of directories which should be created as data volumes in a container running this image.
func (g *Generator) RemoveConfigVolume(volume string) {
	delete(g.image.Config.Volumes, volume)
}

// ConfigVolumes returns the set of directories which should be created as data volumes in a container running this image.
func (g *Generator) ConfigVolumes() map[string]struct{} {
	// We have to make a copy to preserve the privacy of g.image.Config.
	copy := map[string]struct{}{}
	for k, v := range g.image.Config.Volumes {
		copy[k] = v
	}
	return copy
}

// ClearConfigLabels clears the set of arbitrary metadata for the container.
func (g *Generator) ClearConfigLabels() {
	g.image.Config.Labels = map[string]string{}
}

// AddConfigLabel adds a label to the set of arbitrary metadata for the container.
func (g *Generator) AddConfigLabel(label, value string) {
	g.image.Config.Labels[label] = value
}

// RemoveConfigLabel removes a label from the set of arbitrary metadata for the container.
func (g *Generator) RemoveConfigLabel(label string) {
	delete(g.image.Config.Labels, label)
}

// ConfigLabels returns the set of arbitrary metadata for the container.
func (g *Generator) ConfigLabels() map[string]string {
	// We have to make a copy to preserve the privacy of g.image.Config.
	copy := map[string]string{}
	for k, v := range g.image.Config.Labels {
		copy[k] = v
	}
	return copy
}

// SetConfigWorkingDir sets the current working directory of the entrypoint process in the container.
func (g *Generator) SetConfigWorkingDir(workingDir string) {
	g.image.Config.WorkingDir = workingDir
}

// ConfigWorkingDir returns the current working directory of the entrypoint process in the container.
func (g *Generator) ConfigWorkingDir() string {
	return g.image.Config.WorkingDir
}

// SetConfigStopSignal sets the system call signal that will be sent to the container to exit.
func (g *Generator) SetConfigStopSignal(stopSignal string) {
	g.image.Config.StopSignal = stopSignal
}

// ConfigStopSignal returns the system call signal that will be sent to the container to exit.
func (g *Generator) ConfigStopSignal() string {
	return g.image.Config.StopSignal
}

// SetRootfsType sets the type of the rootfs.
func (g *Generator) SetRootfsType(rootfsType string) {
	g.image.RootFS.Type = rootfsType
}

// RootfsType returns the type of the rootfs.
func (g *Generator) RootfsType() string {
	return g.image.RootFS.Type
}

// ClearRootfsDiffIDs clears the array of layer content hashes (DiffIDs), in order from bottom-most to top-most.
func (g *Generator) ClearRootfsDiffIDs() {
	g.image.RootFS.DiffIDs = []digest.Digest{}
}

// AddRootfsDiffID appends to the array of layer content hashes (DiffIDs), in order from bottom-most to top-most.
func (g *Generator) AddRootfsDiffID(diffid digest.Digest) {
	g.image.RootFS.DiffIDs = append(g.image.RootFS.DiffIDs, diffid)
}

// RootfsDiffIDs returns the the array of layer content hashes (DiffIDs), in order from bottom-most to top-most.
func (g *Generator) RootfsDiffIDs() []digest.Digest {
	copy := []digest.Digest{}
	for _, v := range g.image.RootFS.DiffIDs {
		copy = append(copy, v)
	}
	return copy
}

// ClearHistory clears the history of each layer.
func (g *Generator) ClearHistory() {
	g.image.History = []ispec.History{}
}

// AddHistory appends to the history of the layers.
func (g *Generator) AddHistory(history ispec.History) {
	g.image.History = append(g.image.History, history)
}

// History returns the history of each layer.
func (g *Generator) History() []ispec.History {
	copy := []ispec.History{}
	for _, v := range g.image.History {
		copy = append(copy, v)
	}
	return copy
}

// ISO8601 represents the format of an ISO-8601 time string, which is identical
// to Go's RFC3339 specification.
const ISO8601 = time.RFC3339Nano

// SetCreated sets the combined date and time at which the image was created.
func (g *Generator) SetCreated(created time.Time) {
	g.image.Created = &created
}

// Created gets the combined date and time at which the image was created.
func (g *Generator) Created() time.Time {
	if g.image.Created == nil {
		// TODO: Maybe we should be returning pointers?
		return time.Time{}
	}
	return *g.image.Created
}

// SetAuthor sets the name and/or email address of the person or entity which created and is responsible for maintaining the image.
func (g *Generator) SetAuthor(author string) {
	g.image.Author = author
}

// Author returns the name and/or email address of the person or entity which created and is responsible for maintaining the image.
func (g *Generator) Author() string {
	return g.image.Author
}

// SetArchitecture is the CPU architecture which the binaries in this image are built to run on.
func (g *Generator) SetArchitecture(arch string) {
	g.image.Architecture = arch
}

// Architecture returns the CPU architecture which the binaries in this image are built to run on.
func (g *Generator) Architecture() string {
	return g.image.Architecture
}

// SetOS sets the name of the operating system which the image is built to run on.
func (g *Generator) SetOS(os string) {
	g.image.OS = os
}

// OS returns the name of the operating system which the image is built to run on.
func (g *Generator) OS() string {
	return g.image.OS
}
