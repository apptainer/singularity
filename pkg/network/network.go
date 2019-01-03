package network

import (
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/containernetworking/cni/libcni"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/plugins/plugins/ipam/host-local/backend/allocator"
	"github.com/sylabs/singularity/internal/pkg/util/env"
)

type netError string

func (e netError) Error() string { return string(e) }

const (
	// ErrNoCNIConfig corresponds to a missing CNI configuration path
	ErrNoCNIConfig = netError("no CNI configuration path provided")
	// ErrNoCNIPlugin corresponds to a missing CNI plugin path
	ErrNoCNIPlugin = netError("no CNI plugin path provided")
)

// CNIPath contains path to CNI configuration directory and path to executable
// CNI plugins directory
type CNIPath struct {
	Conf   string
	Plugin string
}

// Setup contains network installation setup
type Setup struct {
	networks        []string
	networkConfList []*libcni.NetworkConfigList
	runtimeConf     []*libcni.RuntimeConf
	result          []types.Result
	cniPath         *CNIPath
	containerID     string
	netNS           string
}

// PortMapEntry describes a port mapping between host and container
type PortMapEntry struct {
	HostPort      int    `json:"hostPort"`
	ContainerPort int    `json:"containerPort"`
	Protocol      string `json:"protocol"`
	HostIP        string `json:"hostIP,omitempty"`
}

// AvailableNetworks lists configured networks in configuration path directory
// provided by cniPath
func AvailableNetworks(cniPath *CNIPath) ([]string, error) {
	networks := make([]string, 0)

	if cniPath == nil {
		return networks, ErrNoCNIConfig
	}
	if cniPath.Conf == "" {
		return networks, ErrNoCNIConfig
	}

	files, err := libcni.ConfFiles(cniPath.Conf, []string{".conf", ".json", ".conflist"})
	if err != nil {
		return nil, err
	} else if len(files) == 0 {
		return nil, libcni.NoConfigsFoundError{Dir: cniPath.Conf}
	}
	sort.Strings(files)

	for _, file := range files {
		if strings.HasSuffix(file, ".conflist") {
			conf, err := libcni.ConfListFromFile(file)
			if err != nil {
				return nil, err
			}
			networks = append(networks, conf.Name)
		} else {
			conf, err := libcni.ConfFromFile(file)
			if err != nil {
				return nil, err
			}
			networks = append(networks, conf.Network.Name)
		}
	}
	return networks, nil
}

// NewSetup creates and returns a network setup to configure, add and remove
// network interfaces in container
func NewSetup(networks []string, containerID string, netNS string, cniPath *CNIPath) (*Setup, error) {
	id := containerID

	if id == "" {
		id = strconv.Itoa(os.Getpid())
	}

	if cniPath == nil {
		return nil, ErrNoCNIConfig
	}
	if cniPath.Conf == "" {
		return nil, ErrNoCNIConfig
	}
	if cniPath.Plugin == "" {
		return nil, ErrNoCNIPlugin
	}

	networkConfList := make([]*libcni.NetworkConfigList, 0)
	runtimeConf := make([]*libcni.RuntimeConf, 0)

	ifIndex := 0
	for _, network := range networks {
		nlist, err := libcni.LoadConfList(cniPath.Conf, network)
		if err != nil {
			return nil, err
		}

		rt := &libcni.RuntimeConf{
			ContainerID:    containerID,
			NetNS:          netNS,
			IfName:         fmt.Sprintf("eth%d", ifIndex),
			CapabilityArgs: make(map[string]interface{}, 0),
			Args:           make([][2]string, 0),
		}

		runtimeConf = append(runtimeConf, rt)
		networkConfList = append(networkConfList, nlist)

		ifIndex++
	}

	return &Setup{
			networks:        networks,
			networkConfList: networkConfList,
			runtimeConf:     runtimeConf,
			cniPath:         cniPath,
			netNS:           netNS,
			containerID:     id,
		},
		nil
}

func parseArg(arg string) ([][2]string, error) {
	argList := make([][2]string, 0)

	pairs := strings.Split(arg, ";")
	for _, pair := range pairs {
		keyVal := strings.Split(pair, "=")
		if len(keyVal) != 2 {
			return nil, fmt.Errorf("invalid argument: %s", pair)
		}
		argList = append(argList, [2]string{keyVal[0], keyVal[1]})
	}
	return argList, nil
}

// SetCapability sets capability arguments for the corresponding network plugin
// uses by a configured network
func (m *Setup) SetCapability(network string, capName string, args interface{}) error {
	for i := range m.networks {
		if m.networks[i] == network {
			hasCap := false
			for _, plugin := range m.networkConfList[i].Plugins {
				if plugin.Network.Capabilities[capName] {
					hasCap = true
					break
				}
			}

			if !hasCap {
				return fmt.Errorf("%s network doesn't have %s capability", network, capName)
			}

			switch args.(type) {
			case PortMapEntry:
				if m.runtimeConf[i].CapabilityArgs[capName] == nil {
					m.runtimeConf[i].CapabilityArgs[capName] = make([]PortMapEntry, 0)
				}
				m.runtimeConf[i].CapabilityArgs[capName] = append(
					m.runtimeConf[i].CapabilityArgs[capName].([]PortMapEntry),
					args.(PortMapEntry),
				)
			case []allocator.Range:
				if m.runtimeConf[i].CapabilityArgs[capName] == nil {
					m.runtimeConf[i].CapabilityArgs[capName] = []allocator.RangeSet{args.([]allocator.Range)}
				}
			}
		}
	}
	return nil
}

// SetArgs affects arguments to corresponding network plugins
func (m *Setup) SetArgs(args []string) error {
	if len(m.networks) < 1 {
		return fmt.Errorf("there is no configured network in list")
	}

	for _, arg := range args {
		var splitted []string
		networkName := ""

		if strings.IndexByte(arg, ':') > strings.IndexByte(arg, '=') {
			splitted = []string{m.networks[0], arg}
		} else {
			splitted = strings.SplitN(arg, ":", 2)
		}
		if len(splitted) < 1 && len(splitted) > 2 {
			return fmt.Errorf("argument must be of form '<network>:KEY1=value1;KEY2=value1' or 'KEY1=value1;KEY2=value1'")
		}
		n := len(splitted) - 1
		if n == 0 {
			networkName = m.networks[0]
		} else {
			networkName = splitted[0]
		}
		hasNetwork := false
		for _, network := range m.networks {
			if network == networkName {
				hasNetwork = true
				break
			}
		}
		if !hasNetwork {
			return fmt.Errorf("network %s wasn't specified in --network option", networkName)
		}
		argList, err := parseArg(splitted[n])
		if err != nil {
			return err
		}
		for _, kv := range argList {
			key := kv[0]
			value := kv[1]
			if key == "portmap" {
				pm := &PortMapEntry{}

				splittedPort := strings.SplitN(value, "/", 2)
				if len(splittedPort) != 2 {
					return fmt.Errorf("badly formatted portmap argument '%s', must be of form portmap=hostPort:containerPort/protocol", splitted[1])
				}
				pm.Protocol = splittedPort[1]
				if pm.Protocol != "tcp" && pm.Protocol != "udp" {
					return fmt.Errorf("only tcp and udp protocol can be specified")
				}
				ports := strings.Split(splittedPort[0], ":")
				if len(ports) != 1 && len(ports) != 2 {
					return fmt.Errorf("portmap port argument is badly formatted")
				}
				if n, err := strconv.ParseInt(ports[0], 0, 16); err == nil {
					pm.HostPort = int(n)
					if pm.HostPort <= 0 {
						return fmt.Errorf("host port must be greater than zero")
					}
				} else {
					return fmt.Errorf("can't convert host port '%s': %s", ports[0], err)
				}
				if len(ports) == 2 {
					if n, err := strconv.ParseInt(ports[1], 0, 16); err == nil {
						pm.ContainerPort = int(n)
						if pm.ContainerPort <= 0 {
							return fmt.Errorf("container port must be greater than zero")
						}
					} else {
						return fmt.Errorf("can't convert container port '%s': %s", ports[1], err)
					}
				} else {
					pm.ContainerPort = pm.HostPort
				}
				if err := m.SetCapability(networkName, "portMappings", *pm); err != nil {
					return err
				}
			} else if key == "ipRange" {
				ipRange := make([]allocator.Range, 1)
				_, subnet, err := net.ParseCIDR(value)
				if err != nil {
					return err
				}
				ipRange[0].Subnet = types.IPNet(*subnet)
				if err := m.SetCapability(networkName, "ipRanges", ipRange); err != nil {
					return err
				}
			} else {
				for i := range m.networks {
					if m.networks[i] == networkName {
						m.runtimeConf[i].Args = append(m.runtimeConf[i].Args, kv)
					}
				}
			}
		}
	}
	return nil
}

// AddNetworks brings up networks interface in container
func (m *Setup) AddNetworks() error {
	return m.command("ADD")
}

// DelNetworks tears down networks interface in container
func (m *Setup) DelNetworks() error {
	return m.command("DEL")
}

func (m *Setup) command(command string) error {
	backupEnv := os.Environ()
	os.Clearenv()
	os.Setenv("PATH", "/bin:/sbin:/usr/bin:/usr/sbin")
	defer env.SetFromList(backupEnv)

	config := &libcni.CNIConfig{Path: []string{m.cniPath.Plugin}}

	if command == "ADD" {
		m.result = make([]types.Result, len(m.networkConfList))
		for i := 0; i < len(m.networkConfList); i++ {
			var err error
			if m.result[i], err = config.AddNetworkList(m.networkConfList[i], m.runtimeConf[i]); err != nil {
				for j := i - 1; j >= 0; j-- {
					if err := config.DelNetworkList(m.networkConfList[j], m.runtimeConf[j]); err != nil {
						return err
					}
				}
				return err
			}
		}
	} else if command == "DEL" {
		for i := 0; i < len(m.networkConfList); i++ {
			if err := config.DelNetworkList(m.networkConfList[i], m.runtimeConf[i]); err != nil {
				return err
			}
		}
	}
	return nil
}
