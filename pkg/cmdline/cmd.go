// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cmdline

import (
	"github.com/spf13/cobra"
)

// CommandManager ...
type CommandManager struct {
	rootCmd   *cobra.Command
	groupCmds map[string][]*cobra.Command
}

// NewCommandManager ...
func NewCommandManager(rootCmd *cobra.Command) *CommandManager {
	return &CommandManager{
		rootCmd:   rootCmd,
		groupCmds: make(map[string][]*cobra.Command),
	}
}

// RegisterCmd ...
func (m *CommandManager) RegisterCmd(cmd *cobra.Command, interspersed bool) {
	m.rootCmd.AddCommand(cmd)
	cmd.Flags().SetInterspersed(interspersed)
}

// RegisterSubCmd ...
func (m *CommandManager) RegisterSubCmd(parentCmd, subCmd *cobra.Command, interspersed bool) {
	parentCmd.AddCommand(subCmd)
	subCmd.Flags().SetInterspersed(interspersed)
}

// SetCmdGroup ...
func (m *CommandManager) SetCmdGroup(name string, cmds ...*cobra.Command) {
	m.groupCmds[name] = cmds
}

// GetCmdGroup ...
func (m *CommandManager) GetCmdGroup(name string) []*cobra.Command {
	return m.groupCmds[name]
}

// GetRootCmd ...
func (m *CommandManager) GetRootCmd() *cobra.Command {
	return m.rootCmd
}

// GetCmd ...
func (m *CommandManager) GetCmd(name string) *cobra.Command {
	for _, c := range m.rootCmd.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

// GetSubCmd ...
func (m *CommandManager) GetSubCmd(parentCmd *cobra.Command, name string) *cobra.Command {
	for _, c := range parentCmd.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}
