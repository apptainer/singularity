// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package network

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"

	"github.com/containernetworking/cni/libcni"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/test"
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
			"cniVersion": "0.4.0",
			"name": "test-bridge",
			"plugins": [
				{
					"type": "loopback"
				},
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
		file: "10_badbridge.conflist",
		content: `{
			"cniVersion": "0.4.0",
			"name": "test-badbridge",
			"plugins": [
				{
					"type": "badbridge",
					"bridge": "bbr0"
				}
			]
		}`,
	},
	{
		name: "test-bridge-iprange",
		file: "20_bridge_iprange.conflist",
		content: `{
			"cniVersion": "0.4.0",
			"name": "test-bridge-iprange",
			"plugins": [
				{
					"type": "loopback"
				},
				{
					"type": "bridge",
					"bridge": "tipbr0",
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

// testNetworks will contains configured network
var testNetworks []string

func TestGetAllNetworkConfigList(t *testing.T) {
	test.EnsurePrivilege(t)

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
			name:    "nil CNIPath",
			cniPath: nil,
			success: false,
		},
		{
			name: "empty configuration path",
			cniPath: &CNIPath{
				Conf:   "",
				Plugin: "",
			},
			success: false,
		},
		{
			name: "empty configuration directory",
			cniPath: &CNIPath{
				Conf:   emptyDir,
				Plugin: "",
			},
			success: false,
		},
		{
			name: "default configuration/plugin path",
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
			t.Errorf("unexpected failure for %q test: %s", c.name, err)
		} else if err == nil && !c.success {
			t.Errorf("unexpected success for %q test", c.name)
		} else if c.validationFunc != nil {
			if err := c.validationFunc(networkList); err != nil {
				t.Error(err)
			}
		}
	}
}

func testSetArgs(setup *Setup, t *testing.T) {
	var testArgs = []struct {
		desc    string
		args    []string
		success bool
	}{
		{
			desc:    "empty arg",
			args:    []string{""},
			success: false,
		},
		{
			desc:    "badly formatted arg #1",
			args:    []string{"test-bridge:"},
			success: false,
		},
		{
			desc:    "badly formatted arg #2",
			args:    []string{":portmap=80/tcp"},
			success: false,
		},
		{
			desc:    "badly formatted arg #3",
			args:    []string{"portmap=80/tcp;portmap="},
			success: false,
		},
		{
			desc:    "empty portmap",
			args:    []string{"test-bridge:portmap="},
			success: false,
		},
		{
			desc:    "unknown portmap protocol",
			args:    []string{"test-bridge:portmap=80/icmp"},
			success: false,
		},
		{
			desc:    "portmap 0",
			args:    []string{"test-bridge:portmap=0/tcp"},
			success: false,
		},
		{
			desc:    "good portmap arg #1",
			args:    []string{"test-bridge:portmap=80:80/tcp", "portmap=80:80/tcp"},
			success: true,
		},
		{
			desc:    "good portmap arg #2",
			args:    []string{"portmap=80:80/tcp;portmap=8080/udp"},
			success: true,
		},
		{
			desc:    "good 1-1 portmap arg",
			args:    []string{"test-bridge:portmap=80/udp", "test-bridge-iprange:portmap=8080/tcp"},
			success: true,
		},
		{
			desc:    "ipRange not supported arg",
			args:    []string{"test-bridge:ipRange=10.1.1.0/16"},
			success: false,
		},
		{
			desc:    "good ipRange arg",
			args:    []string{"test-bridge-iprange:ipRange=10.1.1.0/16"},
			success: true,
		},
		{
			desc:    "bad ipRange arg",
			args:    []string{"test-bridge-iprange:ipRange=1024.1.1.0/16"},
			success: false,
		},
		{
			desc:    "IP arg",
			args:    []string{"test-bridge:IP=10.1.1.1"},
			success: true,
		},
		{
			desc:    "Any arg",
			args:    []string{"test-bridge:any=test"},
			success: true,
		},
	}
	for _, a := range testArgs {
		err := setup.SetArgs(a.args)
		if err != nil && a.success {
			t.Errorf("unexpected failure for %q test: %s", a.desc, err)
		} else if err == nil && !a.success {
			t.Errorf("unexpected success for %q test", a.desc)
		}
	}
}

func TestNewSetup(t *testing.T) {
	test.EnsurePrivilege(t)

	var cniPath = &CNIPath{
		Conf:   defaultCNIConfPath,
		Plugin: defaultCNIPluginPath,
	}
	var testSetup = []struct {
		desc     string
		networks []string
		id       string
		nspath   string
		cniPath  *CNIPath
		success  bool
		subTest  func(*Setup, *testing.T)
	}{
		{
			desc:     "no name network",
			networks: []string{""},
			id:       "testing",
			nspath:   "/proc/self/net/ns",
			cniPath:  cniPath,
			success:  false,
		},
		{
			desc:     "bad network",
			networks: []string{"fake-network"},
			id:       "testing",
			nspath:   "/proc/self/net/ns",
			cniPath:  cniPath,
			success:  false,
		},
		{
			desc:     "bad networks",
			networks: []string{"test-bridge", "fake-network"},
			id:       "testing",
			nspath:   "/proc/self/net/ns",
			cniPath:  cniPath,
			success:  false,
		},
		{
			desc:     "good network",
			networks: []string{"test-bridge"},
			nspath:   "/proc/self/net/ns",
			cniPath:  cniPath,
			success:  true,
		},
		{
			desc:     "good networks",
			networks: []string{"test-bridge", "test-bridge-iprange"},
			nspath:   "/proc/self/net/ns",
			cniPath:  cniPath,
			success:  true,
			subTest:  testSetArgs,
		},
		{
			desc:     "nil cni path",
			networks: []string{""},
			id:       "testing",
			success:  false,
		},
	}

	for _, s := range testSetup {
		setup, err := NewSetup(s.networks, s.id, s.nspath, s.cniPath)
		if err != nil && s.success {
			t.Errorf("unexpected failure for %q test: %s", s.desc, err)
		} else if err == nil && !s.success {
			t.Errorf("unexpected success for %q test", s.desc)
		} else if s.subTest != nil {
			s.subTest(setup, t)
		}
	}
}

// ping requested IP from host
func testPingIP(nsPath string, cniPath *CNIPath, stdin io.WriteCloser, stdout io.ReadCloser) error {
	testIP := "10.111.111.10"

	setup, err := NewSetup([]string{"test-bridge"}, "_test_", nsPath, cniPath)
	if err != nil {
		return err
	}
	setup.SetArgs([]string{"IP=" + testIP})
	if err := setup.AddNetworks(); err != nil {
		return err
	}
	defer setup.DelNetworks()

	ip, err := setup.GetNetworkIP("test-bridge", "4")
	if err != nil {
		return err
	}
	cmdPath, err := exec.LookPath("ping")
	if err != nil {
		return err
	}
	if ip.String() != testIP {
		return fmt.Errorf("%s doesn't match with requested ip %s", ip.String(), testIP)
	}
	cmd := exec.Command(cmdPath, "-c", "1", testIP)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

// ping random acquired IP from host
func testPingRandomIP(nsPath string, cniPath *CNIPath, stdin io.WriteCloser, stdout io.ReadCloser) error {
	setup, err := NewSetup([]string{"test-bridge"}, "_test_", nsPath, cniPath)
	if err != nil {
		return err
	}
	if err := setup.AddNetworks(); err != nil {
		return err
	}
	defer setup.DelNetworks()

	ip, err := setup.GetNetworkIP("test-bridge", "4")
	if err != nil {
		return err
	}
	cmdPath, err := exec.LookPath("ping")
	if err != nil {
		return err
	}
	cmd := exec.Command(cmdPath, "-c", "1", ip.String())
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

// ping IP from host within requested IP range
func testPingIPRange(nsPath string, cniPath *CNIPath, stdin io.WriteCloser, stdout io.ReadCloser) error {
	setup, err := NewSetup([]string{"test-bridge-iprange"}, "_test_", nsPath, cniPath)
	if err != nil {
		return err
	}
	setup.SetArgs([]string{"ipRange=10.111.112.0/24"})
	if err := setup.AddNetworks(); err != nil {
		return err
	}
	defer setup.DelNetworks()

	ip, err := setup.GetNetworkIP("test-bridge", "4")
	if err != nil {
		ip, err = setup.GetNetworkIP("test-bridge-iprange", "4")
		if err != nil {
			return err
		}
	}
	cmdPath, err := exec.LookPath("ping")
	if err != nil {
		return err
	}
	if !strings.HasPrefix(ip.String(), "10.111.112") {
		return fmt.Errorf("ip address %s not in net range 10.111.112.0/24", ip.String())
	}
	cmd := exec.Command(cmdPath, "-c", "1", ip.String())
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

// test port mapping by connecting to port 80 mapped inside container
// to 31080 on host
func testHTTPPortmap(nsPath string, cniPath *CNIPath, stdin io.WriteCloser, stdout io.ReadCloser) error {
	setup, err := NewSetup([]string{"test-bridge"}, "_test_", nsPath, cniPath)
	if err != nil {
		return err
	}
	setup.SetArgs([]string{"portmap=31080:80/tcp"})
	if err := setup.AddNetworks(); err != nil {
		return err
	}
	defer setup.DelNetworks()

	eth, err := setup.GetNetworkInterface("test-bridge-iprange")
	if err != nil {
		eth, err = setup.GetNetworkInterface("test-bridge")
		if err != nil {
			return err
		}
	}
	if eth != "eth0" {
		return fmt.Errorf("unexpected interface %s", eth)
	}
	conn, err := net.Dial("tcp", ":31080")
	if err != nil {
		return err
	}
	message := "test\r\n"

	if _, err := conn.Write([]byte(message)); err != nil {
		return err
	}
	conn.Close()

	received, err := ioutil.ReadAll(stdout)
	if err != nil {
		return err
	}
	if string(received) != message {
		return fmt.Errorf("received data doesn't match message: %s", string(received))
	}
	return nil
}

// try with an non existent plugin
func testBadBridge(nsPath string, cniPath *CNIPath, stdin io.WriteCloser, stdout io.ReadCloser) error {
	setup, err := NewSetup([]string{"test-badbridge"}, "", nsPath, cniPath)
	if err != nil {
		return err
	}
	if err := setup.AddNetworks(); err == nil {
		return fmt.Errorf("unexpected success while calling non existent plugin")
	}
	defer setup.DelNetworks()

	return nil
}

func TestAddDelNetworks(t *testing.T) {
	test.EnsurePrivilege(t)

	// centos 6 doesn't support brigde/veth, only macvlan
	// just skip tests on centos 6
	b, err := ioutil.ReadFile("/etc/system-release-cpe")
	if err == nil {
		fields := strings.Split(string(b), ":")
		switch fields[2] {
		case "centos":
			if strings.HasPrefix(fields[4], "6") {
				t.SkipNow()
			}
		}
	}

	var cniPath = &CNIPath{
		Conf:   defaultCNIConfPath,
		Plugin: defaultCNIPluginPath,
	}

	for _, c := range []struct {
		name    string
		command string
		args    []string
		runFunc func(string, *CNIPath, io.WriteCloser, io.ReadCloser) error
	}{
		{
			name:    "TestPingIP",
			command: "cat",
			runFunc: testPingIP,
		},
		{
			name:    "TestPingRandomIP",
			command: "cat",
			runFunc: testPingRandomIP,
		},
		{
			name:    "TestHTTPPortmap",
			command: "nc",
			args:    []string{"-l", "-p", "80"},
			runFunc: testHTTPPortmap,
		},
		{
			name:    "TestPingIPRange",
			command: "cat",
			runFunc: testPingIPRange,
		},
		{
			name:    "TestBadBridge",
			command: "cat",
			runFunc: testBadBridge,
		},
	} {
		var err error
		var cmdPath string
		var stdinPipe io.WriteCloser
		var stdoutPipe io.ReadCloser

		cmdPath, err = exec.LookPath(c.command)
		if err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(cmdPath, c.args...)
		cmd.SysProcAttr = &syscall.SysProcAttr{}
		cmd.SysProcAttr.Cloneflags = syscall.CLONE_NEWNET

		stdinPipe, err = cmd.StdinPipe()
		if err != nil {
			t.Fatal(err)
		}
		stdoutPipe, err = cmd.StdoutPipe()
		if err != nil {
			t.Fatal(err)
		}

		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}

		nsPath := fmt.Sprintf("/proc/%d/ns/net", cmd.Process.Pid)
		if err := c.runFunc(nsPath, cniPath, stdinPipe, stdoutPipe); err != nil {
			t.Errorf("unexpected failure for %q: %s", c.name, err)
		}

		stdoutPipe.Close()
		stdinPipe.Close()

		if err := cmd.Wait(); err != nil {
			t.Error(err)
		}
	}
}

func TestMain(m *testing.M) {
	var err error

	test.EnsurePrivilege(nil)

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
