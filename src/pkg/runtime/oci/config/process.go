package config

import (
	"fmt"
	"github.com/opencontainers/runtime-spec/specs-go"
	"strings"
)

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

type DefaultRuntimeOciProcess struct {
	RuntimeOciSpec *RuntimeOciSpec
}

func (c *DefaultRuntimeOciProcess) init() {
	if c.RuntimeOciSpec.Process == nil {
		c.RuntimeOciSpec.Process = &specs.Process{}
	}
}

func (c *DefaultRuntimeOciProcess) GetSpec() *specs.Process {
	c.init()
	return c.RuntimeOciSpec.Process
}

func (c *DefaultRuntimeOciProcess) GetTerminal() bool {
	c.init()
	return c.RuntimeOciSpec.Process.Terminal
}

func (c *DefaultRuntimeOciProcess) SetTerminal(enable bool) {
	c.init()
	c.RuntimeOciSpec.Process.Terminal = enable
}

func (c *DefaultRuntimeOciProcess) GetConsoleSize() (uint, uint) {
	c.init()
	if c.RuntimeOciSpec.Process.ConsoleSize == nil {
		c.RuntimeOciSpec.Process.ConsoleSize = &specs.Box{}
	}
	return c.RuntimeOciSpec.Process.ConsoleSize.Height, c.RuntimeOciSpec.Process.ConsoleSize.Width
}

func (c *DefaultRuntimeOciProcess) SetConsoleSize(height uint, width uint) {
	c.init()
	if c.RuntimeOciSpec.Process.ConsoleSize == nil {
		c.RuntimeOciSpec.Process.ConsoleSize = &specs.Box{}
	}
	c.RuntimeOciSpec.Process.ConsoleSize.Height = height
	c.RuntimeOciSpec.Process.ConsoleSize.Width = width
}

func (c *DefaultRuntimeOciProcess) GetUID() uint32 {
	c.init()
	return c.RuntimeOciSpec.Process.User.UID
}

func (c *DefaultRuntimeOciProcess) SetUID(uid uint32) {
	c.init()
	c.RuntimeOciSpec.Process.User.UID = uid
}

func (c *DefaultRuntimeOciProcess) GetGID() uint32 {
	c.init()
	return c.RuntimeOciSpec.Process.User.GID
}

func (c *DefaultRuntimeOciProcess) SetGID(gid uint32) {
	c.init()
	c.RuntimeOciSpec.Process.User.GID = gid
}

func (c *DefaultRuntimeOciProcess) GetAdditionalGids() []uint32 {
	c.init()
	return c.RuntimeOciSpec.Process.User.AdditionalGids
}

func (c *DefaultRuntimeOciProcess) SetAdditionalGids(gids []uint32) error {
	c.init()
	return nil
}

func (c *DefaultRuntimeOciProcess) AddAdditionalGid(gid uint32) error {
	c.init()
	return nil
}
func (c *DefaultRuntimeOciProcess) DelAdditionalGid(gid uint32) error {
	c.init()
	return nil
}

func (c *DefaultRuntimeOciProcess) GetUsername() string {
	c.init()
	return c.RuntimeOciSpec.Process.User.Username
}

func (c *DefaultRuntimeOciProcess) SetUsername(name string) {
	c.init()
}

func (c *DefaultRuntimeOciProcess) SetArgs(args []string) error {
	c.init()
	for _, arg := range args {
		c.AddArg(arg)
	}
	return nil
}

func (c *DefaultRuntimeOciProcess) GetArgs() []string {
	c.init()
	return c.RuntimeOciSpec.Process.Args
}

func (c *DefaultRuntimeOciProcess) AddArg(arg string) error {
	c.init()
	c.RuntimeOciSpec.Process.Args = append(c.RuntimeOciSpec.Process.Args, arg)
	return nil
}

func (c *DefaultRuntimeOciProcess) DelArg(arg string) error {
	c.init()
	return nil
}

func (c *DefaultRuntimeOciProcess) SetEnv(env []string) error {
	c.init()
	for _, e := range env {
		if err := c.AddEnv(e); err != nil {
			return err
		}
	}
	return nil
}

func (c *DefaultRuntimeOciProcess) GetEnv() []string {
	c.init()
	return c.RuntimeOciSpec.Process.Env
}

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

func (c *DefaultRuntimeOciProcess) DelEnv(env string) error {
	c.init()
	if i := strings.IndexByte(env, '='); i != -1 {
		if idx, present := environ[env[:i]]; present {
			c.RuntimeOciSpec.Process.Env = append(c.RuntimeOciSpec.Process.Env[:idx], c.RuntimeOciSpec.Process.Env[idx+1:]...)
			delete(environ, env[:i])
			return nil
		}
		return fmt.Errorf("environment variable %s doesn't exists", env[:i])
	} else {
		return fmt.Errorf("bad formatted environment variable: %s", env)
	}
}

func (c *DefaultRuntimeOciProcess) SetCwd(cwd string) {
	c.init()
	c.RuntimeOciSpec.Process.Cwd = cwd
}

func (c *DefaultRuntimeOciProcess) GetCwd() string {
	c.init()
	return c.RuntimeOciSpec.Process.Cwd
}
