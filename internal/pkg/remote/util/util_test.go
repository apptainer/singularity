// Copyright (c) 2020, Control Command Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package util

import (
	"net/url"
	"testing"
)

func TestSameKeyserver(t *testing.T) {
	tests := []struct {
		name        string
		u1          string
		u2          string
		mustBeEqual bool
	}{
		{
			name:        "Identical http",
			u1:          "http://localhost:11371",
			u2:          "http://localhost:11371",
			mustBeEqual: true,
		},
		{
			name:        "Identical https",
			u1:          "https://localhost",
			u2:          "https://localhost",
			mustBeEqual: true,
		},
		{
			name:        "Identical https 8443",
			u1:          "https://localhost:8443",
			u2:          "https://localhost:8443",
			mustBeEqual: true,
		},
		{
			name:        "Identical http and hkp",
			u1:          "http://localhost:11371",
			u2:          "hkp://localhost",
			mustBeEqual: true,
		},
		{
			name:        "Identical https and hkps",
			u1:          "https://localhost",
			u2:          "hkps://localhost",
			mustBeEqual: true,
		},
		{
			name:        "Different https and hkps port",
			u1:          "https://localhost:8443",
			u2:          "hkps://localhost",
			mustBeEqual: false,
		},
		{
			name:        "Different http and hkp port",
			u1:          "http://localhost",
			u2:          "hkp://localhost",
			mustBeEqual: false,
		},
		{
			name:        "Identical http with different path",
			u1:          "http://localhost/path/a",
			u2:          "http://localhost/path/b",
			mustBeEqual: true,
		},
		{
			name:        "Not support scheme first URL",
			u1:          "file://localhost/path/a",
			u2:          "http://localhost/path/b",
			mustBeEqual: false,
		},
		{
			name:        "Not support scheme second URL",
			u1:          "http://localhost/path/a",
			u2:          "file://localhost/path/b",
			mustBeEqual: false,
		},
		{
			name:        "Empty URLs",
			u1:          "",
			u2:          "",
			mustBeEqual: false,
		},
		{
			name:        "Control bytes URLs",
			u1:          "http://localhost\n",
			u2:          "http://localhost\n",
			mustBeEqual: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eq := SameKeyserver(tt.u1, tt.u2)
			if eq && !tt.mustBeEqual {
				t.Errorf("unexpected match between %s and %s", tt.u1, tt.u2)
			} else if !eq && tt.mustBeEqual {
				t.Errorf("unexpected diff between %s and %s", tt.u1, tt.u2)
			}
		})
	}
}

func TestNormalizeKeyserverURI(t *testing.T) {
	tests := []struct {
		name        string
		uri         string
		expectedURI string
		wantErr     bool
	}{
		{
			name:        "Http without port",
			uri:         "http://localhost",
			expectedURI: "http://localhost",
		},
		{
			name:        "Http with port",
			uri:         "http://localhost:11370",
			expectedURI: "http://localhost:11370",
		},
		{
			name:        "Https without port",
			uri:         "https://localhost",
			expectedURI: "https://localhost",
		},
		{
			name:        "Https with port",
			uri:         "https://localhost:8443",
			expectedURI: "https://localhost:8443",
		},
		{
			name:        "HKP form",
			uri:         "hkp://localhost",
			expectedURI: "http://localhost:11371",
		},
		{
			name:        "HKP form with port",
			uri:         "hkp://localhost:11370",
			expectedURI: "http://localhost:11370",
		},
		{
			name:        "HKPS form",
			uri:         "hkps://localhost",
			expectedURI: "https://localhost",
		},
		{
			name:        "HKPS form with port",
			uri:         "hkps://localhost:8443",
			expectedURI: "https://localhost:8443",
		},
		{
			name:    "Empty URI",
			uri:     "",
			wantErr: true,
		},
		{
			name:    "Control bytes URLs",
			uri:     "http://localhost\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri, err := NormalizeKeyserverURI(tt.uri)
			if err == nil && tt.wantErr {
				t.Errorf("unexpected success for URI %s", tt.uri)
			} else if err != nil && !tt.wantErr {
				t.Errorf("unexpected error for URI %s: %s", tt.uri, err)
			} else if !tt.wantErr && uri.String() != tt.expectedURI {
				t.Errorf("unexpected URI returned: got %s instead of %s", uri, tt.expectedURI)
			}
		})
	}
}

func TestEqual(t *testing.T) {
	tests := []struct {
		name        string
		u1          *url.URL
		u2          *url.URL
		mustBeEqual bool
	}{
		{
			name:        "equal OK (scheme)",
			u1:          &url.URL{Host: "localhost:8000", Scheme: "http"},
			u2:          &url.URL{Host: "localhost:8000", Scheme: "http"},
			mustBeEqual: true,
		},
		{
			name:        "equal KO (scheme)",
			u1:          &url.URL{Host: "localhost:8000", Scheme: "http"},
			u2:          &url.URL{Host: "localhost:8000", Scheme: "https"},
			mustBeEqual: false,
		},
		{
			name:        "equal OK (port)",
			u1:          &url.URL{Host: "localhost:8000", Scheme: "http"},
			u2:          &url.URL{Host: "localhost:8000", Scheme: "http"},
			mustBeEqual: true,
		},
		{
			name:        "equal KO (port)",
			u1:          &url.URL{Host: "localhost:8000", Scheme: "http"},
			u2:          &url.URL{Host: "localhost:8001", Scheme: "http"},
			mustBeEqual: false,
		},
		{
			name:        "equal OK (hostname)",
			u1:          &url.URL{Host: "localhost:8000", Scheme: "http"},
			u2:          &url.URL{Host: "localhost:8000", Scheme: "http"},
			mustBeEqual: true,
		},
		{
			name:        "equal KO (hostname)",
			u1:          &url.URL{Host: "localhost:8000", Scheme: "http"},
			u2:          &url.URL{Host: "localhos:8000", Scheme: "http"},
			mustBeEqual: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			equal := Equal(tt.u1, tt.u2)
			if !equal && tt.mustBeEqual {
				t.Errorf("%s and %s are not equal for test %s", tt.u1, tt.u2, tt.name)
			} else if equal && !tt.mustBeEqual {
				t.Errorf("%s and %s are equal for test %s", tt.u1, tt.u2, tt.name)
			}
		})
	}
}

func TestSameURI(t *testing.T) {
	tests := []struct {
		name        string
		u1          string
		u2          string
		mustBeEqual bool
	}{
		{
			name:        "Http scheme",
			u1:          "http://localhost:8080",
			u2:          "http://localhost:8080",
			mustBeEqual: true,
		},
		{
			name:        "Http scheme with path",
			u1:          "http://localhost:8080/a/b",
			u2:          "http://localhost:8080/b/a",
			mustBeEqual: true,
		},
		{
			name:        "Https scheme",
			u1:          "https://localhost",
			u2:          "https://localhost",
			mustBeEqual: true,
		},
		{
			name:        "Different port",
			u1:          "http://localhost:80",
			u2:          "http://localhost:81",
			mustBeEqual: false,
		},
		{
			name:        "Docker scheme",
			u1:          "docker://docker.io",
			u2:          "docker://docker.io",
			mustBeEqual: true,
		},
		{
			name:        "ORAS scheme",
			u1:          "oras://localhost:5000",
			u2:          "oras://localhost:5000",
			mustBeEqual: true,
		},
		{
			name:        "No scheme",
			u1:          "localhost",
			u2:          "localhost",
			mustBeEqual: false,
		},
		{
			name:        "No scheme with port",
			u1:          "localhost:8080",
			u2:          "localhost:8080",
			mustBeEqual: false,
		},
		{
			name:        "Control bytes in first",
			u1:          "http://localhost\n",
			u2:          "",
			mustBeEqual: false,
		},
		{
			name:        "Control bytes in second",
			u1:          "",
			u2:          "http://localhost\n",
			mustBeEqual: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			equal := SameURI(tt.u1, tt.u2)
			if !equal && tt.mustBeEqual {
				t.Errorf("%s and %s are not equal for test %s", tt.u1, tt.u2, tt.name)
			} else if equal && !tt.mustBeEqual {
				t.Errorf("%s and %s are equal for test %s", tt.u1, tt.u2, tt.name)
			}
		})
	}
}
