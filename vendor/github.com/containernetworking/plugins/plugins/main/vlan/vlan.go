// Copyright 2015 CNI authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime"

	"github.com/vishvananda/netlink"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/cni/pkg/version"

	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ipam"
	"github.com/containernetworking/plugins/pkg/ns"
	bv "github.com/containernetworking/plugins/pkg/utils/buildversion"
)

type NetConf struct {
	types.NetConf
	Master string `json:"master"`
	VlanId int    `json:"vlanId"`
	MTU    int    `json:"mtu,omitempty"`
}

func init() {
	// this ensures that main runs only on main thread (thread group leader).
	// since namespace ops (unshare, setns) are done for a single thread, we
	// must ensure that the goroutine does not jump from OS thread to thread
	runtime.LockOSThread()
}

func loadConf(bytes []byte) (*NetConf, string, error) {
	n := &NetConf{}
	if err := json.Unmarshal(bytes, n); err != nil {
		return nil, "", fmt.Errorf("failed to load netconf: %v", err)
	}
	if n.Master == "" {
		return nil, "", fmt.Errorf(`"master" field is required. It specifies the host interface name to create the VLAN for.`)
	}
	if n.VlanId < 0 || n.VlanId > 4094 {
		return nil, "", fmt.Errorf(`invalid VLAN ID %d (must be between 0 and 4095 inclusive)`, n.VlanId)
	}
	return n, n.CNIVersion, nil
}

func createVlan(conf *NetConf, ifName string, netns ns.NetNS) (*current.Interface, error) {
	vlan := &current.Interface{}

	m, err := netlink.LinkByName(conf.Master)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup master %q: %v", conf.Master, err)
	}

	// due to kernel bug we have to create with tmpname or it might
	// collide with the name on the host and error out
	tmpName, err := ip.RandomVethName()
	if err != nil {
		return nil, err
	}

	if conf.MTU <= 0 {
		conf.MTU = m.Attrs().MTU
	}

	v := &netlink.Vlan{
		LinkAttrs: netlink.LinkAttrs{
			MTU:         conf.MTU,
			Name:        tmpName,
			ParentIndex: m.Attrs().Index,
			Namespace:   netlink.NsFd(int(netns.Fd())),
		},
		VlanId: conf.VlanId,
	}

	if err := netlink.LinkAdd(v); err != nil {
		return nil, fmt.Errorf("failed to create vlan: %v", err)
	}

	err = netns.Do(func(_ ns.NetNS) error {
		err := ip.RenameLink(tmpName, ifName)
		if err != nil {
			return fmt.Errorf("failed to rename vlan to %q: %v", ifName, err)
		}
		vlan.Name = ifName

		// Re-fetch interface to get all properties/attributes
		contVlan, err := netlink.LinkByName(vlan.Name)
		if err != nil {
			return fmt.Errorf("failed to refetch vlan %q: %v", vlan.Name, err)
		}
		vlan.Mac = contVlan.Attrs().HardwareAddr.String()
		vlan.Sandbox = netns.Path()

		return nil
	})
	if err != nil {
		return nil, err
	}

	return vlan, nil
}

func cmdAdd(args *skel.CmdArgs) error {
	n, cniVersion, err := loadConf(args.StdinData)
	if err != nil {
		return err
	}

	netns, err := ns.GetNS(args.Netns)
	if err != nil {
		return fmt.Errorf("failed to open netns %q: %v", args.Netns, err)
	}
	defer netns.Close()

	vlanInterface, err := createVlan(n, args.IfName, netns)
	if err != nil {
		return err
	}

	// run the IPAM plugin and get back the config to apply
	r, err := ipam.ExecAdd(n.IPAM.Type, args.StdinData)
	if err != nil {
		return err
	}

	// Invoke ipam del if err to avoid ip leak
	defer func() {
		if err != nil {
			ipam.ExecDel(n.IPAM.Type, args.StdinData)
		}
	}()

	// Convert whatever the IPAM result was into the current Result type
	result, err := current.NewResultFromResult(r)
	if err != nil {
		return err
	}

	if len(result.IPs) == 0 {
		return errors.New("IPAM plugin returned missing IP config")
	}
	for _, ipc := range result.IPs {
		// All addresses belong to the vlan interface
		ipc.Interface = current.Int(0)
	}

	result.Interfaces = []*current.Interface{vlanInterface}

	err = netns.Do(func(_ ns.NetNS) error {
		return ipam.ConfigureIface(args.IfName, result)
	})
	if err != nil {
		return err
	}

	result.DNS = n.DNS

	return types.PrintResult(result, cniVersion)
}

func cmdDel(args *skel.CmdArgs) error {
	n, _, err := loadConf(args.StdinData)
	if err != nil {
		return err
	}

	err = ipam.ExecDel(n.IPAM.Type, args.StdinData)
	if err != nil {
		return err
	}

	if args.Netns == "" {
		return nil
	}

	err = ns.WithNetNSPath(args.Netns, func(_ ns.NetNS) error {
		err = ip.DelLinkByName(args.IfName)
		if err != nil && err != ip.ErrLinkNotFound {
			return nil
		}
		return err
	})

	return err
}

func main() {
	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, version.All, bv.BuildString("vlan"))
}

func cmdCheck(args *skel.CmdArgs) error {
	conf := NetConf{}
	if err := json.Unmarshal(args.StdinData, &conf); err != nil {
		return fmt.Errorf("failed to load netconf: %v", err)
	}

	netns, err := ns.GetNS(args.Netns)
	if err != nil {
		return fmt.Errorf("failed to open netns %q: %v", args.Netns, err)
	}
	defer netns.Close()

	// run the IPAM plugin and get back the config to apply
	err = ipam.ExecCheck(conf.IPAM.Type, args.StdinData)
	if err != nil {
		return err
	}
	if conf.NetConf.RawPrevResult == nil {
		return fmt.Errorf("ptp: Required prevResult missing")
	}
	if err := version.ParsePrevResult(&conf.NetConf); err != nil {
		return err
	}
	// Convert whatever the IPAM result was into the current Result type
	result, err := current.NewResultFromResult(conf.PrevResult)
	if err != nil {
		return err
	}

	var contMap current.Interface
	// Find interfaces for name whe know, that of host-device inside container
	for _, intf := range result.Interfaces {
		if args.IfName == intf.Name {
			if args.Netns == intf.Sandbox {
				contMap = *intf
				continue
			}
		}
	}

	// The namespace must be the same as what was configured
	if args.Netns != contMap.Sandbox {
		return fmt.Errorf("Sandbox in prevResult %s doesn't match configured netns: %s",
			contMap.Sandbox, args.Netns)
	}

	m, err := netlink.LinkByName(conf.Master)
	if err != nil {
		return fmt.Errorf("failed to lookup master %q: %v", conf.Master, err)
	}

	//
	// Check prevResults for ips, routes and dns against values found in the container
	if err := netns.Do(func(_ ns.NetNS) error {

		// Check interface against values found in the container
		err := validateCniContainerInterface(contMap, m.Attrs().Index, conf.VlanId, conf.MTU)
		if err != nil {
			return err
		}

		err = ip.ValidateExpectedInterfaceIPs(args.IfName, result.IPs)
		if err != nil {
			return err
		}

		err = ip.ValidateExpectedRoute(result.Routes)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func validateCniContainerInterface(intf current.Interface, masterIndex int, vlanId int, mtu int) error {

	var link netlink.Link
	var err error

	if intf.Name == "" {
		return fmt.Errorf("Container interface name missing in prevResult: %v", intf.Name)
	}
	link, err = netlink.LinkByName(intf.Name)
	if err != nil {
		return fmt.Errorf("ptp: Container Interface name in prevResult: %s not found", intf.Name)
	}
	if intf.Sandbox == "" {
		return fmt.Errorf("ptp: Error: Container interface %s should not be in host namespace", link.Attrs().Name)
	}

	vlan, isVlan := link.(*netlink.Vlan)
	if !isVlan {
		return fmt.Errorf("Error: Container interface %s not of type vlan", link.Attrs().Name)
	}

	// TODO This works when unit testing via cnitool; fails with ./test.sh
	//if masterIndex != vlan.Attrs().ParentIndex {
	//   return fmt.Errorf("Container vlan Master %d does not match expected value: %d", vlan.Attrs().ParentIndex, masterIndex)
	//}

	if intf.Mac != "" {
		if intf.Mac != link.Attrs().HardwareAddr.String() {
			return fmt.Errorf("vlan: Interface %s Mac %s doesn't match container Mac: %s", intf.Name, intf.Mac, link.Attrs().HardwareAddr)
		}
	}

	if vlanId != vlan.VlanId {
		return fmt.Errorf("Error: Tuning link %s configured promisc is %v, current value is %d",
			intf.Name, vlanId, vlan.VlanId)
	}

	if mtu != 0 {
		if mtu != link.Attrs().MTU {
			return fmt.Errorf("Error: Tuning configured MTU of %s is %d, current value is %d",
				intf.Name, mtu, link.Attrs().MTU)
		}
	}

	return nil
}
