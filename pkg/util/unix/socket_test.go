// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package unix

import (
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sylabs/singularity/internal/pkg/test"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func TestCreateSocket(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	paths := []string{
		filepath.Join(os.TempDir(), randSeq(10), "socket"),  // short path
		filepath.Join(os.TempDir(), randSeq(100), "socket"), // long path
	}

	for _, path := range paths {
		syncCh := make(chan bool, 1)

		dir := filepath.Dir(path)
		os.Mkdir(dir, 0700)

		defer os.RemoveAll(dir)

		// create socket and returns listener
		ln, err := CreateSocket(path)
		if err != nil {
			t.Fatal(err)
		}

		// accept connection and test if received data are correct
		go func() {
			buf := make([]byte, 4)

			conn, err := ln.Accept()
			if err != nil {
				t.Error(err)
			}
			n, err := conn.Read(buf)
			if err != nil {
				t.Error(err)
			}
			if n != 4 {
				t.Error()
			}
			if string(buf) != "send" {
				t.Errorf("unexpected data returned: %s", string(buf))
			}
			syncCh <- true
		}()

		// open / write / close to socket path
		if err := WriteSocket(path, []byte("send")); err != nil {
			t.Error(err)
		}

		<-syncCh

		// close socket implies to delete file automatically
		os.Chdir(dir)
		ln.Close()

		// socket file is deleted by net package at close
		if err := os.Remove(path); err == nil {
			t.Errorf("unexpected success during socket removal")
		}
	}
}
