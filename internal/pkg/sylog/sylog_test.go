// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build sylog

package sylog

import (
	"strings"
	"testing"
)

// TestSylogSetLevel to ensure the correct integer is returned
func TestSylogSetLevel(t *testing.T) {
	var levelTests = []struct {
		name  string
		level int
	}{
		{"SetLevelFatal", int(fatal)},
		{"SetLevelError", int(error)},
		{"SetLevelWarn", int(warn)},
		{"SetLevelLog", int(log)},
		{"SetLevelInfo", int(info)},
		{"SetLevelVerbose", int(verbose)},
		{"SetLevelVerbose2", int(verbose2)},
		{"SetLevelVerbose3", int(verbose3)},
		{"SetLevelDebug", int(debug)},
	}

	for _, tt := range levelTests {
		t.Run(tt.name, func(t *testing.T) {
			SetLevel(tt.level)

			l := GetLevel()

			if l != tt.level {
				t.Errorf("got %d, want %d", l, tt.level)
			}
		})
	}
}

func TestSylogPrefix(t *testing.T) {

	var logSuffix = "\x1b[0m "
	var levelTests = []struct {
		name  string
		level messageLevel
	}{
		{"\x1b[31mFATAL", fatal},
		{"\x1b[31mERROR", error},
		{"\x1b[33mWARNING", warn},
		{"\x1b[0mLOG", log},
		{"\x1b[34mINFO", info},
		{"\x1b[0mVERBOSE", verbose},
		{"\x1b[0mVERBOSE", verbose2},
		{"\x1b[0mVERBOSE", verbose3},
		{"\x1b[0mDEBUG", debug},
	}

	// Test that prefix is colored
	for _, tt := range levelTests {
		t.Run(tt.name, func(t *testing.T) {

			SetLevel(int(tt.level))
			levelPrefix := prefix(tt.level)

			// Check that we start with the color prefix
			if !strings.HasPrefix(levelPrefix, tt.name) {
				t.Errorf("got prefix %s, want %s", levelPrefix, tt.name)
			}

			// The suffix should be consistently the "off" color string
			if !strings.HasSuffix(levelPrefix, logSuffix) && tt.level != debug {
				t.Errorf("%s does not end with %s", levelPrefix, logSuffix)
			}
		})
	}
}

func TestSylogDisableColor(t *testing.T) {

	var logSuffix = "\x1b[0m "
	var levelTests = []struct {
		name  string
		level messageLevel
	}{
		{"FATAL", fatal},
		{"ERROR", error},
		{"WARNING", warn},
		{"LOG", log},
		{"INFO", info},
		{"VERBOSE", verbose},
		{"VERBOSE", verbose2},
		{"VERBOSE", verbose3},
		{"DEBUG", debug},
	}

	// Disable all color output, removing prefix and off suffix
	DisableColor()

	// Test that prefix is colored
	for _, tt := range levelTests {
		t.Run(tt.name, func(t *testing.T) {

			SetLevel(int(tt.level))
			levelPrefix := prefix(tt.level)

			// Check that we start with the color prefix
			if !strings.HasPrefix(levelPrefix, tt.name) {
				t.Errorf("got prefix %s, want %s", levelPrefix, tt.name)
			}

			// The suffix should be consistently the "off" color string
			if strings.HasSuffix(levelPrefix, logSuffix) && tt.level != debug {
				t.Errorf("%s does ends with %s", levelPrefix, logSuffix)
			}
		})
	}
}
