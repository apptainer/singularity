// Copyright (c) 2019-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sylog

type messageLevel int

// Log levels.
const (
	FatalLevel    messageLevel = iota - 4 // FatalLevel    : -4
	ErrorLevel                            // ErrorLevel    : -3
	WarnLevel                             // WarnLevel     : -2
	LogLevel                              // LogLevel      : -1
	_                                     // SKIP     : 0
	InfoLevel                             // InfoLevel     : 1
	VerboseLevel                          // VerboseLevel  : 2
	Verbose2Level                         // Verbose2Level : 3
	Verbose3Level                         // Verbose3Level : 4
	DebugLevel                            // DebugLevel    : 5
)

func (l messageLevel) String() string {
	str, ok := messageLabels[l]

	if !ok {
		str = "????"
	}

	return str
}

var messageLabels = map[messageLevel]string{
	FatalLevel:    "FATAL",
	ErrorLevel:    "ERROR",
	WarnLevel:     "WARNING",
	LogLevel:      "LOG",
	InfoLevel:     "INFO",
	VerboseLevel:  "VERBOSE",
	Verbose2Level: "VERBOSE",
	Verbose3Level: "VERBOSE",
	DebugLevel:    "DEBUG",
}
