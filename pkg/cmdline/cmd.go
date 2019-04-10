// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cmdline

import (
	"fmt"

	"github.com/spf13/cobra"
)

// CommandManager holds root command or first parent
// and can stores group of command
type CommandManager struct {
	rootCmd   *cobra.Command
	groupCmds map[string][]*cobra.Command
	errPool   []error
	fm        *flagManager
}

// NewCommandManager instantiates a CommandManager
func NewCommandManager(rootCmd *cobra.Command) *CommandManager {
	if rootCmd == nil {
		panic("nil root command passed")
	}
	cm := &CommandManager{
		rootCmd:   rootCmd,
		groupCmds: make(map[string][]*cobra.Command),
		errPool:   make([]error, 0),
		fm:        newFlagManager(),
	}
	return cm
}

func (m *CommandManager) pushError(f string, a ...interface{}) {
	m.errPool = append(m.errPool, fmt.Errorf(f, a...))
}

// GetError returns the error pool
func (m *CommandManager) GetError() []error {
	return m.errPool
}

// RegisterCmd registers a child command for the root command
func (m *CommandManager) RegisterCmd(cmd *cobra.Command, interspersed bool) {
	// panic here because it's a misuse of API and generally from
	// global context or init() functions
	if cmd == nil {
		panic("nil command passed")
	}
	m.rootCmd.AddCommand(cmd)
	cmd.Flags().SetInterspersed(interspersed)
}

// RegisterSubCmd registers a child command for parent command given as argument
func (m *CommandManager) RegisterSubCmd(parentCmd, childCmd *cobra.Command, interspersed bool) {
	// panic here because it's a misuse of API and generally from
	// global context or init() functions
	if parentCmd == nil {
		panic("nil parent command passed")
	} else if childCmd == nil {
		panic("nil child command passed")
	}
	parentCmd.AddCommand(childCmd)
	childCmd.Flags().SetInterspersed(interspersed)
}

// SetCmdGroup creates a group of commands identified by name
func (m *CommandManager) SetCmdGroup(name string, cmds ...*cobra.Command) {
	m.groupCmds[name] = make([]*cobra.Command, 0)
	for _, c := range cmds {
		if c == nil {
			panic("nil command passed")
		}
		m.groupCmds[name] = append(m.groupCmds[name], c)
	}
}

// GetCmdGroup returns group of commands corresponding to name
func (m *CommandManager) GetCmdGroup(name string) []*cobra.Command {
	return m.groupCmds[name]
}

// GetRootCmd returns the root command
func (m *CommandManager) GetRootCmd() *cobra.Command {
	return m.rootCmd
}

// GetCmd returns the named command associated with root command
func (m *CommandManager) GetCmd(name string) *cobra.Command {
	for _, c := range m.rootCmd.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

// GetSubCmd return the named command associated with parent command given as argument
func (m *CommandManager) GetSubCmd(parentCmd *cobra.Command, name string) *cobra.Command {
	for _, c := range parentCmd.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

// RegisterCmdFlag registers a flag for a command
func (m *CommandManager) RegisterCmdFlag(flag *Flag, cmds ...*cobra.Command) {
	if err := m.fm.registerCmdFlag(flag, cmds...); err != nil {
		m.pushError(err.Error())
	}
}

// UpdateCmdFlagFromEnv updates flag's values based on environment variables
// associated with all flags belonging to command provided as argument
func (m *CommandManager) UpdateCmdFlagFromEnv(envPrefix string) {
	for _, e := range m.fm.updateCmdFlagFromEnv(m.rootCmd, envPrefix) {
		m.pushError(e.Error())
	}
}
