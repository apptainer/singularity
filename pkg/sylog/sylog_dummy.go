// Copyright (c) 2019-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

//go:build !sylog
// +build !sylog

package sylog

import (
	"io"
	"io/ioutil"
	"os"
)

// Fatalf is a dummy function exiting with code 255. This
// function must not be used in public packages.
func Fatalf(format string, a ...interface{}) {
	os.Exit(255)
}

// Errorf is a dummy function doing nothing.
func Errorf(format string, a ...interface{}) {}

// Warningf is a dummy function doing nothing.
func Warningf(format string, a ...interface{}) {}

// Infof is a dummy function doing nothing.
func Infof(format string, a ...interface{}) {}

// Verbosef is a dummy function doing nothing.
func Verbosef(format string, a ...interface{}) {}

// Debugf is a dummy function doing nothing
func Debugf(format string, a ...interface{}) {}

// SetLevel is a dummy function doing nothing.
func SetLevel(l int, color bool) {}

// DisableColor for the logger
func DisableColor() {}

// GetLevel is a dummy function returning lowest message level.
func GetLevel() int {
	return int(-1)
}

// GetEnvVar is a dummy function returning environment variable
// with lowest message level.
func GetEnvVar() string {
	return "SINGULARITY_MESSAGELEVEL=-1"
}

// Writer is a dummy function returning ioutil.Discard writer.
func Writer() io.Writer {
	return ioutil.Discard
}

// DebugLogger is an implementation of the go-log/log Logger interface that will
// output log messages via sylog.debug when required by external packages such
// as the scs-library-client
type DebugLogger struct{}

// Log is a dummy function doing nothing.
func (t DebugLogger) Log(v ...interface{}) {}

// Logf is a dummy function doing nothing.
func (t DebugLogger) Logf(format string, v ...interface{}) {}
