// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

/*
func TestShub(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	tests := []struct {
		name        string
		dir         string
		needCleanup bool
		expected    string
	}{
		{
			name:        "Default Shub",
			dir:         "",
			needCleanup: false, // Never clean up the default cache
			expected:    filepath.Join(cacheDefault, "shub"),
		},
		{
			name:        "Custom Shub",
			dir:         cacheCustom,
			needCleanup: true,
			expected:    filepath.Join(expectedCacheCustomRoot, "shub"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewHandle(tt.dir)
			if err != nil {
				t.Fatalf("failed to create new image cache handle: %s", err)
			}
			if tt.needCleanup {
				defer c.cleanAllCaches()
			}

			if c.Shub != tt.expected {
				t.Errorf("Unexpected result for %s: %s (expected %s)", tt.name, c.Shub, tt.expected)
			}
		})
	}
}

func TestShubImageExists(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	tempImageCache, err := ioutil.TempDir("", "image-cache-")
	if err != nil {
		t.Fatal("failed to create temporary image cache directory:", err)
	}
	os.RemoveAll(tempImageCache)

	c, err := NewHandle(tempImageCache)
	if err != nil {
		t.Fatalf("failed to create a new image cache handle: %s", err)
	}

	tests := []struct {
		name     string
		sum      string
		path     string
		expected bool
	}{
		{
			name:     "valid data",
			sum:      validSHASum,
			path:     validPath,
			expected: true,
		},
		{
			name:     "invalid SHA sum; valid path",
			sum:      invalidSHASum,
			path:     validPath,
			expected: true,
		},
		{
			name:     "valid SHA sum; invalid path",
			sum:      validSHASum,
			path:     invalidPath,
			expected: false,
		},
		{
			name:     "invalid data",
			sum:      invalidSHASum,
			path:     invalidPath,
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			exists, err := c.ShubImageExists(test.sum, test.path)
			if err != nil {
				t.Fatal("ShubImageExists() failed")
			}
			if test.expected == false && exists == true {
				t.Fatal("NetImageExists() did not fail for an invalid case")
			}
			if test.expected == true && exists == false {
				t.Fatal("NetImageExists() failed while expected to succeed")
			}
		})
	}
}
*/
