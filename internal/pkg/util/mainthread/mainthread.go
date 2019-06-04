// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package mainthread

import (
	"os"
	"path/filepath"
	"syscall"
)

// FuncChannel passes functions executed in main thread
var FuncChannel = make(chan func())

// Execute allows to execute a function in the main thread
func Execute(f func()) {
	done := make(chan bool)
	FuncChannel <- func() {
		f()
		done <- true
	}
	<-done
}

// Stat retrieves file stat information from main thread
func Stat(name string) (fi os.FileInfo, err error) {
	Execute(func() {
		fi, err = os.Stat(name)
	})
	return
}

// Readlink returns the destination of link name from main thread
func Readlink(name string) (dest string, err error) {
	Execute(func() {
		dest, err = os.Readlink(name)
	})
	return
}

// EvalSymlinks returns the evaluated path after link resolution from main thread
func EvalSymlinks(path string) (rpath string, err error) {
	Execute(func() {
		rpath, err = filepath.EvalSymlinks(path)
	})
	return
}

// Chdir changes current working directory to the provided directory
func Chdir(dir string) (err error) {
	Execute(func() {
		err = os.Chdir(dir)
	})
	return
}

// Fchdir changes current working directory to the directory pointed
func Fchdir(fd int) (err error) {
	Execute(func() {
		err = syscall.Fchdir(fd)
	})
	return
}
