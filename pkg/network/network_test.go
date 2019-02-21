// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package network

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/containernetworking/cni/libcni"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
)

var confFiles = []struct {
	name    string
	file    string
	content string
}{
	{
		name: "test-bridge",
		file: "00_test-bridge.conflist",
		content: `{
			"cniVersion": "0.3.1",
			"name": "test-bridge",
			"plugins": [
				{
					"type": "bridge",
					"bridge": "tbr0",
					"isGateway": true,
					"ipMasq": true,
					"ipam": {
						"type": "host-local",
						"subnet": "10.111.111.0/24",
						"routes": [
							{ "dst": "0.0.0.0/0" }
						]
					}
				},
				{
					"type": "portmap",
					"capabilities": {"portMappings": true},
					"snat": true
				}
			]
		}`,
	},
	{
		name: "test-badbridge",
		file: "10_badbridge.conf",
		content: `{
			"cniVersion": "0.3.1",
			"name": "test-badbridge",
			"plugins": [
				{
					"type": "badbridge",
					"bridge": "tbr0"
				}
			]
		}`,
	},
	{
		name: "test-bridge-iprange",
		file: "20_bridge_iprange.conflist",
		content: `{
			"cniVersion": "0.3.1",
			"name": "test-bridge-iprange",
			"plugins": [
				{
					"type": "bridge",
					"bridge": "tibr0",
					"isGateway": true,
					"ipMasq": true,
					"capabilities": {"ipRanges": true},
					"ipam": {
						"type": "host-local",
						"routes": [
							{ "dst": "0.0.0.0/0" }
						]
					}
				},
				{
					"type": "portmap",
					"capabilities": {"portMappings": true},
					"snat": true
				}
			]
		}`,
	},
}

// defaultCNIConfPath is the default directory to CNI network configuration files
var defaultCNIConfPath = ""

// defaultCNIPluginPath is the default directory to CNI plugins executables
var defaultCNIPluginPath = filepath.Join(buildcfg.LIBEXECDIR, "singularity", "cni")

var testNetworks []string

func TestGetAllNetworkConfigList(t *testing.T) {
	emptyDir, err := ioutil.TempDir("", "empty_conf_")
	if err != nil {
		t.Errorf("failed to creaty empty configuration directory: %s", err)
	}
	defer os.Remove(emptyDir)

	var testCNIPath = []struct {
		name           string
		cniPath        *CNIPath
		success        bool
		validationFunc func([]*libcni.NetworkConfigList) error
	}{
		{
			name:    "'nil CNIPath'",
			cniPath: nil,
			success: false,
		},
		{
			name: "'empty configuration path'",
			cniPath: &CNIPath{
				Conf:   "",
				Plugin: "",
			},
			success: false,
		},
		{
			name: "'empty configuration directory'",
			cniPath: &CNIPath{
				Conf:   emptyDir,
				Plugin: "",
			},
			success: false,
		},
		{
			name: "'default configuration/plugin path'",
			cniPath: &CNIPath{
				Conf:   defaultCNIConfPath,
				Plugin: defaultCNIPluginPath,
			},
			success: true,
			validationFunc: func(networkList []*libcni.NetworkConfigList) error {
				var networks []string
				for _, n := range networkList {
					networks = append(networks, n.Name)
				}
				if !reflect.DeepEqual(networks, testNetworks) {
					return fmt.Errorf("wrong network list returned: %v", networks)
				}
				return nil
			},
		},
	}

	for _, c := range testCNIPath {
		networkList, err := GetAllNetworkConfigList(c.cniPath)
		if err != nil && c.success {
			t.Errorf("unexpected failure for %s test: %s", c.name, err)
		} else if err == nil && !c.success {
			t.Errorf("unexpected success for %s test", c.name)
		} else if c.validationFunc != nil {
			if err := c.validationFunc(networkList); err != nil {
				t.Error(err)
			}
		}
	}
}

func TestMain(m *testing.M) {
	var err error

	defaultCNIConfPath, err = ioutil.TempDir("", "conf_test_")
	if err != nil {
		os.Exit(1)
	}

	for _, conf := range confFiles {
		testNetworks = append(testNetworks, conf.name)
		path := filepath.Join(defaultCNIConfPath, conf.file)
		if err := ioutil.WriteFile(path, []byte(conf.content), 0644); err != nil {
			os.RemoveAll(defaultCNIConfPath)
			os.Exit(1)
		}
	}

	e := m.Run()
	os.RemoveAll(defaultCNIConfPath)
	os.Exit(e)
}
