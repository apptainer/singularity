// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"io/ioutil"
	"math/rand"
	"os"
	"os/user"
	"testing"
	"time"
)

func TestGetConfDir(t *testing.T) {
	usr, _ := user.Current()
	confDir := getConfDir(usr.Username)
	hd, _ := os.UserHomeDir()
	exptDir := hd + "/.singularity" // test currently assumes ~/.singularity
	if confDir != exptDir {
		t.Errorf("expected %s, got %s", exptDir, confDir)
	}
}

func TestCreateConfDir(t *testing.T) {
	// create a random name for a directory
	rand.Seed(time.Now().UnixNano())
	bytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		bytes[i] = byte(65 + rand.Intn(25))
	}
	dir := "/tmp/" + string(bytes)

	// create the directory and check that it exists
	createConfDir(dir)
	defer os.RemoveAll(dir)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("failed to create directory %s", dir)
	} else {
		// stick something in the directory and make sure it isn't deleted
		ioutil.WriteFile(dir+"/foo", []byte(""), 655)
		createConfDir(dir)
		if _, err := os.Stat(dir + "/foo"); os.IsNotExist(err) {
			t.Errorf("inadvertantly overwrote existing directory %s", dir)
		}
	}
}
