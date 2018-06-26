// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// Package sylog implements a basic logger for Singularity Go code to log
// messages in the same format as singularity_message() from C code
package sylog

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
)

type messageLevel int

const (
	fatal messageLevel = iota - 4
	error
	warn
	log
	info
	_
	verbose
	verbose2
	verbose3
	debug
)

func (l messageLevel) String() string {
	str, ok := messageLabels[l]

	if !ok {
		str = "????"
	}

	return str
}

var messageLabels = map[messageLevel]string{
	fatal:    "FATAL",
	error:    "ERROR",
	warn:     "WARNING",
	log:      "LOG",
	info:     "INFO",
	verbose:  "VERBOSE",
	verbose2: "VERBOSE",
	verbose3: "VERBOSE",
	debug:    "DEBUG",
}

var messageColors = map[messageLevel]string{
	fatal: "\x1b[31m",
	error: "\x1b[31m",
	warn:  "\x1b[33m",
	info:  "\x1b[34m",
}

const colorReset string = "\x1b[0m"

var loggerLevel messageLevel

func init() {
	_level, ok := os.LookupEnv("SINGULARITY_MESSAGELEVEL")
	if !ok {
		loggerLevel = debug
		//Printf(INFO, "SINGULARITY_MESSAGELEVEL not set, defaulting to 5")
	} else {
		_levelint, err := strconv.Atoi(_level)
		if err != nil {
			loggerLevel = debug
		} else {
			loggerLevel = messageLevel(_levelint)
		}
	}
}

func writef(level messageLevel, format string, a ...interface{}) {
	if loggerLevel < level {
		return
	}

	pc, _, _, ok := runtime.Caller(2)
	details := runtime.FuncForPC(pc)

	var funcName string
	if ok && details == nil {
		fmt.Printf("Unable to get details of calling function\n")
		funcName = "UNKNOWN CALLING FUNC"
	} else {
		funcNameSplit := strings.Split(details.Name(), ".")
		funcName = funcNameSplit[len(funcNameSplit)-1] + "()"
	}

	uid := os.Getuid()
	pid := os.Getpid()

	uidStr := fmt.Sprintf("[U=%d,P=%d]", uid, pid)

	messageColor, ok := messageColors[level]
	if !ok {
		messageColor = "\x1b[0m"
	}

	prefix := fmt.Sprintf("%s%-8s%s%-19s%-30s", messageColor, level, colorReset, uidStr, funcName)
	message := fmt.Sprintf(format, a...)

	message = strings.TrimSuffix(message, "\n")

	fmt.Fprintf(os.Stderr, "%s%s\n", prefix, message)
}

// Fatalf is equivalent to a call to Errorf followed by os.Exit(255). Code that
// may be imported by other projects should NOT use Fatalf.
func Fatalf(format string, a ...interface{}) {
	writef(fatal, format, a...)
	os.Exit(255)
}

// Errorf writes an ERROR level message to the log but does not exit. This
// should be called when an error is being returned to the calling thread
func Errorf(format string, a ...interface{}) {
	writef(error, format, a...)
}

// Warningf writes a WARNING level message to the log.
func Warningf(format string, a ...interface{}) {
	writef(warn, format, a...)
}

// Infof writes an INFO level message to the log. By default, INFO level messages
// will always be output (unless running in silent)
func Infof(format string, a ...interface{}) {
	writef(info, format, a...)
}

// Verbosef writes a VERBOSE level message to the log. This should probably be
// deprecated since the granularity is often too fine to be useful.
func Verbosef(format string, a ...interface{}) {
	writef(verbose, format, a...)
}

// Debugf writes a DEBUG level message to the log.
func Debugf(format string, a ...interface{}) {
	writef(debug, format, a...)
}
