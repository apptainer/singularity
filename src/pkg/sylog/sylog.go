/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package sylog

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
)

const ABRT = -4
const ERROR = -3
const WARNING = -2
const LOG = -1
const INFO = 1
const VERBOSE = 2
const VERBOSE1 = 2
const VERBOSE2 = 3
const VERBOSE3 = 4
const DEBUG = 5

var messagelevel int

func init() {
	_level, ok := os.LookupEnv("SINGULARITY_MESSAGELEVEL")
	if !ok {
		messagelevel = 5
		Printf(INFO, "SINGULARITY_MESSAGELEVEL not set, defaulting to 5")
	} else {
		_levelint, err := strconv.Atoi(_level)
		if err != nil {
			messagelevel = 5
		} else {
			messagelevel = _levelint
		}
	}
}

func Printf(level int, format string, a ...interface{}) {
	var prefix_level string

	if messagelevel < level {
		return
	}

	pc, _, _, ok := runtime.Caller(1)
	details := runtime.FuncForPC(pc)

	if ok && details == nil {
		fmt.Printf("Unable to get details of calling function\n")
	}

	funcNameSplit := strings.Split(details.Name(), ".")
	funcName := funcNameSplit[len(funcNameSplit)-1] + "()"

	uid := os.Getuid()
	pid := os.Getpid()

	uidstring := fmt.Sprintf("[U=%d,P=%d]", uid, pid)

	switch level {
	case ABRT:
		prefix_level = "ABRT"
	case ERROR:
		prefix_level = "ERROR"
	case WARNING:
		prefix_level = "WARNING"
	case INFO:
		prefix_level = "INFO"
	case VERBOSE:
		prefix_level = "VERBOSE"
	case VERBOSE2:
		prefix_level = "VERBOSE"
	case VERBOSE3:
		prefix_level = "VERBOSE"
	case DEBUG:
		prefix_level = "DEBUG"
	default:
		prefix_level = "????????"
	}

	prefix := fmt.Sprintf("%-8s%-19s%-30s", prefix_level, uidstring, funcName)
	message := fmt.Sprintf(format, a...)

	fmt.Printf("%s%s", prefix, message)
}
