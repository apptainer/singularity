// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package home

import (
	_ "github.com/singularityware/singularity/src/pkg/util/user"
)

/*
const testroot = "homedirs"

var homeTests = []struct {
	src        string
	dest       string
	srcResult  string
	destResult string
	succeeds   bool
}{
	{"/path", "/path", "/path", "/path", true},
}

func TestAddDefault(t *testing.T) {
}

func TestAddCustom(t *testing.T) {
	root := setupDirs()
	defer os.RemoveAll(root)

	for _, home := range homeTests {
		if filepath.IsAbs(home.src) {
			home.src = filepath.Join(root, home.src)
		} else {
			home.src = filepath.Join(testroot, home.src)
		}

		t.Logf("home.src: %v", home.src)

		if err := os.Mkdir(home.src, 0777); err != nil {
			t.Error(err)
		}

	}

	points := &mount.Points{}

}

func setupDirs() string {
	wd, err := os.Getwd()
	root := filepath.Join(wd, testroot)
	os.Mkdir(root, 0777)

	return root
}
*/
