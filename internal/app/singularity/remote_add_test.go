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

func createInvalidCfgFile(t *testing.T) string {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("cannot create temporary configuration file for testing: %s\n", err)
	}

	path := f.Name()

	// Set an invalid configuration
	type aDummyStruct struct {
		NoneSenseRemote string
	}
	cfg := aDummyStruct{
		NoneSenseRemote: "toto",
	}

	yaml, err := yaml.Marshal(cfg)
	if err != nil {
		f.Close()
		os.Remove(path)
		t.Fatalf("cannot marshal YANK: %s\n", err)
	}

	f.Write(yaml)
	f.Close()

	return path
}

func createValidCfgFile(t *testing.T) string {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("cannot create temporary configuration file for testing: %s\n", err)
	}

	path := f.Name()

	// Set a valid configuration
	cfg := remote.Config{
		DefaultRemote: "cloud_testing",
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

func TestRemoteAddAndRemove(t *testing.T) {
	validCfgFile := createValidCfgFile(t)
	defer os.Remove(validCfgFile)

	invalidCfgFile := createInvalidCfgFile(t)
	defer os.Remove(invalidCfgFile)

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
			name:       "1: invalid config file; empty remote name; invalid URI, local",
			cfgfile:    invalidCfgFile,
			remoteName: "",
			uri:        invalidURI,
			global:     false,
			shallPass:  true,
		},
		{
			name:       "2: invalid config file; empty remote name; empty URI; local",
			cfgfile:    invalidCfgFile,
			remoteName: "",
			uri:        "",
			global:     false,
			shallPass:  true,
		},
		{
			name:       "3: invalid config file; invalid remote name; invalid URI; local",
			cfgfile:    invalidCfgFile,
			remoteName: invalidRemoteName,
			uri:        invalidURI,
			global:     false,
			shallPass:  true,
		},
		{
			name:       "4: invalid config file; invalid remote name; invalid URI; local",
			cfgfile:    invalidCfgFile,
			remoteName: invalidRemoteName,
			uri:        invalidURI,
			global:     false,
			shallPass:  true,
		},
		{
			name:       "5: valid config file; empty remote name; invalid URI, local",
			cfgfile:    validCfgFile,
			remoteName: "",
			uri:        invalidURI,
			global:     false,
			shallPass:  true,
		},
		{
			name:       "6: valid config file; empty remote name; empty URI; local",
			cfgfile:    validCfgFile,
			remoteName: "",
			uri:        "",
			global:     false,
			shallPass:  true,
		},
		{
			name:       "7: valid config file; invalid remote name; invalid URI; local",
			cfgfile:    validCfgFile,
			remoteName: invalidRemoteName,
			uri:        invalidURI,
			global:     false,
			shallPass:  true,
		},
		{
			name:       "8: valid config gile; invalid remote name; invalid URI; local",
			cfgfile:    validCfgFile,
			remoteName: invalidRemoteName,
			uri:        invalidURI,
			global:     false,
			shallPass:  true,
		},
		{
			// This test checks both RemoteAdd() and RemoteRemove(), we stil have a separate test
			// for corner cases in the context of RemoveRemove().
			name:       "9: valid config file; valid remote name; valid URI; local",
			cfgfile:    validCfgFile,
			remoteName: "cloud_testing",
			uri:        "cloud.random.io",
			global:     false,
			shallPass:  true,
		},
		{
			name:       "10: valid but dummy file; empty remote name; empty URI; local",
			cfgfile:    invalidCfgFile,
			remoteName: "",
			uri:        "",
			global:     false,
			shallPass:  true,
		},
		// Global cases
		{
			name:       "11: invalid config file; empty remote name; invalid URI, global",
			cfgfile:    invalidCfgFile,
			remoteName: "",
			uri:        invalidURI,
			global:     true,
			shallPass:  true,
		},
		{
			name:       "12: invalid config file; empty remote name; empty URI; global",
			cfgfile:    invalidCfgFile,
			remoteName: "",
			uri:        "",
			global:     true,
			shallPass:  true,
		},
		{
			name:       "13: invalid config file; invalid remote name; invalid URI; global",
			cfgfile:    invalidCfgFile,
			remoteName: invalidRemoteName,
			uri:        invalidURI,
			global:     true,
			shallPass:  true,
		},
		{
			name:       "14: invalid config file; invalid remote name; invalid URI; global",
			cfgfile:    invalidCfgFile,
			remoteName: invalidRemoteName,
			uri:        invalidURI,
			global:     true,
			shallPass:  true,
		},
		{
			name:       "15: valid config file; empty remote name; invalid URI, global",
			cfgfile:    validCfgFile,
			remoteName: "",
			uri:        invalidURI,
			global:     true,
			shallPass:  true,
		},
		{
			name:       "16: valid config file; empty remote name; empty URI; global",
			cfgfile:    validCfgFile,
			remoteName: "",
			uri:        "",
			global:     true,
			shallPass:  true,
		},
		{
			name:       "17: valid config file; invalid remote name; invalid URI; global",
			cfgfile:    validCfgFile,
			remoteName: invalidRemoteName,
			uri:        invalidURI,
			global:     true,
			shallPass:  true,
		},
		{
			name:      "18: valid config file; invalid remote name; invalid URI; global",
			cfgfile:   validCfgFile,
			uri:       invalidURI,
			global:    true,
			shallPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RemoteAdd(tt.cfgfile, tt.remoteName, tt.uri, tt.global)
			if tt.shallPass == true && err != nil {
				t.Fatalf("valid case failed: %s\n", err)
			}

			if tt.shallPass == false && err == nil {
				RemoteRemove(tt.cfgfile, tt.remoteName)
				t.Fatal("invalid case passed")
			}

			if tt.shallPass == true && err == nil {
				RemoteRemove(tt.cfgfile, tt.remoteName)
			}
		})
	}
}
