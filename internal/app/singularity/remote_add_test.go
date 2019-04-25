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
	"github.com/sylabs/singularity/internal/pkg/test"
	"gopkg.in/yaml.v2"
)

const (
	invalidCfgFilePath = "/not/a/real/file"
	invalidRemoteName  = "notacorrectremotename"
	invalidURI         = "really//not/a/URI"
	validURI           = "cloud.random.io"
	validRemoteName    = "cloud_testing"
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
		t.Fatalf("cannot marshal YAML: %s\n", err)
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
		DefaultRemote: validRemoteName,
		Remotes: map[string]*remote.EndPoint{
			"random": {
				URI:   "validURI",
				Token: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.TCYt5XsITJX1CxPCT8yAV-TVkIEq_PbChOMqsLfRoPsnsgw5WEuts01mq-pQy7UJiN5mgRxD-WUcX16dUEMGlv50aqzpqh4Qktb3rk-BuQy72IFLOqV0G_zS245-kronKb78cPN25DGlcTwLtjPAYuNzVBAh4vGHSrQyHUdBBPM",
			},
			"cloud": {
				URI:   "validURI",
				Token: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.TCYt5XsITJX1CxPCT8yAV-TVkIEq_PbChOMqsLfRoPsnsgw5WEuts01mq-pQy7UJiN5mgRxD-WUcX16dUEMGlv50aqzpqh4Qktb3rk-BuQy72IFLOqV0G_zS245-kronKb78cPN25DGlcTwLtjPAYuNzVBAh4vGHSrQyHUdBBPM",
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
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

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
		{
			name:       "1: invalid config file; empty remote name; invalid URI, local",
			cfgfile:    invalidCfgFile,
			remoteName: "",
			uri:        invalidURI,
			global:     false,
			shallPass:  false,
		},
		{
			name:       "2: invalid config file; empty remote name; empty URI; local",
			cfgfile:    invalidCfgFile,
			remoteName: "",
			uri:        "",
			global:     false,
			shallPass:  false,
		},
		{
			name:       "3: invalid config file; invalid remote name; invalid URI; local",
			cfgfile:    invalidCfgFile,
			remoteName: invalidRemoteName,
			uri:        invalidURI,
			global:     false,
			shallPass:  false,
		},
		{
			name:       "4: invalid config file; invalid remote name; empty URI; local",
			cfgfile:    invalidCfgFile,
			remoteName: invalidRemoteName,
			uri:        "",
			global:     false,
			shallPass:  false,
		},
		{
			name:       "5: valid config file; empty remote name; invalid URI, local",
			cfgfile:    validCfgFile,
			remoteName: "",
			uri:        invalidURI,
			global:     false,
			shallPass:  false,
		},
		{
			name:       "6: valid config file; empty remote name; empty URI; local",
			cfgfile:    validCfgFile,
			remoteName: "",
			uri:        "",
			global:     false,
			shallPass:  false,
		},
		{
			name:       "7: valid config file; invalid remote name; empty URI; local",
			cfgfile:    validCfgFile,
			remoteName: invalidRemoteName,
			uri:        "",
			global:     false,
			shallPass:  false,
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
			// This test checks both RemoteAdd() and RemoteRemove(), we stil
			// have a separate test for corner cases in the context of
			// RemoveRemove().
			name:       "9: valid config file; valid remote name; valid URI; local",
			cfgfile:    validCfgFile,
			remoteName: "cloud_testing",
			uri:        "cloud.random.io",
			global:     false,
			shallPass:  true,
		},
		{
			name:       "10: valid config file; valid remote name; empty URI; local",
			cfgfile:    validCfgFile,
			remoteName: "cloud_testing",
			uri:        "",
			global:     false,
			shallPass:  false,
		},
		{
			name:       "11: valid config file; empty remote name; valid URI; local",
			cfgfile:    validCfgFile,
			remoteName: "",
			uri:        "cloud.random.io",
			global:     false,
			shallPass:  false,
		},
		{
			name:       "12: valid config file; valid remote name; invalid URI; local",
			cfgfile:    validCfgFile,
			remoteName: "cloud_testing",
			uri:        invalidURI,
			global:     false,
			shallPass:  true,
		},
		{
			name:       "13: valid config file: invalid remote name; valid URI; local",
			cfgfile:    validCfgFile,
			remoteName: invalidRemoteName,
			uri:        "cloud.random.io",
			global:     false,
			shallPass:  true,
		},
		{
			name:       "14: invalid config file; empty remote name; invalid URI, global",
			cfgfile:    invalidCfgFile,
			remoteName: "",
			uri:        invalidURI,
			global:     true,
			shallPass:  false,
		},
		{
			name:       "15: invalid config file; empty remote name; empty URI; global",
			cfgfile:    invalidCfgFile,
			remoteName: "",
			uri:        "",
			global:     true,
			shallPass:  false,
		},
		{
			name:       "16: invalid config file; invalid remote name; invalid URI; global",
			cfgfile:    invalidCfgFile,
			remoteName: invalidRemoteName,
			uri:        invalidURI,
			global:     true,
			shallPass:  false,
		},
		{
			name:       "17: invalid config file; invalid remote name; invalid URI; global",
			cfgfile:    invalidCfgFile,
			remoteName: invalidRemoteName,
			uri:        "",
			global:     true,
			shallPass:  false,
		},
		{
			name:       "18: valid config file; empty remote name; invalid URI, global",
			cfgfile:    validCfgFile,
			remoteName: "",
			uri:        invalidURI,
			global:     true,
			shallPass:  false,
		},
		{
			name:       "19: valid config file; empty remote name; empty URI; global",
			cfgfile:    validCfgFile,
			remoteName: "",
			uri:        "",
			global:     true,
			shallPass:  false,
		},
		{
			name:       "20: valid config file; invalid remote name; invalid URI; global",
			cfgfile:    validCfgFile,
			remoteName: invalidRemoteName,
			uri:        invalidURI,
			global:     true,
			shallPass:  true,
		},
		{
			name:       "21: valid config file; invalid remote name; invalid URI; global",
			cfgfile:    validCfgFile,
			remoteName: invalidRemoteName,
			uri:        "",
			global:     true,
			shallPass:  false,
		},
		{
			name:       "22: invalid config file path; invalid remote name; invalid URI; local",
			cfgfile:    invalidCfgFilePath,
			remoteName: invalidRemoteName,
			uri:        invalidURI,
			global:     false,
			shallPass:  false,
		},
		{
			name:       "23: invalid config file path; invalid remote name; invalid URI; global",
			cfgfile:    invalidCfgFilePath,
			remoteName: "",
			uri:        invalidURI,
			global:     true,
			shallPass:  false,
		},
		{
			name:       "24: invalid config file path; empty remote name; invalid URI; local",
			cfgfile:    invalidCfgFilePath,
			remoteName: invalidRemoteName,
			uri:        "",
			global:     false,
			shallPass:  false,
		},
		{
			name:       "25: invalid config file path; empty remote name; invalid URI; global",
			cfgfile:    invalidCfgFilePath,
			remoteName: "",
			uri:        "",
			global:     true,
			shallPass:  false,
		},
		{
			name:       "26: valid config file; valid remote name; valid URI; local",
			cfgfile:    validCfgFile,
			remoteName: "cloud_testing",
			uri:        "cloud.random.io",
			global:     true,
			shallPass:  true,
		},
		{
			name:       "27: valid config file; valid remote name; empty URI; local",
			cfgfile:    validCfgFile,
			remoteName: "cloud_testing",
			uri:        "",
			global:     true,
			shallPass:  false,
		},
		{
			name:       "28: valid config file; empty remote name; valid URI; local",
			cfgfile:    validCfgFile,
			remoteName: "",
			uri:        validURI,
			global:     true,
			shallPass:  false,
		},
		{
			name:       "29: valid config file; valid remote name; invalid URI; local",
			cfgfile:    validCfgFile,
			remoteName: "cloud_testing",
			uri:        invalidURI,
			global:     true,
			shallPass:  true,
		},
		{
			name:       "30: valid config file: invalid remote name; valid URI; local",
			cfgfile:    validCfgFile,
			remoteName: invalidRemoteName,
			uri:        "cloud.random.io",
			global:     true,
			shallPass:  true,
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
