// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestHelpSingularity(t *testing.T) {
	tests := []struct {
		name string
		argv []string
	}{
		{"NoCommand", []string{}},
		{"FlagShort", []string{"-h"}},
		{"FlagLong", []string{"--help"}},
		{"Command", []string{"help"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(cmdPath, tt.argv...)
			if b, err := cmd.CombinedOutput(); err != nil {
				t.Log(string(b))
				t.Fatalf("unexpected failure running '%v': %v", strings.Join(tt.argv, " "), err)
			}
		})
	}
}

func TestHelpFailure(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(cmdPath, tt.argv...)
			if b, err := cmd.CombinedOutput(); err == nil {
				t.Log(string(b))
				t.Fatalf("unexpected success running '%v'", strings.Join(tt.argv, " "))
			}
		})
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
		{"Image", []string{"image"}},
		{"ImageDotCreate", []string{"image.create"}},
		{"ImageDotExpand", []string{"image.expand"}},
		{"ImageDotExport", []string{"image.export"}},
		{"ImageDotImport", []string{"image.import"}},
		{"ImageCreate", []string{"image", "create"}},
		{"ImageExpand", []string{"image", "expand"}},
		{"ImageExport", []string{"image", "export"}},
		{"ImageImport", []string{"image", "import"}},
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
		t.Run(tt.name, func(t *testing.T) {
			tests := []struct {
				name string
				argv []string
			}{
				{"PostFlagShort", append(tt.argv, "-h")},
				{"PostFlagLong", append(tt.argv, "--help")},
				{"PostCommand", append(tt.argv, "help")},
				{"PreFlagShort", append([]string{"-h"}, tt.argv...)},
				{"PreFlagLong", append([]string{"--help"}, tt.argv...)},
				{"PreCommand", append([]string{"help"}, tt.argv...)},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					cmd := exec.Command(cmdPath, tt.argv...)
					if b, err := cmd.CombinedOutput(); err != nil {
						t.Log(string(b))
						t.Fatalf("unexpected failure running '%v': %v", strings.Join(tt.argv, " "), err)
					}
				})
			}
		})
	}
}
