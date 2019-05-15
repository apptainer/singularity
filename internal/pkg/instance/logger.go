// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package instance

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	// BasicLogFormat represents basic log format.
	BasicLogFormat = "basic"
	// KubernetesLogFormat represents kubernetes log format.
	KubernetesLogFormat = "kubernetes"
	// JSONLogFormat represents JSON log format.
	JSONLogFormat = "json"
)

// LogFormatter implements a log formatter.
type LogFormatter func(stream string, data string) string

func kubernetesLogFormatter(stream, data string) string {
	return fmt.Sprintf("%s %s F %s\n", time.Now().Format(time.RFC3339Nano), stream, data)
}

func jsonLogFormatter(stream, data string) string {
	return fmt.Sprintf("{\"time\":\"%s\",\"stream\":\"%s\",\"log\":\"%s\"}\n", time.Now().Format(time.RFC3339Nano), stream, data)
}

func basicLogFormatter(stream, data string) string {
	if stream != "" {
		return fmt.Sprintf("%s %s %s\n", time.Now().Format(time.RFC3339Nano), stream, data)
	}
	return fmt.Sprintf("%s %s\n", time.Now().Format(time.RFC3339Nano), data)
}

type closer func()

// LogFormats contains supported log format by default.
var LogFormats = map[string]LogFormatter{
	BasicLogFormat:      basicLogFormatter,
	KubernetesLogFormat: kubernetesLogFormatter,
	JSONLogFormat:       jsonLogFormatter,
}

// Logger defines a file logger.
type Logger struct {
	file      *os.File
	filename  string
	formatter LogFormatter
	closers   []closer
	sync.Mutex
}

// NewLogger instantiates a new logger with formatter and return it.
func NewLogger(logPath string, formatter LogFormatter) (*Logger, error) {
	logger := &Logger{
		formatter: formatter,
		closers:   make([]closer, 0),
	}

	if logger.formatter == nil {
		logger.formatter = basicLogFormatter
	}

	if err := logger.openFile(logPath); err != nil {
		return nil, err
	}

	return logger, nil
}

func (l *Logger) openFile(path string) (err error) {
	oldmask := syscall.Umask(0)
	defer syscall.Umask(oldmask)

	l.Lock()
	l.file, err = os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	l.filename = l.file.Name()
	l.Unlock()

	return err
}

func (l *Logger) scanOutput(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		return i + 1, data[0 : i+1], nil
	}

	if atEOF {
		return len(data), data, nil
	}

	return 0, nil, nil
}

// NewWriter create a new pipe pair for corresponding stream.
func (l *Logger) NewWriter(stream string, dropCRNL bool) *io.PipeWriter {
	reader, writer := io.Pipe()
	closer := l.scan(stream, reader, writer, dropCRNL)
	l.Lock()
	l.closers = append(l.closers, closer)
	l.Unlock()
	return writer
}

func (l *Logger) scan(stream string, pr *io.PipeReader, pw *io.PipeWriter, dropCRNL bool) closer {
	r := strings.NewReplacer("\r", "\\r", "\n", "\\n")
	scanner := bufio.NewScanner(pr)
	if !dropCRNL {
		scanner.Split(l.scanOutput)
	}

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		for scanner.Scan() {
			l.Lock()
			if !dropCRNL {
				fmt.Fprint(l.file, l.formatter(stream, r.Replace(scanner.Text())))
			} else {
				fmt.Fprint(l.file, l.formatter(stream, scanner.Text()))
			}
			l.Unlock()
		}
		pr.Close()
		wg.Done()
	}()

	// closer function
	return func() {
		pw.Close()
		wg.Wait()
	}
}

// Close closes all pipe pairs created with NewWriter and also closes
// log file descriptor.
func (l *Logger) Close() {
	// closer will terminate scan goroutines spawned with NewWriter
	// by closing pipe write end
	for _, closer := range l.closers {
		closer()
	}

	l.Lock()
	l.closers = nil
	l.file.Sync()
	l.file.Close()
	l.Unlock()
}

// ReOpenFile closes and re-open log file (eg: log rotation).
func (l *Logger) ReOpenFile() {
	l.Lock()
	l.file.Sync()
	l.file.Close()
	l.Unlock()

	// open log file
	l.openFile(l.filename)
}
