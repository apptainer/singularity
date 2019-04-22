// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/remote"
	"gopkg.in/yaml.v2"
)

const (
	invalidCfgFile    = "/not/a/real/file"
	invalidRemoteName = "notacorrectremotename"
	invalidURI        = "really//not/a/URI"
)

func createValidCfgFile(t *testing.T) string {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("cannot create temporary configuration file for testing: %s\n", err)
	}

	path := f.Name()

	// Set a valid configuration
	cfg := remote.Config{
		DefaultRemote: "cloud",
		Remotes: map[string]*remote.EndPoint{
			"random": {
				URI:   "cloud.random.io",
				Token: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxM     jM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.TCY     t5XsITJX1CxPCT8yAV-TVkIEq_PbChOMqsLfRoPsnsgw5WEuts01mq-pQy7UJiN5mgRxD-WUcX16dUEMGl     v50aqzpqh4Qktb3rk-BuQy72IFLOqV0G_zS245-kronKb78cPN25DGlcTwLtjPAYuNzVBAh4vGHSrQyHUd     BBPM",
			},
			"cloud": {
				URI:   "cloud.sylabs.io",
				Token: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxM     jM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.TCY     t5XsITJX1CxPCT8yAV-TVkIEq_PbChOMqsLfRoPsnsgw5WEuts01mq-pQy7UJiN5mgRxD-WUcX16dUEMGl     v50aqzpqh4Qktb3rk-BuQy72IFLOqV0G_zS245-kronKb78cPN25DGlcTwLtjPAYuNzVBAh4vGHSrQyHUd     BBPM",
			},
		},
	}

	yaml, err := yaml.Marshal(cfg)
	if err != nil {
		f.Close()
		os.Remove(path)
		t.Fatalf("cannot marshal YAML: %s\n", err)
	}

	f.Write(yaml)
	f.Close()

	return path
}

func TestRemoteAdd(t *testing.T) {
	validCfgFile := createValidCfgFile(t)
	defer os.Remove(validCfgFile)

	tests := []struct {
		name       string
		cfgfile    string
		remoteName string
		uri        string
		global     bool
		shallPass  bool
	}{
		// Local cases
		{
			name:       "invalid config file; empty remote name; invalid URI, local",
			cfgfile:    invalidCfgFile,
			remoteName: "",
			uri:        invalidURI,
			global:     false,
			shallPass:  false,
		},
		{
			name:       "invalid config file; empty remote name; empty URI; local",
			cfgfile:    invalidCfgFile,
			remoteName: "",
			uri:        "",
			global:     false,
			shallPass:  false,
		},
		{
			name:       "invalid config file; invalid remote name; invalid URI; local",
			cfgfile:    invalidCfgFile,
			remoteName: invalidRemoteName,
			uri:        invalidURI,
			global:     false,
			shallPass:  false,
		},
		{
			name:       "invalid config file; invalid remote name; invalid URI; local",
			cfgfile:    invalidCfgFile,
			remoteName: invalidRemoteName,
			uri:        invalidURI,
			global:     false,
			shallPass:  false,
		},
		{
			name:       "valid config file; empty remote name; invalid URI, local",
			cfgfile:    validCfgFile,
			remoteName: "",
			uri:        invalidURI,
			global:     false,
			shallPass:  true,
		},
		{
			name:       "valid config file; empty remote name; empty URI; local",
			cfgfile:    validCfgFile,
			remoteName: "",
			uri:        "",
			global:     false,
			shallPass:  false,
		},
		{
			name:       "valid config file; invalid remote name; invalid URI; local",
			cfgfile:    validCfgFile,
			remoteName: invalidRemoteName,
			uri:        invalidURI,
			global:     false,
			shallPass:  true,
		},
		{
			name:      "valid config gile; invalid remote name; invalid URI; local",
			cfgfile:   validCfgFile,
			uri:       invalidURI,
			global:    false,
			shallPass: false,
		},
		// Global cases
		{
			name:       "invalid config file; empty remote name; invalid URI, global",
			cfgfile:    invalidCfgFile,
			remoteName: "",
			uri:        invalidURI,
			global:     true,
			shallPass:  false,
		},
		{
			name:       "invalid config file; empty remote name; empty URI; global",
			cfgfile:    invalidCfgFile,
			remoteName: "",
			uri:        "",
			global:     true,
			shallPass:  false,
		},
		{
			name:       "invalid config file; invalid remote name; invalid URI; global",
			cfgfile:    invalidCfgFile,
			remoteName: invalidRemoteName,
			uri:        invalidURI,
			global:     true,
			shallPass:  false,
		},
		{
			name:       "invalid config file; invalid remote name; invalid URI; global",
			cfgfile:    invalidCfgFile,
			remoteName: invalidRemoteName,
			uri:        invalidURI,
			global:     true,
			shallPass:  false,
		},
		{
			name:       "valid config file; empty remote name; invalid URI, global",
			cfgfile:    validCfgFile,
			remoteName: "",
			uri:        invalidURI,
			global:     true,
			shallPass:  false,
		},
		{
			name:       "valid config file; empty remote name; empty URI; global",
			cfgfile:    validCfgFile,
			remoteName: "",
			uri:        "",
			global:     true,
			shallPass:  false,
		},
		{
			name:       "valid config file; invalid remote name; invalid URI; global",
			cfgfile:    validCfgFile,
			remoteName: invalidRemoteName,
			uri:        invalidURI,
			global:     true,
			shallPass:  false,
		},
		{
			name:      "valid config gile; invalid remote name; invalid URI; global",
			cfgfile:   validCfgFile,
			uri:       invalidURI,
			global:    true,
			shallPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RemoteAdd(tt.cfgfile, tt.remoteName, tt.uri, tt.global)
			if tt.shallPass == true && err != nil {
				t.Fatalf("valid case failed: %s\n", err)
			}

			if tt.shallPass == false && err == nil {
				t.Fatal("invalid case passed")
			}
		})
	}
}
