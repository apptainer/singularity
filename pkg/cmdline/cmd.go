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
		fm:        newFlagManager(),
	}
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
	m.SetCmdGroup(m.GetCmdName(cmd), cmd)
}

// RegisterSubCmd registers a child command for parent command given as argument
func (m *CommandManager) RegisterSubCmd(parentCmd, childCmd *cobra.Command, interspersed bool) {
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
	childCmd.Flags().SetInterspersed(interspersed)
	m.SetCmdGroup(m.GetCmdName(childCmd), childCmd)
}

// SetCmdGroup creates a group of commands identified by name
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
	if len(tmp) == 0 {
		panic("creation of an empty group")
	}
	m.groupCmds[name] = tmp
}

// GetRootCmd returns the root command
func (m *CommandManager) GetRootCmd() *cobra.Command {
	return m.rootCmd
}

// GetCmdGroup returns group of commands corresponding to name
func (m *CommandManager) GetCmdGroup(name string) []*cobra.Command {
	return m.groupCmds[name]
}

// GetCmd returns the named command associated with root command
func (m *CommandManager) GetCmd(name string) *cobra.Command {
	cmds := m.groupCmds[name]
	if cmds == nil || len(cmds) > 1 {
		return nil
	}
	return cmds[0]
}

// GetCmdName returns name associated with the provided command.
// If command is named child and has two parents named parent1 and parent2,
// this function will return "parent1_parent2_child".
// Passing the root command to this function returns an empty string.
func (m *CommandManager) GetCmdName(cmd *cobra.Command) string {
	var names []string

	c := cmd
	for c != nil {
		if c == m.rootCmd {
			break
		}
		names = append([]string{c.Name()}, names...)
		c = c.Parent()
	}

	return strings.Join(names, "_")
}

// RegisterCmdFlag registers a flag for a command
func (m *CommandManager) RegisterCmdFlag(flag *Flag, cmds ...*cobra.Command) {
	if err := m.fm.registerCmdFlag(flag, cmds...); err != nil {
		m.pushError(err)
	}
}

// UpdateCmdFlagFromEnv updates flag's values based on environment variables
// associated with all flags belonging to command provided as argument
func (m *CommandManager) UpdateCmdFlagFromEnv(envPrefix string) {
	for _, e := range m.fm.updateCmdFlagFromEnv(m.rootCmd, envPrefix) {
		m.pushError(e)
	}
}
