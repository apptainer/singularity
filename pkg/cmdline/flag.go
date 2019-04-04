// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cmdline

import (
	"os"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Supported operating systems
const (
	// Darwin OS
	Darwin = "darwin"
	// Linux OS
	Linux = "linux"
)

// Flag ...
type Flag struct {
	ID           string
	Value        interface{}
	DefaultValue interface{}
	Name         string
	ShortHand    string
	Usage        string
	Tag          string
	Deprecated   string
	Hidden       bool
	EnvKeys      []string
	EnvHandler   EnvHandler
	ExcludedOS   []string
}

// FlagManager ...
type FlagManager struct {
	flags map[string]*Flag
}

// NewFlagManager ...
func NewFlagManager() *FlagManager {
	return &FlagManager{make(map[string]*Flag)}
}

// RegisterFlagAnnotation ...
func (m *FlagManager) RegisterFlagAnnotation(flag *Flag, cmd *cobra.Command) {
	cmd.Flags().SetAnnotation(flag.Name, "argtag", []string{flag.Tag})

	if len(flag.EnvKeys) > 0 {
		cmd.Flags().SetAnnotation(flag.Name, "envkey", flag.EnvKeys)
	}
	cmd.Flags().SetAnnotation(flag.Name, "ID", []string{flag.ID})

	if flag.Deprecated != "" {
		cmd.Flags().MarkDeprecated(flag.Name, flag.Deprecated)
	}
	if flag.Hidden {
		cmd.Flags().MarkHidden(flag.Name)
	}
}

// RegisterCmdFlag ...
func (m *FlagManager) RegisterCmdFlag(flag *Flag, cmds ...*cobra.Command) {
	for _, os := range flag.ExcludedOS {
		if os == runtime.GOOS {
			return
		}
	}
	switch flag.DefaultValue.(type) {
	case string:
		if flag.EnvHandler == nil && len(flag.EnvKeys) > 0 {
			flag.EnvHandler = EnvStringNSlice
		}
		m.registerStringVar(flag, cmds)
	case []string:
		if flag.EnvHandler == nil && len(flag.EnvKeys) > 0 {
			flag.EnvHandler = EnvStringNSlice
		}
		m.registerStringSliceVar(flag, cmds)
	case bool:
		if flag.EnvHandler == nil && len(flag.EnvKeys) > 0 {
			flag.EnvHandler = EnvBool
		}
		m.registerBoolVar(flag, cmds)
	case int:
		flag.EnvHandler = nil
		m.registerIntVar(flag, cmds)
	case uint32:
		flag.EnvHandler = nil
		m.registerUint32Var(flag, cmds)
	}
	m.flags[flag.ID] = flag
}

func (m *FlagManager) registerStringVar(flag *Flag, cmds []*cobra.Command) {
	for _, c := range cmds {
		if flag.ShortHand != "" {
			c.Flags().StringVarP(flag.Value.(*string), flag.Name, flag.ShortHand, flag.DefaultValue.(string), flag.Usage)
		} else {
			c.Flags().StringVar(flag.Value.(*string), flag.Name, flag.DefaultValue.(string), flag.Usage)
		}
		m.RegisterFlagAnnotation(flag, c)
	}
}

func (m *FlagManager) registerStringSliceVar(flag *Flag, cmds []*cobra.Command) {
	for _, c := range cmds {
		if flag.ShortHand != "" {
			c.Flags().StringSliceVarP(flag.Value.(*[]string), flag.Name, flag.ShortHand, flag.DefaultValue.([]string), flag.Usage)
		} else {
			c.Flags().StringSliceVar(flag.Value.(*[]string), flag.Name, flag.DefaultValue.([]string), flag.Usage)
		}
		m.RegisterFlagAnnotation(flag, c)
	}
}

func (m *FlagManager) registerBoolVar(flag *Flag, cmds []*cobra.Command) {
	for _, c := range cmds {
		if flag.ShortHand != "" {
			c.Flags().BoolVarP(flag.Value.(*bool), flag.Name, flag.ShortHand, flag.DefaultValue.(bool), flag.Usage)
		} else {
			c.Flags().BoolVar(flag.Value.(*bool), flag.Name, flag.DefaultValue.(bool), flag.Usage)
		}
		m.RegisterFlagAnnotation(flag, c)
	}
}

func (m *FlagManager) registerIntVar(flag *Flag, cmds []*cobra.Command) {
	for _, c := range cmds {
		if flag.ShortHand != "" {
			c.Flags().IntVarP(flag.Value.(*int), flag.Name, flag.ShortHand, flag.DefaultValue.(int), flag.Usage)
		} else {
			c.Flags().IntVar(flag.Value.(*int), flag.Name, flag.DefaultValue.(int), flag.Usage)
		}
		m.RegisterFlagAnnotation(flag, c)
	}
}

func (m *FlagManager) registerUint32Var(flag *Flag, cmds []*cobra.Command) {
	for _, c := range cmds {
		if flag.ShortHand != "" {
			c.Flags().Uint32VarP(flag.Value.(*uint32), flag.Name, flag.ShortHand, flag.DefaultValue.(uint32), flag.Usage)
		} else {
			c.Flags().Uint32Var(flag.Value.(*uint32), flag.Name, flag.DefaultValue.(uint32), flag.Usage)
		}
		m.RegisterFlagAnnotation(flag, c)
	}
}

// UpdateCmdFlagFromEnv ...
func (m *FlagManager) UpdateCmdFlagFromEnv(cmd *cobra.Command, prefix string) {
	fn := func(flag *pflag.Flag) {
		envKeys, ok := flag.Annotations["envkey"]
		if !ok {
			return
		}
		id, ok := flag.Annotations["ID"]
		if !ok {
			return
		}
		mflag, ok := m.flags[id[0]]
		if !ok {
			return
		}
		for _, key := range envKeys {
			val, set := os.LookupEnv(prefix + key)
			if !set {
				continue
			}
			if mflag.EnvHandler != nil {
				mflag.EnvHandler(flag, val)
			}
		}
	}
	cmd.Flags().VisitAll(fn)
}
