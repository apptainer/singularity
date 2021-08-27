// Copyright (c) 2018-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

//go:build sylog
// +build sylog

package sylog

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"strconv"
	"strings"
)

const messageLevelEnv = "SINGULARITY_MESSAGELEVEL"

var messageColors = map[messageLevel]string{
	FatalLevel: "\x1b[31m",
	ErrorLevel: "\x1b[31m",
	WarnLevel:  "\x1b[33m",
	InfoLevel:  "\x1b[34m",
}

var (
	noColorLevel messageLevel = 90
	loggerLevel  messageLevel = InfoLevel
)

var logWriter = (io.Writer)(os.Stderr)

func init() {
	level, err := strconv.Atoi(os.Getenv(messageLevelEnv))
	if err == nil {
		loggerLevel = messageLevel(level)
	}
}

func prefix(logLevel, msgLevel messageLevel) string {
	colorReset := "\x1b[0m"
	messageColor, ok := messageColors[msgLevel]
	if !ok || logLevel != loggerLevel {
		colorReset = ""
		messageColor = ""
	}

	// This section builds and returns the prefix for levels < debug
	if logLevel < DebugLevel {
		return fmt.Sprintf("%s%-8s%s ", messageColor, msgLevel.String()+":", colorReset)
	}

	pc, _, _, ok := runtime.Caller(3)
	details := runtime.FuncForPC(pc)

	var funcName string
	if ok && details == nil {
		funcName = "????()"
	} else {
		funcNameSplit := strings.Split(details.Name(), ".")
		funcName = funcNameSplit[len(funcNameSplit)-1] + "()"
	}

	uid := os.Geteuid()
	pid := os.Getpid()
	uidStr := fmt.Sprintf("[U=%d,P=%d]", uid, pid)

	return fmt.Sprintf("%s%-8s%s%-19s%-30s", messageColor, msgLevel, colorReset, uidStr, funcName)
}

func writef(msgLevel messageLevel, format string, a ...interface{}) {
	logLevel := getLoggerLevel()
	if logLevel < msgLevel {
		return
	}

	message := fmt.Sprintf(format, a...)
	message = strings.TrimRight(message, "\n")

	fmt.Fprintf(logWriter, "%s%s\n", prefix(logLevel, msgLevel), message)
}

func getLoggerLevel() messageLevel {
	if loggerLevel <= -noColorLevel {
		return loggerLevel + noColorLevel
	} else if loggerLevel >= noColorLevel {
		return loggerLevel - noColorLevel
	}
	return loggerLevel
}

// Fatalf is equivalent to a call to Errorf followed by os.Exit(255). Code that
// may be imported by other projects should NOT use Fatalf.
func Fatalf(format string, a ...interface{}) {
	writef(FatalLevel, format, a...)
	os.Exit(255)
}

// Errorf writes an ERROR level message to the log but does not exit. This
// should be called when an error is being returned to the calling thread
func Errorf(format string, a ...interface{}) {
	writef(ErrorLevel, format, a...)
}

// Warningf writes a WARNING level message to the log.
func Warningf(format string, a ...interface{}) {
	writef(WarnLevel, format, a...)
}

// Infof writes an INFO level message to the log. By default, INFO level messages
// will always be output (unless running in silent)
func Infof(format string, a ...interface{}) {
	writef(InfoLevel, format, a...)
}

// Verbosef writes a VERBOSE level message to the log. This should probably be
// deprecated since the granularity is often too fine to be useful.
func Verbosef(format string, a ...interface{}) {
	writef(VerboseLevel, format, a...)
}

// Debugf writes a DEBUG level message to the log.
func Debugf(format string, a ...interface{}) {
	writef(DebugLevel, format, a...)
}

// SetLevel explicitly sets the loggerLevel
func SetLevel(l int, color bool) {
	loggerLevel = messageLevel(l)
	if !color {
		if loggerLevel >= InfoLevel {
			loggerLevel = loggerLevel + noColorLevel
		} else if loggerLevel <= LogLevel {
			loggerLevel = loggerLevel - noColorLevel
		}
	}
}

// GetLevel returns the current log level as integer
func GetLevel() int {
	return int(getLoggerLevel())
}

// GetEnvVar returns a formatted environment variable string which
// can later be interpreted by init() in a child proc
func GetEnvVar() string {
	return fmt.Sprintf("%s=%d", messageLevelEnv, loggerLevel)
}

// Writer returns an io.Writer to pass to an external packages logging utility.
// i.e when --quiet option is set, this function returns ioutil.Discard writer to ignore output
func Writer() io.Writer {
	if loggerLevel <= LogLevel {
		return ioutil.Discard
	}

	return logWriter
}

// DebugLogger is an implementation of the go-log/log Logger interface that will
// output log messages via sylog.debug when required by external packages such
// as the scs-library-client
type DebugLogger struct{}

// Log outputs a log message via sylog.Debugf
func (t DebugLogger) Log(v ...interface{}) {
	writef(DebugLevel, "%s", fmt.Sprint(v...))
}

// Logf outputs a formatted log message via sylog.Debugf
func (t DebugLogger) Logf(format string, v ...interface{}) {
	writef(DebugLevel, format, v...)
}
