// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package config

import (
	"fmt"
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"
)

// RuntimeOciProcess describes the methods required for an OCI process implementation.
type RuntimeOciProcess interface {
	GetSpec() *specs.Process

	GetTerminal() bool
	SetTerminal(enable bool)

	GetConsoleSize() (uint, uint)
	SetConsoleSize(height uint, width uint)

	GetUID() uint32
	SetUID(uid uint32)

	GetGID() uint32
	SetGID(gid uint32)

	GetAdditionalGids() []uint32
	SetAdditionalGids(gids []uint32) error
	AddAdditionalGid(gid uint32) error
	DelAdditionalGid(gid uint32) error

	GetUsername() string
	SetUsername(name string)

	SetArgs(args []string) error
	GetArgs() []string
	AddArg(arg string) error
	DelArg(arg string) error

	SetEnv(env []string) error
	GetEnv() []string
	AddEnv(env string) error
	DelEnv(env string) error

	SetCwd(cwd string)
	GetCwd() string

	ProcessPlatform
}

var environ = map[string]int{}

// DefaultRuntimeOciProcess describes the default runtime OCI process information.
type DefaultRuntimeOciProcess struct {
	RuntimeOciSpec *RuntimeOciSpec
}

func (c *DefaultRuntimeOciProcess) init() {
	if c.RuntimeOciSpec.Process == nil {
		c.RuntimeOciSpec.Process = &specs.Process{}
	}
}

// GetSpec retrieves the runtime OCI process spec.
func (c *DefaultRuntimeOciProcess) GetSpec() *specs.Process {
	c.init()
	return c.RuntimeOciSpec.Process
}

// GetTerminal retrieves the runtime OCI process terminal flag.
func (c *DefaultRuntimeOciProcess) GetTerminal() bool {
	c.init()
	return c.RuntimeOciSpec.Process.Terminal
}

// SetTerminal sets the runtime OCI process terminal flag.
func (c *DefaultRuntimeOciProcess) SetTerminal(enable bool) {
	c.init()
	c.RuntimeOciSpec.Process.Terminal = enable
}

// GetConsoleSize retrieves the runtime OCI process console size.
func (c *DefaultRuntimeOciProcess) GetConsoleSize() (uint, uint) {
	c.init()
	if c.RuntimeOciSpec.Process.ConsoleSize == nil {
		c.RuntimeOciSpec.Process.ConsoleSize = &specs.Box{}
	}
	return c.RuntimeOciSpec.Process.ConsoleSize.Height, c.RuntimeOciSpec.Process.ConsoleSize.Width
}

// SetConsoleSize sets the runtime OCI process console size.
func (c *DefaultRuntimeOciProcess) SetConsoleSize(height uint, width uint) {
	c.init()
	if c.RuntimeOciSpec.Process.ConsoleSize == nil {
		c.RuntimeOciSpec.Process.ConsoleSize = &specs.Box{}
	}
	c.RuntimeOciSpec.Process.ConsoleSize.Height = height
	c.RuntimeOciSpec.Process.ConsoleSize.Width = width
}

// GetUID retrieves the runtime OCI process UID.
func (c *DefaultRuntimeOciProcess) GetUID() uint32 {
	c.init()
	return c.RuntimeOciSpec.Process.User.UID
}

// SetUID sets the runtime OCI process UID.
func (c *DefaultRuntimeOciProcess) SetUID(uid uint32) {
	c.init()
	c.RuntimeOciSpec.Process.User.UID = uid
}

// GetGID retrieves the runtime OCI process GID.
func (c *DefaultRuntimeOciProcess) GetGID() uint32 {
	c.init()
	return c.RuntimeOciSpec.Process.User.GID
}

// SetGID sets the runtime OCI process GID.
func (c *DefaultRuntimeOciProcess) SetGID(gid uint32) {
	c.init()
	c.RuntimeOciSpec.Process.User.GID = gid
}

// GetAdditionalGids retrieves the runtime OCI additional GIDs.
func (c *DefaultRuntimeOciProcess) GetAdditionalGids() []uint32 {
	c.init()
	return c.RuntimeOciSpec.Process.User.AdditionalGids
}

// SetAdditionalGids sets the runtime OCI additional GIDs.
func (c *DefaultRuntimeOciProcess) SetAdditionalGids(gids []uint32) error {
	c.init()
	return nil
}

// AddAdditionalGid add a GID to the runtime OCI process.
func (c *DefaultRuntimeOciProcess) AddAdditionalGid(gid uint32) error {
	c.init()
	return nil
}

// DelAdditionalGid deletes a GID to the runtime OCI process.
func (c *DefaultRuntimeOciProcess) DelAdditionalGid(gid uint32) error {
	c.init()
	return nil
}

// GetUsername gets the username associated with the runtime OCI process.
func (c *DefaultRuntimeOciProcess) GetUsername() string {
	c.init()
	return c.RuntimeOciSpec.Process.User.Username
}

// SetUsername sets the username associated with the runtime OCI process.
func (c *DefaultRuntimeOciProcess) SetUsername(name string) {
	c.init()
}

// SetArgs sets the arguments associated with the runtime OCI process.
func (c *DefaultRuntimeOciProcess) SetArgs(args []string) error {
	c.init()
	for _, arg := range args {
		c.AddArg(arg)
	}
	return nil
}

// GetArgs gets the arguments associated with the runtime OCI process.
func (c *DefaultRuntimeOciProcess) GetArgs() []string {
	c.init()
	return c.RuntimeOciSpec.Process.Args
}

// AddArg adds an argument to the runtime OCI process.
func (c *DefaultRuntimeOciProcess) AddArg(arg string) error {
	c.init()
	c.RuntimeOciSpec.Process.Args = append(c.RuntimeOciSpec.Process.Args, arg)
	return nil
}

// DelArg deletes an argument from the runtime OCI process.
func (c *DefaultRuntimeOciProcess) DelArg(arg string) error {
	c.init()
	return nil
}

// SetEnv sets an environment variable in the runtime OCI process.
func (c *DefaultRuntimeOciProcess) SetEnv(env []string) error {
	c.init()
	for _, e := range env {
		if err := c.AddEnv(e); err != nil {
			return err
		}
	}
	return nil
}

// GetEnv gets an environment variable from the runtime OCI process.
func (c *DefaultRuntimeOciProcess) GetEnv() []string {
	c.init()
	return c.RuntimeOciSpec.Process.Env
}

// AddEnv adds an environment variable to the runtime OCI process.
func (c *DefaultRuntimeOciProcess) AddEnv(env string) error {
	c.init()
	if i := strings.IndexByte(env, '='); i != -1 {
		if _, present := environ[env[:i]]; present {
			return fmt.Errorf("environment variable %s already exists", env[:i])
		}
		environ[env[:i]] = len(c.RuntimeOciSpec.Process.Env)
	} else {
		return fmt.Errorf("bad formatted environment variable: %s", env)
	}
	c.RuntimeOciSpec.Process.Env = append(c.RuntimeOciSpec.Process.Env, env)
	return nil
}

// DelEnv deletes an environment variable from the runtime OCI process.
func (c *DefaultRuntimeOciProcess) DelEnv(env string) error {
	c.init()
	if i := strings.IndexByte(env, '='); i != -1 {
		if idx, present := environ[env[:i]]; present {
			c.RuntimeOciSpec.Process.Env = append(c.RuntimeOciSpec.Process.Env[:idx], c.RuntimeOciSpec.Process.Env[idx+1:]...)
			delete(environ, env[:i])
			return nil
		}
		return fmt.Errorf("environment variable %s doesn't exists", env[:i])
	}
	return fmt.Errorf("bad formatted environment variable: %s", env)
}

// SetCwd sets the current working directory of the runtime OCI process.
func (c *DefaultRuntimeOciProcess) SetCwd(cwd string) {
	c.init()
	c.RuntimeOciSpec.Process.Cwd = cwd
}

// GetCwd gets the current working directory of the runtime OCI process.
func (c *DefaultRuntimeOciProcess) GetCwd() string {
	c.init()
	return c.RuntimeOciSpec.Process.Cwd
}
