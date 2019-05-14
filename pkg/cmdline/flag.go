// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cmdline

import (
	"fmt"
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

// Flag holds information about a command flag
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
	Required     bool
	EnvKeys      []string
	EnvHandler   EnvHandler
	ExcludedOS   []string
}

// flagManager manages cobra command flags and store them
// in a hash map
type flagManager struct {
	flags map[string]*Flag
}

// newFlagManager instantiates a flag manager and returns it
func newFlagManager() *flagManager {
	return &flagManager{
		flags: make(map[string]*Flag),
	}
}

func (m *flagManager) setFlagOptions(flag *Flag, cmd *cobra.Command) {
	cmd.Flags().SetAnnotation(flag.Name, "argtag", []string{flag.Tag})
	cmd.Flags().SetAnnotation(flag.Name, "ID", []string{flag.ID})

	if len(flag.EnvKeys) > 0 {
		cmd.Flags().SetAnnotation(flag.Name, "envkey", flag.EnvKeys)
	}
	if flag.Deprecated != "" {
		cmd.Flags().MarkDeprecated(flag.Name, flag.Deprecated)
	}
	if flag.Hidden {
		cmd.Flags().MarkHidden(flag.Name)
	}
	if flag.Required {
		cmd.MarkFlagRequired(flag.Name)
	}
}

func (m *flagManager) registerFlagForCmd(flag *Flag, cmds ...*cobra.Command) error {
	for _, c := range cmds {
		if c == nil {
			return fmt.Errorf("nil command provided")
		}
	}
	if flag == nil {
		return fmt.Errorf("nil flag provided")
	}
	for _, os := range flag.ExcludedOS {
		if os == runtime.GOOS {
			return nil
		}
	}
	if flag.EnvHandler == nil {
		flag.EnvHandler = EnvSetValue
	}
	switch t := flag.DefaultValue.(type) {
	case string:
		m.registerStringVar(flag, cmds)
	case []string:
		m.registerStringSliceVar(flag, cmds)
	case bool:
		m.registerBoolVar(flag, cmds)
	case int:
		m.registerIntVar(flag, cmds)
	case uint32:
		m.registerUint32Var(flag, cmds)
	default:
		return fmt.Errorf("flag of type %s is not supported", t)
	}
	m.flags[flag.ID] = flag
	return nil
}

func (m *flagManager) registerStringVar(flag *Flag, cmds []*cobra.Command) error {
	for _, c := range cmds {
		if flag.ShortHand != "" {
			c.Flags().StringVarP(flag.Value.(*string), flag.Name, flag.ShortHand, flag.DefaultValue.(string), flag.Usage)
		} else {
			c.Flags().StringVar(flag.Value.(*string), flag.Name, flag.DefaultValue.(string), flag.Usage)
		}
		m.setFlagOptions(flag, c)
	}
	return nil
}

func (m *flagManager) registerStringSliceVar(flag *Flag, cmds []*cobra.Command) error {
	for _, c := range cmds {
		if flag.ShortHand != "" {
			c.Flags().StringSliceVarP(flag.Value.(*[]string), flag.Name, flag.ShortHand, flag.DefaultValue.([]string), flag.Usage)
		} else {
			c.Flags().StringSliceVar(flag.Value.(*[]string), flag.Name, flag.DefaultValue.([]string), flag.Usage)
		}
		m.setFlagOptions(flag, c)
	}
	return nil
}

func (m *flagManager) registerBoolVar(flag *Flag, cmds []*cobra.Command) error {
	for _, c := range cmds {
		if flag.ShortHand != "" {
			c.Flags().BoolVarP(flag.Value.(*bool), flag.Name, flag.ShortHand, flag.DefaultValue.(bool), flag.Usage)
		} else {
			c.Flags().BoolVar(flag.Value.(*bool), flag.Name, flag.DefaultValue.(bool), flag.Usage)
		}
		m.setFlagOptions(flag, c)
	}
	return nil
}

func (m *flagManager) registerIntVar(flag *Flag, cmds []*cobra.Command) error {
	for _, c := range cmds {
		if flag.ShortHand != "" {
			c.Flags().IntVarP(flag.Value.(*int), flag.Name, flag.ShortHand, flag.DefaultValue.(int), flag.Usage)
		} else {
			c.Flags().IntVar(flag.Value.(*int), flag.Name, flag.DefaultValue.(int), flag.Usage)
		}
		m.setFlagOptions(flag, c)
	}
	return nil
}

func (m *flagManager) registerUint32Var(flag *Flag, cmds []*cobra.Command) error {
	for _, c := range cmds {
		if flag.ShortHand != "" {
			c.Flags().Uint32VarP(flag.Value.(*uint32), flag.Name, flag.ShortHand, flag.DefaultValue.(uint32), flag.Usage)
		} else {
			c.Flags().Uint32Var(flag.Value.(*uint32), flag.Name, flag.DefaultValue.(uint32), flag.Usage)
		}
		m.setFlagOptions(flag, c)
	}
	return nil
}

func (m *flagManager) updateCmdFlagFromEnv(cmd *cobra.Command, prefix string) error {
	var errs []error

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
				if err := mflag.EnvHandler(flag, val); err != nil {
					errs = append(errs, err)
					break
				}
			}
		}
	}

	// visit each child commands
	cmd.Flags().VisitAll(fn)
	if len(errs) > 0 {
		errStr := ""
		for _, e := range errs {
			errStr += fmt.Sprintf("\n%s", e.Error())
		}
		return fmt.Errorf(errStr)
	}

	return nil
}
