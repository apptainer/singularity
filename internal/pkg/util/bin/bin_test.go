// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package bin

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/buildcfg"
)

func TestCryptsetup(t *testing.T) {
	if buildcfg.CRYPTSETUP_PATH == "" {
		t.Skip("cryptsetup is not available on this platform")
	}

	cases := map[string]struct {
		expectSuccess bool
		config        string
		expectPath    string
	}{
		"buildcfg CRYPTSETUP_PATH": {
			config:        buildcfg.CRYPTSETUP_PATH,
			expectPath:    buildcfg.CRYPTSETUP_PATH,
			expectSuccess: true,
		},
		"buildcfg CRYPTSETUP_PATH dir": {
			config:        filepath.Dir(buildcfg.CRYPTSETUP_PATH),
			expectPath:    buildcfg.CRYPTSETUP_PATH,
			expectSuccess: true,
		},
		"empty config": {
			expectPath:    buildcfg.CRYPTSETUP_PATH,
			expectSuccess: true,
		},
		"arbitrary program in config": {
			config:        "/bin/true",
			expectPath:    "/bin/true",
			expectSuccess: true,
		},
		"invalid path": {
			config:        "/invalid/path",
			expectSuccess: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f, err := ioutil.TempFile("", "test.conf")
			if err != nil {
				t.Fatalf("cannot create temporary test configuration: %+v", err)
			}
			f.Close()
			defer os.Remove(f.Name())

			cfg := fmt.Sprintf("cryptsetup path = %s\n", tc.config)
			ioutil.WriteFile(f.Name(), []byte(cfg), 0644)

			path, err := cryptsetup(f.Name())

			t.Log(path, err)

			switch {
			case tc.expectSuccess && err == nil:
				// expect success, no error, check path
				if path != tc.expectPath {
					t.Errorf("calling cryptsetup with config = %q, expecting %q, got %q",
						cfg, tc.expectPath, path)
				}

			case tc.expectSuccess && err != nil:
				// expect success, got error
				t.Errorf("unexpected error calling cryptsetup with config = %q, err = %+v",
					cfg, err)

			case !tc.expectSuccess && err == nil:
				// expect failure, got no error
				t.Errorf("unexpected result calling cryptsetup with config = %q, got path = %s",
					cfg, path)

			case !tc.expectSuccess && err != nil:
				// expect failure, got error
				t.Logf("got expected failure calling cryptsetup with config = %q, err = %+v",
					cfg, err)
			}
		})
	}
}
