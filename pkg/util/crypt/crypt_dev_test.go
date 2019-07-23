// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package crypt

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestEncrypt(t *testing.T) {
	dev := &Device{}

	emptyFile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("failed to create temporary file: %s", err)
	}
	emptyFile.Close()
	defer os.Remove(emptyFile.Name())

	tests := []struct {
		name      string
		path      string
		key       []byte
		shallPass bool
	}{
		{
			name:      "empty path",
			path:      "",
			key:       []byte("dummyKey"),
			shallPass: false,
		},
		{
			name:      "empty file",
			path:      emptyFile.Name(),
			key:       []byte("dummyKey"),
			shallPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := dev.EncryptFilesystem(tt.path, tt.key)
			if tt.shallPass && err != nil {
				t.Fatalf("test %s expected to succeed but failed: %s", tt.name, err)
			}
			if !tt.shallPass && err == nil {
				t.Fatalf("test %s expected to fail but succeeded", tt.name)
			}
		})
	}
}
