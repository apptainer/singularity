package network

import (
	"fmt"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/containernetworking/cni/libcni"
	"github.com/singularityware/singularity/src/pkg/buildcfg"
)

// CNIPath ...
type CNIPath struct {
	Conf   string
	Plugin string
}

// List ...
type List struct {
	networks      []networkList
	cniConfPath   string
	cniPluginPath string
	containerID   string
	netNS         string
	ifIndex       uint16
}

// networkList ...
type networkList struct {
	name    string
	portMap []portMap
	args    [][2]string
}

// portMap ...
type portMap struct {
	hostPort      uint16
	containerPort uint16
	protocol      string
}

// DefaultCNIConfPath is the default path to CNI network configuration files
var DefaultCNIConfPath = path.Join(buildcfg.SYSCONFDIR, "singularity/network")

// DefaultCNIPluginPath is the default path to CNI plugins executables
var DefaultCNIPluginPath = path.Join(buildcfg.LIBEXECDIR, "singularity/cni")

// AvailableNetworks ...
func AvailableNetworks(cniPath *CNIPath) ([]string, error) {
	networks := make([]string, 0)
	cniConfPath := DefaultCNIConfPath

	if cniPath != nil {
		cniConfPath = cniPath.Conf
	}

	files, err := libcni.ConfFiles(cniConfPath, []string{".conf", ".json", ".conflist"})
	if err != nil {
		return nil, err
	} else if len(files) == 0 {
		return nil, libcni.NoConfigsFoundError{Dir: cniConfPath}
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

// NewNetworkList ...
func NewNetworkList(networks []string, containerID string, netNS string, cniPath *CNIPath) (*List, error) {
	nlist := make([]networkList, 0)
	cniConfPath := DefaultCNIConfPath
	cniPluginPath := DefaultCNIPluginPath
	id := containerID

	if id == "" {
		id = strconv.Itoa(os.Getpid())
	}

	if cniPath != nil {
		cniConfPath = cniPath.Conf
		cniPluginPath = cniPath.Plugin
	}
	hasNone := false
	for _, network := range networks {
		if network == "none" {
			hasNone = true
			break
		}
		nlist = append(nlist, networkList{
			name:    network,
			portMap: make([]portMap, 0),
			args:    make([][2]string, 0),
		})
	}
	if hasNone {
		if len(networks) > 1 {
			return nil, fmt.Errorf("none network can't be specified with another network")
		}
		return &List{networks: []networkList{}}, nil
	}
	return &List{
			networks:      nlist,
			cniConfPath:   cniConfPath,
			cniPluginPath: cniPluginPath,
			netNS:         netNS,
			containerID:   id,
			ifIndex:       0,
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

// AddNetworkArgs ...
func (l *List) AddNetworkArgs(args []string) error {
	var hostPort uint16
	var containerPort uint16

	if len(l.networks) < 1 {
		return fmt.Errorf("there is no configured network in list")
	}

	for _, arg := range args {
		networkName := ""

		splitted := strings.SplitN(arg, ":", 2)
		if len(splitted) < 1 && len(splitted) > 2 {
			return fmt.Errorf("argument must be of form '<network>:KEY1=value1;KEY2=value1' or 'KEY1=value1;KEY2=value1'")
		}
		if len(splitted) == 1 {
			networkName = l.networks[0].name
		} else {
			networkName = splitted[0]
		}
		hasNetwork := false
		for _, network := range l.networks {
			if network.name == networkName {
				hasNetwork = true
				break
			}
		}
		if !hasNetwork {
			return fmt.Errorf("network %s not found", networkName)
		}
		argList, err := parseArg(splitted[1])
		if err != nil {
			return err
		}
		for _, kv := range argList {
			key := kv[0]
			value := kv[1]
			if key == "portmap" {
				splittedPort := strings.SplitN(value, "/", 2)
				if len(splittedPort) != 2 {
					return fmt.Errorf("badly formatted portmap argument '%s', must be of form portmap:hostPort:containerPort/protocol", splitted[1])
				}
				protocol := splittedPort[1]
				if protocol != "tcp" && protocol != "udp" {
					return fmt.Errorf("only tcp and udp protocol can be specified")
				}
				ports := strings.Split(splittedPort[0], ":")
				if len(ports) != 1 && len(ports) != 2 {
					return fmt.Errorf("portmap port argument is badly formatted")
				}
				if n, err := strconv.ParseUint(ports[0], 0, 16); err == nil {
					hostPort = uint16(n)
					if hostPort == 0 {
						return fmt.Errorf("host port can't be zero")
					}
				} else {
					return fmt.Errorf("can't convert host port '%s': %s", ports[0], err)
				}
				if len(ports) == 2 {
					if n, err := strconv.ParseUint(ports[1], 0, 16); err == nil {
						containerPort = uint16(n)
						if containerPort == 0 {
							return fmt.Errorf("container port can't be zero")
						}
					} else {
						return fmt.Errorf("can't convert container port '%s': %s", ports[1], err)
					}
				} else {
					containerPort = hostPort
				}
				for i := range l.networks {
					if l.networks[i].name == networkName {
						l.networks[i].portMap = append(l.networks[i].portMap, portMap{
							containerPort: containerPort,
							hostPort:      hostPort,
							protocol:      protocol,
						})
					}
				}
			} else {
				for i := range l.networks {
					if l.networks[i].name == networkName {
						l.networks[i].args = append(l.networks[i].args, kv)
					}
				}
			}
		}
	}
	return nil
}

// SetupNetworks ...
func (l *List) SetupNetworks() error {
	return l.command("ADD")
}

// CleanupNetworks ...
func (l *List) CleanupNetworks() error {
	return l.command("DEL")
}

// NetworkList create network interface in container
func (l *List) command(command string) error {
	networkConfList := make([]*libcni.NetworkConfigList, 0)
	runtimeConf := make([]*libcni.RuntimeConf, 0)
	config := &libcni.CNIConfig{Path: []string{l.cniPluginPath}}

	for _, network := range l.networks {
		capabilityArgs := make(map[string]interface{}, 0)
		portMappings := make([]map[string]interface{}, 0)

		nlist, err := libcni.LoadConfList(l.cniConfPath, network.name)
		if err != nil {
			return err
		}
		networkConfList = append(networkConfList, nlist)
		ifName := fmt.Sprintf("eth%d", l.ifIndex)

		for _, portMap := range network.portMap {
			portMappings = append(portMappings, map[string]interface{}{
				"hostPort":      portMap.hostPort,
				"containerPort": portMap.containerPort,
				"protocol":      portMap.protocol,
			})
		}

		if len(portMappings) > 0 {
			capabilityArgs["portMappings"] = portMappings
		}
		rt := &libcni.RuntimeConf{
			ContainerID:    l.containerID,
			NetNS:          l.netNS,
			IfName:         ifName,
			CapabilityArgs: capabilityArgs,
			Args:           network.args,
		}
		runtimeConf = append(runtimeConf, rt)
		l.ifIndex++
	}
	l.ifIndex = 0
	for i := 0; i < len(networkConfList); i++ {
		switch command {
		case "ADD":
			if _, err := config.AddNetworkList(networkConfList[i], runtimeConf[i]); err != nil {
				for j := i - 1; j >= 0; j-- {
					if err := config.DelNetworkList(networkConfList[j], runtimeConf[j]); err != nil {
						return err
					}
				}
				return err
			}
		case "DEL":
			if err := config.DelNetworkList(networkConfList[i], runtimeConf[i]); err != nil {
				return err
			}
		}
	}
	return nil
}
