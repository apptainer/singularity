// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cmdline

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// CommandManager holds root command or first parent
// can stores group of command. A command group can be
// composed of one or many commands
type CommandManager struct {
	rootCmd   *cobra.Command
	groupCmds map[string][]*cobra.Command
	errPool   []error
	fm        *flagManager
}

// FlagError represents a flag error type
type FlagError string

func (f FlagError) Error() string {
	return string(f)
}

// CommandError represents a command error type
type CommandError string

func (c CommandError) Error() string {
	return string(c)
}

func onError(cmd *cobra.Command, err error) error {
	return FlagError(err.Error())
}

// NewCommandManager instantiates a CommandManager.
func NewCommandManager(rootCmd *cobra.Command) *CommandManager {
	if rootCmd == nil {
		panic("nil root command passed")
	}
	cm := &CommandManager{
		rootCmd:   rootCmd,
		groupCmds: make(map[string][]*cobra.Command),
		fm:        newFlagManager(),
	}
	rootCmd.SetFlagErrorFunc(onError)
	return cm
}

func (m *CommandManager) isRegistered(cmd *cobra.Command) bool {
	c := cmd.Parent()
	for c != nil {
		if c == m.rootCmd {
			break
		}
		c = c.Parent()
	}
	return c != nil
}

func (m *CommandManager) pushError(err error) {
	m.errPool = append(m.errPool, err)
}

// GetError returns the error pool.
func (m *CommandManager) GetError() []error {
	return m.errPool
}

// RegisterCmd registers a child command of the root command.
// The registered command is automatically affected to a unique group
// containing this command only, the group name is based on the command name
func (m *CommandManager) RegisterCmd(cmd *cobra.Command) {
	// panic here because it's a misuse of API and generally from
	// global context or init() functions
	if cmd == nil {
		panic("nil command passed")
	}
	cmd.SetFlagErrorFunc(onError)
	m.rootCmd.AddCommand(cmd)
	cmd.Flags().SetInterspersed(false)
	m.SetCmdGroup(m.GetCmdName(cmd), cmd)
}

// RegisterSubCmd registers a child command for parent command given as argument.
// The registered command is automatically affected to a unique group containing
// this command only, the group name is based on the command name appended to
// parents command name (see GetCmdName for details)
func (m *CommandManager) RegisterSubCmd(parentCmd, childCmd *cobra.Command) {
	// panic here because it's a misuse of API and generally from
	// global context or init() functions
	if parentCmd == nil {
		panic("nil parent command passed")
	} else if childCmd == nil {
		panic("nil child command passed")
	} else if !m.isRegistered(parentCmd) {
		panic("parent command not registered")
	}
	parentCmd.AddCommand(childCmd)
	childCmd.Flags().SetInterspersed(false)
	m.SetCmdGroup(m.GetCmdName(childCmd), childCmd)
}

// SetCmdGroup creates a unique group of commands identified by name.
// If group already exists or empty command is passed, this function
// will panic
func (m *CommandManager) SetCmdGroup(name string, cmds ...*cobra.Command) {
	if m.groupCmds[name] != nil {
		panic(fmt.Sprintf("group %s already exists", name))
	}
	tmp := make([]*cobra.Command, 0, len(cmds))
	for _, c := range cmds {
		if c != nil {
			tmp = append(tmp, c)
		}
	}
	// cmds could contain only nil commands, we check length of
	// the temporary allocated array containing only non nil
	// commands
	if len(tmp) == 0 {
		panic(fmt.Sprintf("creation of an empty group %q", name))
	}
	m.groupCmds[name] = tmp
}

// GetRootCmd returns the root command
func (m *CommandManager) GetRootCmd() *cobra.Command {
	return m.rootCmd
}

// GetCmdGroup returns all commands associated with the group name
func (m *CommandManager) GetCmdGroup(name string) []*cobra.Command {
	return m.groupCmds[name]
}

// GetCmd returns a single command based on its unique group name.
// If the command group has more than one command this function
// return a nil command instead.
func (m *CommandManager) GetCmd(name string) *cobra.Command {
	cmds := m.groupCmds[name]
	if cmds == nil || len(cmds) > 1 {
		return nil
	}
	return cmds[0]
}

// GetCmdName returns name associated with the provided command.
// If command is named "child" and has two parents named "parent1"
// and "parent2", this function will return "parent1_parent2_child".
// Passing the root command to this function returns an empty string.
func (m *CommandManager) GetCmdName(cmd *cobra.Command) string {
	var names []string

	for c := cmd; c != nil; c = c.Parent() {
		if c == m.rootCmd {
			break
		}
		names = append(names, c.Name())
	}
	// reverse slice
	for i, j := 0, len(names)-1; i < j; i, j = i+1, j-1 {
		names[i], names[j] = names[j], names[i]
	}
	return strings.Join(names, "_")
}

// RegisterFlagForCmd registers a flag for one or many commands
func (m *CommandManager) RegisterFlagForCmd(flag *Flag, cmds ...*cobra.Command) {
	if err := m.fm.registerFlagForCmd(flag, cmds...); err != nil {
		m.pushError(err)
	}
}

// UpdateCmdFlagFromEnv updates flag's values based on environment variables
// associated with all flags belonging to command provided as argument
func (m *CommandManager) UpdateCmdFlagFromEnv(cmd *cobra.Command, envPrefix string) error {
	return m.fm.updateCmdFlagFromEnv(cmd, envPrefix)
}
