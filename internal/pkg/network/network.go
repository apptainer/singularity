package network

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/containernetworking/cni/libcni"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/util/env"
)

// CNIPath contains path to CNI configuration directory and path to executable
// CNI plugins directory
type CNIPath struct {
	Conf   string
	Plugin string
}

// Setup contains network installation setup
type Setup struct {
	configs         []config
	networkConfList []*libcni.NetworkConfigList
	runtimeConf     []*libcni.RuntimeConf
	result          []types.Result
	cniPath         *CNIPath
	containerID     string
	netNS           string
}

// config describes a runtime network configuration
type config struct {
	name    string
	portMap []portMap
	args    [][2]string
}

// portMap describes a port mapping between host and container
type portMap struct {
	hostPort      uint16
	containerPort uint16
	protocol      string
}

// DefaultCNIConfPath is the default directory to CNI network configuration files
var DefaultCNIConfPath = filepath.Join(buildcfg.SYSCONFDIR, "singularity", "network")

// DefaultCNIPluginPath is the default directory to CNI plugins executables
var DefaultCNIPluginPath = filepath.Join(buildcfg.LIBEXECDIR, "singularity", "cni")

// AvailableNetworks lists configured networks in configuration path directory
// provided by cniPath
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

// NewSetup creates and returns a network setup to configure, add and remove
// network interfaces in container
func NewSetup(networks []string, containerID string, netNS string, cniPath *CNIPath) (*Setup, error) {
	nlist := make([]config, 0)
	finalCNIPath := &CNIPath{
		Conf:   DefaultCNIConfPath,
		Plugin: DefaultCNIPluginPath,
	}
	id := containerID

	if id == "" {
		id = strconv.Itoa(os.Getpid())
	}

	if cniPath != nil {
		if cniPath.Conf != "" {
			finalCNIPath.Conf = cniPath.Conf
		}
		if cniPath.Plugin != "" {
			finalCNIPath.Plugin = cniPath.Plugin
		}
	}
	hasNone := false
	for _, network := range networks {
		if network == "none" {
			hasNone = true
			break
		}
		nlist = append(nlist, config{
			name:    network,
			portMap: make([]portMap, 0),
			args:    make([][2]string, 0),
		})
	}
	if hasNone {
		if len(networks) > 1 {
			return nil, fmt.Errorf("none network can't be specified with another network")
		}
		return &Setup{configs: []config{}}, nil
	}
	return &Setup{
			configs:     nlist,
			cniPath:     finalCNIPath,
			netNS:       netNS,
			containerID: id,
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

// SetArgs affects arguments to corresponding network plugins
func (m *Setup) SetArgs(args []string) error {
	var hostPort uint16
	var containerPort uint16

	if len(m.configs) < 1 {
		return fmt.Errorf("there is no configured network in list")
	}

	for _, arg := range args {
		var splitted []string
		networkName := ""

		if strings.IndexByte(arg, ':') > strings.IndexByte(arg, '=') {
			splitted = []string{m.configs[0].name, arg}
		} else {
			splitted = strings.SplitN(arg, ":", 2)
		}
		if len(splitted) < 1 && len(splitted) > 2 {
			return fmt.Errorf("argument must be of form '<network>:KEY1=value1;KEY2=value1' or 'KEY1=value1;KEY2=value1'")
		}
		n := len(splitted) - 1
		if n == 0 {
			networkName = m.configs[0].name
		} else {
			networkName = splitted[0]
		}
		hasNetwork := false
		for _, network := range m.configs {
			if network.name == networkName {
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
				for i := range m.configs {
					if m.configs[i].name == networkName {
						m.configs[i].portMap = append(m.configs[i].portMap, portMap{
							containerPort: containerPort,
							hostPort:      hostPort,
							protocol:      protocol,
						})
					}
				}
			} else {
				for i := range m.configs {
					if m.configs[i].name == networkName {
						m.configs[i].args = append(m.configs[i].args, kv)
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

	if m.networkConfList == nil {
		m.networkConfList = make([]*libcni.NetworkConfigList, 0)
	}
	if m.runtimeConf == nil {
		m.runtimeConf = make([]*libcni.RuntimeConf, 0)
	}
	config := &libcni.CNIConfig{Path: []string{m.cniPath.Plugin}}

	if command == "ADD" {
		ifIndex := 0
		for _, config := range m.configs {
			capabilityArgs := make(map[string]interface{}, 0)
			portMappings := make([]map[string]interface{}, 0)

			nlist, err := libcni.LoadConfList(m.cniPath.Conf, config.name)
			if err != nil {
				return err
			}
			m.networkConfList = append(m.networkConfList, nlist)
			ifName := fmt.Sprintf("eth%d", ifIndex)

			for _, portMap := range config.portMap {
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
				ContainerID:    m.containerID,
				NetNS:          m.netNS,
				IfName:         ifName,
				CapabilityArgs: capabilityArgs,
				Args:           config.args,
			}
			m.runtimeConf = append(m.runtimeConf, rt)
			ifIndex++
		}
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
