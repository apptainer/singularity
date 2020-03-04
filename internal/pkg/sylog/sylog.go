// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

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

	apexlog "github.com/apex/log"
	"github.com/vbauerster/mpb/v4"
	"github.com/vbauerster/mpb/v4/decor"
)

type messageLevel int

const (
	lvlFatal    messageLevel = iota - 4 // Fatal    : -4
	lvlError                            // Error    : -3
	lvlWarn                             // Warn     : -2
	lvlLog                              // Log      : -1
	_                                   // SKIP     : 0
	lvlInfo                             // lInfo     : 1
	lvlVerbose                          // Verbose  : 2
	lvlVerbose2                         // Verbose2 : 3
	lvlVerbose3                         // Verbose3 : 4
	lvlDebug                            // Debug    : 5
)

func (l messageLevel) String() string {
	str, ok := messageLabels[l]

	if !ok {
		str = "????"
	}

	return str
}

var messageLabels = map[messageLevel]string{
	lvlFatal:    "FATAL",
	lvlError:    "ERROR",
	lvlWarn:     "WARNING",
	lvlLog:      "LOG",
	lvlInfo:     "INFO",
	lvlVerbose:  "VERBOSE",
	lvlVerbose2: "VERBOSE",
	lvlVerbose3: "VERBOSE",
	lvlDebug:    "DEBUG",
}

var messageColors = map[messageLevel]string{
	lvlFatal: "\x1b[31m",
	lvlError: "\x1b[31m",
	lvlWarn:  "\x1b[33m",
	lvlInfo:  "\x1b[34m",
}

var colorReset = "\x1b[0m"

var loggerLevel messageLevel

func init() {
	_levelint := int(messageLevel(lvlInfo))
	_levelstr, ok := os.LookupEnv("SINGULARITY_MESSAGELEVEL")
	if ok {
		_leveli, err := strconv.Atoi(_levelstr)
		if err == nil {
			_levelint = _leveli
		}
	}
	SetLevel(_levelint)
}

func prefix(level messageLevel) string {
	messageColor, ok := messageColors[level]
	if !ok {
		messageColor = "\x1b[0m"
	}

	// This section builds and returns the prefix for levels < lvlDebug
	if loggerLevel < lvlDebug {
		return fmt.Sprintf("%s%-8s%s ", messageColor, level.String()+":", colorReset)
	}

	pc, _, _, ok := runtime.Caller(3)
	details := runtime.FuncForPC(pc)

	var funcName string
	if ok && details == nil {
		fmt.Printf("Unable to get details of calling function\n")
		funcName = "UNKNOWN CALLING FUNC"
	} else {
		funcNameSplit := strings.Split(details.Name(), ".")
		funcName = funcNameSplit[len(funcNameSplit)-1] + "()"
	}

	uid := os.Geteuid()
	pid := os.Getpid()
	uidStr := fmt.Sprintf("[U=%d,P=%d]", uid, pid)

	return fmt.Sprintf("%s%-8s%s%-19s%-30s", messageColor, level, colorReset, uidStr, funcName)
}

func writef(w io.Writer, level messageLevel, format string, a ...interface{}) {
	if loggerLevel < level {
		return
	}

	message := fmt.Sprintf(format, a...)
	message = strings.TrimSuffix(message, "\n")

	fmt.Fprintf(w, "%s%s\n", prefix(level), message)
}

// Fatalf is equivalent to a call to Errorf followed by os.Exit(255). Code that
// may be imported by other projects should NOT use Fatalf.
func Fatalf(format string, a ...interface{}) {
	writef(os.Stderr, lvlFatal, format, a...)
	os.Exit(255)
}

// Errorf writes an ERROR level message to the lvlLog but does not exit. This
// should be called when an lvlError is being returned to the calling thread
func Errorf(format string, a ...interface{}) {
	writef(os.Stderr, lvlError, format, a...)
}

// Warningf writes a WARNING level message to the lvlLog.
func Warningf(format string, a ...interface{}) {
	writef(os.Stderr, lvlWarn, format, a...)
}

// Infof writes an INFO level message to the lvlLog. By default, INFO level messages
// will always be output (unless running in silent)
func Infof(format string, a ...interface{}) {
	writef(os.Stderr, lvlInfo, format, a...)
}

// Verbosef writes a VERBOSE level message to the lvlLog. This should probably be
// deprecated since the granularity is often too fine to be useful.
func Verbosef(format string, a ...interface{}) {
	writef(os.Stderr, lvlVerbose, format, a...)
}

// Debugf writes a DEBUG level message to the lvlLog.
func Debugf(format string, a ...interface{}) {
	writef(os.Stderr, lvlDebug, format, a...)
}

// SetLevel explicitly sets the loggerLevel
func SetLevel(l int) {
	loggerLevel = messageLevel(l)
	// set the apex lvlLog level, for umoci
	if loggerLevel <= lvlError {
		// silent option
		apexlog.SetLevel(apexlog.ErrorLevel)
	} else if loggerLevel <= lvlLog {
		// quiet option
		apexlog.SetLevel(apexlog.WarnLevel)
	} else if loggerLevel < lvlDebug {
		// lvlVerbose option(s) or default
		apexlog.SetLevel(apexlog.InfoLevel)
	} else {
		// lvlDebug option
		apexlog.SetLevel(apexlog.DebugLevel)
	}
}

// DisableColor for the logger
func DisableColor() {
	messageColors = map[messageLevel]string{
		lvlFatal: "",
		lvlError: "",
		lvlWarn:  "",
		lvlInfo:  "",
	}
	colorReset = ""
}

// GetLevel returns the current lvlLog level as integer
func GetLevel() int {
	return int(loggerLevel)
}

// GetEnvVar returns a formatted environment variable string which
// can later be interpreted by init() in a child proc
func GetEnvVar() string {
	return fmt.Sprintf("SINGULARITY_MESSAGELEVEL=%d", loggerLevel)
}

// Writer returns an io.Writer to pass to an external packages logging utility.
// i.e when --quiet option is set, this function returns ioutil.Discard writer to ignore output
func Writer() io.Writer {
	if loggerLevel <= -1 {
		return ioutil.Discard
	}

	return os.Stderr
}

// ProgressCallback is a function that provides progress information copying from a Reader to a Writer
type ProgressCallback func(int64, io.Reader, io.Writer) error

// ProgressBarCallback returns a progress bar callback unless e.g. --quiet or lower loglevel is set
func ProgressBarCallback() ProgressCallback {

	if loggerLevel <= -1 {
		return nil
	}

	return func(totalSize int64, r io.Reader, w io.Writer) error {
		p := mpb.New()
		bar := p.AddBar(totalSize,
			mpb.PrependDecorators(
				decor.Counters(decor.UnitKiB, "%.1f / %.1f"),
			),
			mpb.AppendDecorators(
				decor.Percentage(),
				decor.AverageSpeed(decor.UnitKiB, " % .1f "),
				decor.AverageETA(decor.ET_STYLE_GO),
			),
		)

		// create proxy reader
		bodyProgress := bar.ProxyReader(r)

		// Write the body to file
		_, err := io.Copy(w, bodyProgress)
		if err != nil {
			return err
		}

		return nil
	}
}
