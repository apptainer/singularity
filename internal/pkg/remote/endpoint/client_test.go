// Copyright (c) 2020, Control Command Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package endpoint

import (
	"testing"

	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
)

func init() {
	useragent.InitValue("singularity", "3.0.0")
}

func TestKeyserverClientConfig(t *testing.T) {
	tests := []struct {
		name          string
		endpoint      *Config
		uri           string
		expectedURI   string
		expectSuccess bool
		op            KeyserverOp
	}{
		{
			name: "Sylabs cloud",
			endpoint: &Config{
				URI: SCSDefaultCloudURI,
			},
			uri:           SCSDefaultKeyserverURI,
			expectedURI:   SCSDefaultKeyserverURI,
			expectSuccess: true,
			op:            KeyserverSearchOp,
		},
		{
			name: "Sylabs cloud verify",
			endpoint: &Config{
				URI: SCSDefaultCloudURI,
				Keyservers: []*ServiceConfig{
					{
						URI:  SCSDefaultKeyserverURI,
						Skip: true,
					},
					{
						URI:      "http://localhost:11371",
						External: true,
					},
				},
			},
			uri:           "",
			expectedURI:   "http://localhost:11371",
			expectSuccess: true,
			op:            KeyserverVerifyOp,
		},
		{
			name: "Sylabs cloud search",
			endpoint: &Config{
				URI: SCSDefaultCloudURI,
				Keyservers: []*ServiceConfig{
					{
						URI: SCSDefaultKeyserverURI,
					},
					{
						URI:      "http://localhost:11371",
						External: true,
					},
				},
			},
			uri:           "",
			expectedURI:   SCSDefaultKeyserverURI,
			expectSuccess: true,
			op:            KeyserverSearchOp,
		},
		{
			name: "Custom library",
			endpoint: &Config{
				URI: SCSDefaultCloudURI,
			},
			uri:           "https://custom.keys",
			expectedURI:   "https://custom.keys",
			expectSuccess: true,
			op:            KeyserverVerifyOp,
		},
		{
			name: "Fake cloud",
			endpoint: &Config{
				URI: "cloud.inexistent-xxxx-domain.io",
			},
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := tt.endpoint.KeyserverClientConfig(tt.uri, tt.op)
			if err != nil && tt.expectSuccess {
				t.Errorf("unexpected error: %s", err)
			} else if err == nil && !tt.expectSuccess {
				t.Errorf("unexpected success for %s", tt.name)
			} else if err != nil && !tt.expectSuccess {
				return
			}
			if config.BaseURL != tt.expectedURI {
				t.Errorf("unexpected uri returned: %s instead of %s", config.BaseURL, tt.expectedURI)
			} else if config.AuthToken != "" {
				t.Errorf("unexpected token returned: %s", config.AuthToken)
			}
		})
	}
}

func TestLibraryClientConfig(t *testing.T) {
	tests := []struct {
		name          string
		endpoint      *Config
		uri           string
		expectSuccess bool
	}{
		{
			name: "Sylabs cloud",
			endpoint: &Config{
				URI: SCSDefaultCloudURI,
			},
			uri:           SCSDefaultLibraryURI,
			expectSuccess: true,
		},
		{
			name: "Custom library",
			endpoint: &Config{
				URI: SCSDefaultCloudURI,
			},
			uri:           "https://custom.library",
			expectSuccess: true,
		},
		{
			name: "Fake cloud",
			endpoint: &Config{
				URI: "cloud.inexistent-xxxx-domain.io",
			},
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := tt.endpoint.LibraryClientConfig(tt.uri)
			if err != nil && tt.expectSuccess {
				t.Errorf("unexpected error: %s", err)
			} else if err == nil && !tt.expectSuccess {
				t.Errorf("unexpected success for %s", tt.name)
			} else if err != nil && !tt.expectSuccess {
				return
			}
			if config.BaseURL != tt.uri {
				t.Errorf("unexpected uri returned: %s instead of %s", config.BaseURL, tt.uri)
			} else if config.AuthToken != "" {
				t.Errorf("unexpected token returned: %s", config.AuthToken)
			}
		})
	}
}

func TestBuilderClientConfig(t *testing.T) {
	tests := []struct {
		name          string
		endpoint      *Config
		uri           string
		expectSuccess bool
	}{
		{
			name: "Sylabs cloud",
			endpoint: &Config{
				URI: SCSDefaultCloudURI,
			},
			uri:           SCSDefaultBuilderURI,
			expectSuccess: true,
		},
		{
			name: "Custom builder",
			endpoint: &Config{
				URI: SCSDefaultCloudURI,
			},
			uri:           "https://custom.builder",
			expectSuccess: true,
		},
		{
			name: "Fake cloud",
			endpoint: &Config{
				URI: "https://cloud.fake-domain.io",
			},
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := tt.endpoint.BuilderClientConfig(tt.uri)
			if err != nil && tt.expectSuccess {
				t.Errorf("unexpected error: %s", err)
			} else if err == nil && !tt.expectSuccess {
				t.Errorf("unexpected success for %s", tt.name)
			} else if err != nil && !tt.expectSuccess {
				return
			}
			if config.BaseURL != tt.uri {
				t.Errorf("unexpected uri returned: %s instead of %s", config.BaseURL, tt.uri)
			} else if config.AuthToken != "" {
				t.Errorf("unexpected token returned: %s", config.AuthToken)
			}
		})
	}
}
