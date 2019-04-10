// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cmdline

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/spf13/cobra"
)

var parentCmd = &cobra.Command{Use: "parent"}
var childCmd = &cobra.Command{Use: "child"}

func newCommandManager(cmd *cobra.Command) (cm *CommandManager, err error) {
	defer func() {
		if t := recover(); t != nil {
			err = fmt.Errorf("%s", t)
		}
	}()
	return NewCommandManager(cmd), nil
}

func registerCmd(cm *CommandManager, cmd *cobra.Command) (err error) {
	defer func() {
		if t := recover(); t != nil {
			err = fmt.Errorf("%s", t)
		}
	}()
	cm.RegisterCmd(cmd, false)
	return
}

func registerSubCmd(cm *CommandManager, parent, child *cobra.Command) (err error) {
	defer func() {
		if t := recover(); t != nil {
			err = fmt.Errorf("%s", t)
		}
	}()
	cm.RegisterSubCmd(parent, child, false)
	return
}

func setCmdGroup(cm *CommandManager, name string, cmds ...*cobra.Command) (err error) {
	defer func() {
		if t := recover(); t != nil {
			err = fmt.Errorf("%s", t)
		}
	}()
	cm.SetCmdGroup(name, cmds...)
	return
}

func TestCommandManager(t *testing.T) {
	_, err := newCommandManager(nil)
	if err == nil {
		t.Errorf("unexpected success with root nil command")
	}
	cm, err := newCommandManager(parentCmd)
	if err != nil {
		t.Errorf("unexpected error while instantiating new command manager: %err", err)
	}
	if cm.GetRootCmd() != parentCmd {
		t.Errorf("unexpected root command returned")
	}
}

func TestRegisterCmd(t *testing.T) {
	cm, err := newCommandManager(parentCmd)
	if err != nil {
		t.Errorf("unexpected error while instantiating new command manager: %err", err)
	}

	if err := registerCmd(cm, nil); err == nil {
		t.Errorf("unexpected success with nil command")
	}

	if err := registerCmd(cm, childCmd); err != nil {
		t.Errorf("unexpected error while registering command: %s", err)
	}

	if cm.GetCmd("nochild") != nil {
		t.Errorf("unexpected command returned")
	}
	if cm.GetCmd("child") != childCmd {
		t.Errorf("unexpected child command returned")
	}
}

func TestRegisterSubCmd(t *testing.T) {
	cm, err := newCommandManager(parentCmd)
	if err != nil {
		t.Errorf("unexpected error while instantiating new command manager: %err", err)
	}

	if err := registerSubCmd(cm, nil, childCmd); err == nil {
		t.Errorf("unexpected success with nil parent command")
	}

	if err := registerSubCmd(cm, parentCmd, nil); err == nil {
		t.Errorf("unexpected success with nil child command")
	}

	if err := registerSubCmd(cm, parentCmd, childCmd); err != nil {
		t.Errorf("unexpected error while registering command: %s", err)
	}

	if cm.GetSubCmd(parentCmd, "nochild") != nil {
		t.Errorf("unexpected command returned")
	}
	if cm.GetSubCmd(parentCmd, "child") != childCmd {
		t.Errorf("unexpected child command returned")
	}
}

func TestCmdGroup(t *testing.T) {
	cm, err := newCommandManager(parentCmd)
	if err != nil {
		t.Errorf("unexpected error while instantiating new command manager: %err", err)
	}

	if err := setCmdGroup(cm, "test", []*cobra.Command{nil}...); err == nil {
		t.Errorf("unexpected success with nil group command")
	}

	testGroup := []*cobra.Command{parentCmd, childCmd}
	if err := setCmdGroup(cm, "test", testGroup...); err != nil {
		t.Errorf("unexpected error while creating group command: %s", err)
	}

	cmdGroup := cm.GetCmdGroup("test")
	if !reflect.DeepEqual(testGroup, cmdGroup) {
		t.Errorf("unexpected group command returned")
	}
}
