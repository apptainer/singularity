// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package copy

import (
	"io"
	"sync"
)

// MultiWriter creates a writer that duplicates its writes to all the provided writers,
// writers can be added / removed dynamically.
type MultiWriter struct {
	mutex   sync.Mutex
	writers []io.Writer
}

// Write implements the standard Write interface to duplicate data to all writers.
func (mw *MultiWriter) Write(p []byte) (n int, err error) {
	mw.mutex.Lock()
	defer mw.mutex.Unlock()

	l := len(p)

	for _, w := range mw.writers {
		n, err = w.Write(p)
		if err != nil {
			return
		}
		if n != l {
			err = io.ErrShortWrite
			return
		}
	}

	return l, nil
}

// Add adds a writer.
func (mw *MultiWriter) Add(writer io.Writer) {
	if writer == nil {
		return
	}
	mw.mutex.Lock()
	mw.writers = append(mw.writers, writer)
	mw.mutex.Unlock()
}

// Del removes a writer.
func (mw *MultiWriter) Del(writer io.Writer) {
	mw.mutex.Lock()
	for i, w := range mw.writers {
		if writer == w {
			mw.writers = append(mw.writers[:i], mw.writers[i+1:]...)
			break
		}
	}
	mw.mutex.Unlock()
}
