// Copyright (c) 2020, Control Command Inc. All rights reserved.
// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package remote

import (
	"bytes"
	"os"
	"reflect"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/remote/endpoint"
	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
	yaml "gopkg.in/yaml.v2"
)

const testToken = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.TCYt5XsITJX1CxPCT8yAV-TVkIEq_PbChOMqsLfRoPsnsgw5WEuts01mq-pQy7UJiN5mgRxD-WUcX16dUEMGlv50aqzpqh4Qktb3rk-BuQy72IFLOqV0G_zS245-kronKb78cPN25DGlcTwLtjPAYuNzVBAh4vGHSrQyHUdBBPM"

//NOTE: VerifyToken() cannot be fully tested unless we have a dummy token for the token service to authenticate, so we basically only test a few error cases.
func TestVerifyToken(t *testing.T) {
	ep := new(endpoint.Config)

	err := ep.VerifyToken("")
	if err == nil {
		t.Fatal("VerifyToken() succeeded with an empty endpoint")
	}
}

func TestMain(m *testing.M) {
	useragent.InitValue("singularity", "3.0.0-alpha.1-303-gaed8d30-dirty")

	os.Exit(m.Run())
}

type writeReadTest struct {
	name string
	c    Config
}

type aDummyData struct {
	NoneSenseRemote string
}

func TestWriteToReadFrom(t *testing.T) {
	testsPass := []writeReadTest{
		{
			name: "empty config",
			c: Config{
				DefaultRemote: "",
				Remotes:       map[string]*endpoint.Config{},
			},
		},
		{
			name: "config with stuff",
			c: Config{
				DefaultRemote: "cloud",
				Remotes: map[string]*endpoint.Config{
					"random": {
						URI:   "cloud.random.io",
						Token: testToken,
					},
					"cloud": {
						URI:   "cloud.sylabs.io",
						Token: testToken,
					},
				},
			},
		},
	}

	testsFail := []struct {
		name string
		data aDummyData
	}{
		{
			name: "invalid data",
			data: aDummyData{NoneSenseRemote: "toto"},
		},
	}

	for _, test := range testsPass {
		t.Run(test.name, func(t *testing.T) {
			var r bytes.Buffer

			test.c.WriteTo(&r)

			new, err := ReadFrom(&r)
			if err != nil {
				t.Errorf("unexpected failure running %s test: %s", test.name, err)
			}

			if !reflect.DeepEqual(test.c, *new) {
				t.Errorf("failed to read/write config:\n\thave: %v\n\twant: %v", test.c, *new)
			}
		})
	}

	for _, test := range testsFail {
		t.Run(test.name, func(t *testing.T) {
			var r bytes.Buffer

			yaml, err := yaml.Marshal(test.data)
			if err != nil {
				t.Fatalf("cannot mashal YAML: %s\n", err)
			}

			_, err = r.Write(yaml)
			if err != nil {
				t.Fatalf("failed to write YAML data")
			}

			_, err = ReadFrom(&r)
			if err == nil {
				t.Fatal("reading an invalid YAML file succeeded")
			}
		})
	}

	t.Run("empty config no data", func(t *testing.T) {
		var r bytes.Buffer

		_, err := ReadFrom(&r)

		if err != nil {
			t.Errorf("unexpected failure running %s test: %s", t.Name(), err)
		}

	})
}

type syncTest struct {
	name string
	sys  Config // sys Input
	usr  Config // usr Input
	res  Config // res Output
}

func TestSyncFrom(t *testing.T) {
	testsPass := []syncTest{
		{
			name: "empty sys config",
			usr: Config{
				Remotes: map[string]*endpoint.Config{
					"sylabs": {
						URI:   "cloud.sylabs.io",
						Token: "fake-token",
					},
				},
			},
			res: Config{
				Remotes: map[string]*endpoint.Config{
					"sylabs": {
						URI:   "cloud.sylabs.io",
						Token: "fake-token",
					},
				},
			},
		}, {
			name: "sys config new endpoint",
			sys: Config{
				Remotes: map[string]*endpoint.Config{
					"sylabs-global": {
						URI:   "cloud.sylabs.io",
						Token: "fake-token", // should be ignored by SyncFrom
					},
				},
			},
			usr: Config{
				Remotes: map[string]*endpoint.Config{
					"sylabs": {
						URI:   "cloud.sylabs.io",
						Token: "fake-token",
					},
				},
			},
			res: Config{
				Remotes: map[string]*endpoint.Config{
					"sylabs-global": {
						URI:    "cloud.sylabs.io",
						System: true,
					},
					"sylabs": {
						URI:   "cloud.sylabs.io",
						Token: "fake-token",
					},
				},
			},
		}, {
			name: "sys config existing endpoint",
			sys: Config{
				Remotes: map[string]*endpoint.Config{
					"sylabs-global": {
						URI:   "cloud.sylabs.io",
						Token: "fake-token", // should be ignored by SyncFrom
					},
				},
			},
			usr: Config{
				Remotes: map[string]*endpoint.Config{
					"sylabs-global": {
						URI:    "cloud.sylabs.io",
						System: true,
					},
					"sylabs": {
						URI:   "cloud.sylabs.io",
						Token: "fake-token",
					},
				},
			},
			res: Config{
				Remotes: map[string]*endpoint.Config{
					"sylabs-global": {
						URI:    "cloud.sylabs.io",
						System: true,
					},
					"sylabs": {
						URI:   "cloud.sylabs.io",
						Token: "fake-token",
					},
				},
			},
		}, {
			name: "sys config update existing endpoint",
			sys: Config{
				Remotes: map[string]*endpoint.Config{
					"sylabs-global": {
						URI:   "cloud.sylabs.io",
						Token: "fake-token", // should be ignored by SyncFrom
					},
				},
			},
			usr: Config{
				Remotes: map[string]*endpoint.Config{
					"sylabs-global": {
						URI:    "cloud.old-url.io",
						System: true,
					},
					"sylabs": {
						URI:   "cloud.sylabs.io",
						Token: "fake-token",
					},
				},
			},
			res: Config{
				Remotes: map[string]*endpoint.Config{
					"sylabs-global": {
						URI:    "cloud.sylabs.io",
						System: true,
					},
					"sylabs": {
						URI:   "cloud.sylabs.io",
						Token: "fake-token",
					},
				},
			},
		}, {
			name: "sys config update default endpoint",
			sys: Config{
				DefaultRemote: "sylabs-global",
				Remotes: map[string]*endpoint.Config{
					"sylabs-global": {
						URI:   "cloud.sylabs.io",
						Token: "fake-token", // should be ignored by SyncFrom
					},
				},
			},
			usr: Config{
				Remotes: map[string]*endpoint.Config{
					"sylabs-global": {
						URI:    "cloud.old-url.io",
						System: true,
					},
					"sylabs": {
						URI:   "cloud.sylabs.io",
						Token: "fake-token",
					},
				},
			},
			res: Config{
				DefaultRemote: "sylabs-global",
				Remotes: map[string]*endpoint.Config{
					"sylabs-global": {
						URI:    "cloud.sylabs.io",
						System: true,
					},
					"sylabs": {
						URI:   "cloud.sylabs.io",
						Token: "fake-token",
					},
				},
			},
		}, {
			name: "sys config dont update default endpoint",
			sys: Config{
				DefaultRemote: "sylabs-global",
				Remotes: map[string]*endpoint.Config{
					"sylabs-global": {
						URI:   "cloud.sylabs.io",
						Token: "fake-token", // should be ignored by SyncFrom
					},
				},
			},
			usr: Config{
				DefaultRemote: "sylabs",
				Remotes: map[string]*endpoint.Config{
					"sylabs-global": {
						URI:    "cloud.old-url.io",
						System: true,
					},
					"sylabs": {
						URI:   "cloud.sylabs.io",
						Token: "fake-token",
					},
				},
			},
			res: Config{
				DefaultRemote: "sylabs",
				Remotes: map[string]*endpoint.Config{
					"sylabs-global": {
						URI:    "cloud.sylabs.io",
						System: true,
					},
					"sylabs": {
						URI:   "cloud.sylabs.io",
						Token: "fake-token",
					},
				},
			},
		},
	}

	for _, test := range testsPass {
		t.Run(test.name, func(t *testing.T) {
			if err := test.usr.SyncFrom(&test.sys); err != nil {
				t.Error("failed to sync from sys")
			}

			if !reflect.DeepEqual(test.usr, test.res) {
				t.Errorf("bad sync from sys:\n\thave: %v\n\twant: %v", test.usr, test.res)
			}
		})
	}

	testsFail := []syncTest{
		{
			name: "sys endpoint collision",
			sys: Config{
				Remotes: map[string]*endpoint.Config{
					"sylabs-global": {
						URI:   "cloud.sylabs.io",
						Token: "fake-token",
					},
				},
			},
			usr: Config{
				Remotes: map[string]*endpoint.Config{
					"sylabs": {
						URI:   "cloud.sylabs.io",
						Token: "fake-token",
					},
					"sylabs-global": {
						URI: "cloud.sylabs.io",
					},
				},
			},
		},
	}

	for _, test := range testsFail {
		t.Run(test.name, func(t *testing.T) {
			if err := test.usr.SyncFrom(&test.sys); err == nil {
				t.Error("unexpected success calling SyncFrom")
			}
		})
	}
}

type remoteTest struct {
	name  string
	old   Config
	new   Config
	id    string
	newID string
	ep    *endpoint.Config
}

func TestAddRemote(t *testing.T) {
	testsPass := []remoteTest{
		{
			name: "add remote to empty config",
			old: Config{
				Remotes: map[string]*endpoint.Config{},
			},
			new: Config{
				DefaultRemote: "",
				Remotes: map[string]*endpoint.Config{
					"cloud": {
						URI:   "cloud.sylabs.io",
						Token: testToken,
					},
				},
			},
			id: "cloud",
			ep: &endpoint.Config{
				URI:   "cloud.sylabs.io",
				Token: testToken,
			},
		},
		{
			name: "add remote to non-empty config",
			old: Config{
				DefaultRemote: "",
				Remotes: map[string]*endpoint.Config{
					"random": {
						URI:   "cloud.random.io",
						Token: testToken,
					},
				},
			},
			new: Config{
				DefaultRemote: "",
				Remotes: map[string]*endpoint.Config{
					"random": {
						URI:   "cloud.random.io",
						Token: testToken,
					},
					"cloud": {
						URI:   "cloud.sylabs.io",
						Token: testToken,
					},
				},
			},
			id: "cloud",
			ep: &endpoint.Config{
				URI:   "cloud.sylabs.io",
				Token: testToken,
			},
		},
	}

	for _, test := range testsPass {
		t.Run(test.name, func(t *testing.T) {
			if err := test.old.Add(test.id, test.ep); err != nil {
				t.Error("failed to add endpoint to config")
			}

			if !reflect.DeepEqual(test.old, test.new) {
				t.Errorf("Add failed to set config:\n\thave: %v\n\twant: %v", test.old, test.new)
			}
		})
	}

	testFail := remoteTest{
		name: "add already existing remote",
		old: Config{
			DefaultRemote: "",
			Remotes: map[string]*endpoint.Config{
				"cloud": {
					URI:   "cloud.sylabs.io",
					Token: testToken,
				},
			},
		},
		id: "cloud",
		ep: &endpoint.Config{
			URI:   "cloud.sylabs.io",
			Token: testToken,
		},
	}

	t.Run(testFail.name, func(t *testing.T) {
		if err := testFail.old.Add(testFail.id, testFail.ep); err == nil {
			t.Error("unexpected success adding already existing remote")
		}
	})
}

func TestRemoveRemote(t *testing.T) {
	testsPass := []remoteTest{
		{
			name: "remove remote to make empty config",
			old: Config{
				DefaultRemote: "",
				Remotes: map[string]*endpoint.Config{
					"cloud": {
						URI:   "cloud.sylabs.io",
						Token: testToken,
					},
				},
			},
			new: Config{
				Remotes: map[string]*endpoint.Config{},
			},
			id: "cloud",
		},
		{
			name: "remove remote to make non-empty config",
			old: Config{
				DefaultRemote: "",
				Remotes: map[string]*endpoint.Config{
					"random": {
						URI:   "cloud.random.io",
						Token: testToken,
					},
					"cloud": {
						URI:   "cloud.sylabs.io",
						Token: testToken,
					},
				},
			},
			new: Config{
				DefaultRemote: "",
				Remotes: map[string]*endpoint.Config{
					"random": {
						URI:   "cloud.random.io",
						Token: testToken,
					},
				},
			},
			id: "cloud",
		},
		{
			name: "remove default remote to make defaultless config",
			old: Config{
				DefaultRemote: "cloud",
				Remotes: map[string]*endpoint.Config{
					"random": {
						URI:   "cloud.random.io",
						Token: testToken,
					},
					"cloud": {
						URI:   "cloud.sylabs.io",
						Token: testToken,
					},
				},
			},
			new: Config{
				DefaultRemote: "",
				Remotes: map[string]*endpoint.Config{
					"random": {
						URI:   "cloud.random.io",
						Token: testToken,
					},
				},
			},
			id: "cloud",
		},
	}

	for _, test := range testsPass {
		t.Run(test.name, func(t *testing.T) {
			if err := test.old.Remove(test.id); err != nil {
				t.Error("failed to remove endpoint from config")
			}

			if !reflect.DeepEqual(test.old, test.new) {
				t.Errorf("Remove failed to set config:\n\thave: %v\n\twant: %v", test.old, test.new)
			}
		})
	}

	testFail := remoteTest{
		name: "remove non-existent remote",
		old: Config{
			DefaultRemote: "",
			Remotes:       map[string]*endpoint.Config{},
		},
		id: "cloud",
	}

	t.Run(testFail.name, func(t *testing.T) {
		if err := testFail.old.Remove(testFail.id); err == nil {
			t.Error("unexpected success removing non-existent remote")
		}
	})
}

func TestRenameRemote(t *testing.T) {
	testsPass := []remoteTest{
		{
			name: "rename remote not default",
			old: Config{
				DefaultRemote: "",
				Remotes: map[string]*endpoint.Config{
					"cloud": {
						URI:   "cloud.sylabs.io",
						Token: testToken,
					},
				},
			},
			new: Config{
				DefaultRemote: "",
				Remotes: map[string]*endpoint.Config{
					"newCloud": {
						URI:   "cloud.sylabs.io",
						Token: testToken,
					},
				},
			},
			id:    "cloud",
			newID: "newCloud",
		},
		{
			name: "rename remote when it's default",
			old: Config{
				DefaultRemote: "cloud",
				Remotes: map[string]*endpoint.Config{
					"random": {
						URI:   "cloud.random.io",
						Token: testToken,
					},
					"cloud": {
						URI:   "cloud.sylabs.io",
						Token: testToken,
					},
				},
			},
			new: Config{
				DefaultRemote: "newCloud",
				Remotes: map[string]*endpoint.Config{
					"random": {
						URI:   "cloud.random.io",
						Token: testToken,
					},
					"newCloud": {
						URI:   "cloud.sylabs.io",
						Token: testToken,
					},
				},
			},
			id:    "cloud",
			newID: "newCloud",
		},
	}

	for _, test := range testsPass {
		t.Run(test.name, func(t *testing.T) {
			if err := test.old.Rename(test.id, test.newID); err != nil {
				t.Error("failed to rename endpoint from config")
			}

			if !reflect.DeepEqual(test.old, test.new) {
				t.Errorf("Rename failed to set config:\n\thave: %v\n\twant: %v", test.old, test.new)
			}
		})
	}

	testsFail := []remoteTest{
		{
			name: "rename non-existent remote",
			old: Config{
				DefaultRemote: "",
				Remotes:       map[string]*endpoint.Config{},
			},
			id:    "cloud",
			newID: "newCloud",
		},

		{
			name: "rename existing remote to existing remote",
			old: Config{
				DefaultRemote: "",
				Remotes:       map[string]*endpoint.Config{},
			},
			id:    "cloud",
			newID: "newCloud",
		},

		{
			name: "default does not exist",
			old: Config{
				DefaultRemote: "",
				Remotes: map[string]*endpoint.Config{
					"random": {
						URI:   "cloud.random.io",
						Token: testToken,
					},
					"cloud": {
						URI:   "cloud.sylabs.io",
						Token: testToken,
					},
				},
			},
			id:    "cloud",
			newID: "random",
		},
	}
	for _, test := range testsFail {
		t.Run(test.name, func(t *testing.T) {
			if err := test.old.Rename(test.id, test.newID); err == nil {
				t.Error("unexpected success renaming remote")
			}
		})
	}
}

func TestGetRemote(t *testing.T) {
	testsPass := []remoteTest{
		{
			name: "get existing remote",
			old: Config{
				DefaultRemote: "cloud",
				Remotes: map[string]*endpoint.Config{
					"random": {
						URI:   "cloud.random.io",
						Token: testToken,
					},
					"cloud": {
						URI:   "cloud.sylabs.io",
						Token: testToken,
					},
				},
			},
			id: "cloud",
			ep: &endpoint.Config{
				URI:   "cloud.sylabs.io",
				Token: testToken,
			},
		},
	}

	for _, test := range testsPass {
		t.Run(test.name, func(t *testing.T) {
			var ep *endpoint.Config
			ep, err := test.old.GetRemote(test.id)
			if err != nil {
				t.Error("failed to get endpoint from config")
			}

			if !reflect.DeepEqual(ep, test.ep) {
				t.Errorf("Add failed to get from config:\n\thave: %v\n\twant: %v", ep, test.ep)
			}
		})
	}

	testsFail := []remoteTest{
		{
			name: "remote does not exist",
			old: Config{
				DefaultRemote: "cloud",
				Remotes: map[string]*endpoint.Config{
					"cloud": {
						URI:   "cloud.sylabs.io",
						Token: testToken,
					},
				},
			},
			id: "notaremote",
		},
	}
	for _, test := range testsFail {
		t.Run(test.name, func(t *testing.T) {
			if _, err := test.old.GetRemote(test.id); err == nil {
				t.Error("unexpected success getting remote")
			}
		})
	}
}

func TestGetDefaultRemote(t *testing.T) {
	testsPass := []remoteTest{
		{
			name: "get existing default remote",
			old: Config{
				DefaultRemote: "cloud",
				Remotes: map[string]*endpoint.Config{
					"random": {
						URI:   "cloud.random.io",
						Token: testToken,
					},
					"cloud": {
						URI:   "cloud.sylabs.io",
						Token: testToken,
					},
				},
			},
			ep: &endpoint.Config{
				URI:   "cloud.sylabs.io",
				Token: testToken,
			},
		},
	}

	for _, test := range testsPass {
		t.Run(test.name, func(t *testing.T) {
			var ep *endpoint.Config
			ep, err := test.old.GetDefault()
			if err != nil {
				t.Error("failed to get endpoint from config")
			}

			if !reflect.DeepEqual(ep, test.ep) {
				t.Errorf("Add failed to get from config:\n\thave: %v\n\twant: %v", ep, test.ep)
			}
		})
	}

	testsFail := []remoteTest{
		{
			name: "no default set",
			old: Config{
				DefaultRemote: "",
				Remotes: map[string]*endpoint.Config{
					"random": {
						URI:   "cloud.random.io",
						Token: testToken,
					},
					"cloud": {
						URI:   "cloud.sylabs.io",
						Token: testToken,
					},
				},
			},
		},
		{
			name: "default does not exist",
			old: Config{
				DefaultRemote: "notaremote",
				Remotes: map[string]*endpoint.Config{
					"cloud": {
						URI:   "cloud.sylabs.io",
						Token: testToken,
					},
				},
			},
		},
	}
	for _, test := range testsFail {
		t.Run(test.name, func(t *testing.T) {
			if _, err := test.old.GetDefault(); err == nil {
				t.Error("unexpected success getting default remote")
			}
		})
	}
}

func TestSetDefaultRemote(t *testing.T) {
	testsPass := []remoteTest{
		{
			name: "set existing remote to default",
			old: Config{
				DefaultRemote: "cloud",
				Remotes: map[string]*endpoint.Config{
					"random": {
						URI:   "cloud.random.io",
						Token: testToken,
					},
					"cloud": {
						URI:   "cloud.sylabs.io",
						Token: testToken,
					},
				},
			},
			new: Config{
				DefaultRemote: "random",
				Remotes: map[string]*endpoint.Config{
					"random": {
						URI:   "cloud.random.io",
						Token: testToken,
					},
					"cloud": {
						URI:   "cloud.sylabs.io",
						Token: testToken,
					},
				},
			},
			id: "random",
		},
	}

	for _, test := range testsPass {
		t.Run(test.name, func(t *testing.T) {
			if err := test.old.SetDefault(test.id, false); err != nil {
				t.Error("failed to set default endpoint in config")
			}

			if !reflect.DeepEqual(test.old, test.new) {
				t.Errorf("Remove failed to set config:\n\thave: %v\n\twant: %v", test.old, test.new)
			}
		})
	}

	testsFail := []remoteTest{
		{
			name: "default does not exist",
			old: Config{
				DefaultRemote: "cloud",
				Remotes: map[string]*endpoint.Config{
					"cloud": {
						URI:   "cloud.sylabs.io",
						Token: testToken,
					},
				},
			},
			id: "notaremote",
		},
	}
	for _, test := range testsFail {
		t.Run(test.name, func(t *testing.T) {
			if err := test.old.SetDefault(test.id, false); err == nil {
				t.Error("unexpected success setting default remote")
			}
		})
	}

	testsExclusivePass := []remoteTest{
		{
			name: "set existing remote to default and exclusive",
			old: Config{
				DefaultRemote: "cloud",
				system:        true,
				Remotes: map[string]*endpoint.Config{
					"random": {
						URI:   "cloud.random.io",
						Token: testToken,
					},
					"cloud": {
						URI:   "cloud.sylabs.io",
						Token: testToken,
					},
				},
			},
			new: Config{
				DefaultRemote: "random",
				system:        true,
				Remotes: map[string]*endpoint.Config{
					"random": {
						URI:       "cloud.random.io",
						Exclusive: true,
						Token:     testToken,
					},
					"cloud": {
						URI:   "cloud.sylabs.io",
						Token: testToken,
					},
				},
			},
			id: "random",
		},
	}
	for _, test := range testsExclusivePass {
		t.Run(test.name, func(t *testing.T) {
			if err := test.old.SetDefault(test.id, true); err != nil {
				t.Errorf("unexpected error setting default remote as exclusive: %s", err)
			}

			if !reflect.DeepEqual(test.old, test.new) {
				t.Errorf("Remove failed to set config:\n\thave: %v\n\twant: %v", test.old, test.new)
			}
		})
	}

	testsUserExclusiveFail := []remoteTest{
		{
			name: "set existing remote to default and exclusive as user",
			old: Config{
				DefaultRemote: "cloud",
				Remotes: map[string]*endpoint.Config{
					"random": {
						URI:   "cloud.random.io",
						Token: testToken,
					},
					"cloud": {
						URI:   "cloud.sylabs.io",
						Token: testToken,
					},
				},
			},
			id: "random",
		},
	}
	for _, test := range testsUserExclusiveFail {
		t.Run(test.name, func(t *testing.T) {
			if err := test.old.SetDefault(test.id, true); err == nil {
				t.Error("unexpected success setting default remote as exclusive as user")
			}
		})
	}

	testsUserDefaultExclusiveFail := []remoteTest{
		{
			name: "set existing remote to default with different exclusive endpoint",
			old: Config{
				DefaultRemote: "cloud",
				Remotes: map[string]*endpoint.Config{
					"random": {
						URI:   "cloud.random.io",
						Token: testToken,
					},
					"cloud": {
						URI:       "cloud.sylabs.io",
						Exclusive: true,
						Token:     testToken,
					},
				},
			},
			id: "random",
		},
	}
	for _, test := range testsUserDefaultExclusiveFail {
		t.Run(test.name, func(t *testing.T) {
			if err := test.old.SetDefault(test.id, false); err == nil {
				t.Error("unexpected success setting default remote")
			}
		})
	}
}

func TestGetServiceURI(t *testing.T) {
	testsPass := []remoteTest{
		{
			name: "get uri from real cloud remote",
			old: Config{
				DefaultRemote: "cloud",
				Remotes: map[string]*endpoint.Config{
					"random": {
						URI:   "cloud.random.io",
						Token: testToken,
					},
					"cloud": {
						URI:   "cloud.sylabs.io",
						Token: testToken,
					},
				},
			},
			id: "cloud",
		},
	}

	for _, test := range testsPass {
		t.Run(test.name, func(t *testing.T) {
			var ep *endpoint.Config
			ep, err := test.old.GetRemote(test.id)
			if err != nil {
				t.Fatal("failed to get endpoint from config")
			}

			if s, err := ep.GetServiceURI("token"); s == "" || err != nil {
				t.Errorf("failed to get service uri:\n\tservice uri: %v\n\t err: %v", s, err)
			}
		})
	}

	testsFail := []remoteTest{
		{
			name: "get uri from non-existent remote",
			old: Config{
				DefaultRemote: "cloud",
				Remotes: map[string]*endpoint.Config{
					"notaremote": {
						URI:   "not.a.remote",
						Token: testToken,
					},
					"cloud": {
						URI:   "cloud.sylabs.io",
						Token: testToken,
					},
				},
			},
			id: "notaremote",
		},
	}
	for _, test := range testsFail {

		t.Run(test.name, func(t *testing.T) {
			var ep *endpoint.Config
			ep, err := test.old.GetRemote(test.id)
			if err != nil {
				t.Fatal("failed to get endpoint from config")
			}

			if _, err := ep.GetServiceURI("token"); err == nil {
				t.Error("unexpected success getting uri for non-existent remote")
			}
		})
	}
}
