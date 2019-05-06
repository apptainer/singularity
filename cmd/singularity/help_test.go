// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// This file has been deprecated and will disappear with version 3.3
// of singularity. The functionality has been moved to e2e/help/help.go

package main

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

func TestHelpSingularity(t *testing.T) {
	tests := []struct {
		name       string
		argv       []string
		shouldPass bool
	}{
		{"NoCommand", []string{}, false},
		{"FlagShort", []string{"-h"}, true},
		{"FlagLong", []string{"--help"}, true},
		{"Command", []string{"help"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			cmd := exec.Command(cmdPath, tt.argv...)
			b, err := cmd.CombinedOutput()
			if err != nil && tt.shouldPass {
				t.Log(string(b))
				t.Fatalf("unexpected failure running '%v': %v", strings.Join(tt.argv, " "), err)
			} else if err == nil && !tt.shouldPass {
				t.Log(string(b))
				t.Fatalf("unexpected success running '%v'", strings.Join(tt.argv, " "))
			}
		}))
	}
}

func TestHelpFailure(t *testing.T) {
	if !*runDisabled {
		t.Skip("disabled until issue addressed") // TODO
	}

	tests := []struct {
		name string
		argv []string
	}{
		{"HelpBogus", []string{"help", "bogus"}},
		{"BogusHelp", []string{"bogus", "help"}},
		{"HelpInstanceBogus", []string{"help", "instance", "bogus"}},
		{"ImageBogusHelp", []string{"image", "bogus", "help"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			cmd := exec.Command(cmdPath, tt.argv...)
			if b, err := cmd.CombinedOutput(); err == nil {
				t.Log(string(b))
				t.Fatalf("unexpected success running '%v'", strings.Join(tt.argv, " "))
			}
		}))
	}
}

func TestHelpCommands(t *testing.T) {
	tests := []struct {
		name string
		argv []string
	}{
		{"Apps", []string{"apps"}},
		{"Bootstrap", []string{"bootstrap"}},
		{"Build", []string{"build"}},
		{"Check", []string{"check"}},
		{"Create", []string{"create"}},
		{"Exec", []string{"exec"}},
		{"Inspect", []string{"inspect"}},
		{"Mount", []string{"mount"}},
		{"Pull", []string{"pull"}},
		{"Run", []string{"run"}},
		{"Shell", []string{"shell"}},
		{"Test", []string{"test"}},
		{"InstanceDotStart", []string{"instance.start"}},
		{"InstanceDotList", []string{"instance.list"}},
		{"InstanceDotStop", []string{"instance.stop"}},
		{"InstanceStart", []string{"instance", "start"}},
		{"InstanceList", []string{"instance", "list"}},
		{"InstanceStop", []string{"instance", "stop"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			tests := []struct {
				name string
				argv []string
				skip bool
			}{
				{"PostFlagShort", append(tt.argv, "-h"), true}, // TODO
				{"PostFlagLong", append(tt.argv, "--help"), false},
				{"PostCommand", append(tt.argv, "help"), false},
				{"PreFlagShort", append([]string{"-h"}, tt.argv...), false},
				{"PreFlagLong", append([]string{"--help"}, tt.argv...), false},
				{"PreCommand", append([]string{"help"}, tt.argv...), false},
			}
			for _, tt := range tests {
				if tt.skip && !*runDisabled {
					t.Skip("disabled until issue addressed")
				}

				t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
					cmd := exec.Command(cmdPath, tt.argv...)
					if b, err := cmd.CombinedOutput(); err != nil {
						t.Log(string(b))
						t.Fatalf("unexpected failure running '%v': %v", strings.Join(tt.argv, " "), err)
					}
				}))
			}
		}))
	}
}
