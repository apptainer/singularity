// Copyright (c) 2019-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// The following build tag is used to _exclude_ this file from builds,
// as it exists solely to add the listed packages are dependencies so
// that they get added to the toplevel go.mod file.
//
// If you need to add a new plugin, simply add it to this import list.
// The build system will pick it up from here.

//go:build cni_plugins
// +build cni_plugins

package singularity

import (
	_ "github.com/containernetworking/plugins/plugins/ipam/dhcp"
	_ "github.com/containernetworking/plugins/plugins/ipam/host-local"
	_ "github.com/containernetworking/plugins/plugins/ipam/static"
	_ "github.com/containernetworking/plugins/plugins/main/bridge"
	_ "github.com/containernetworking/plugins/plugins/main/host-device"
	_ "github.com/containernetworking/plugins/plugins/main/ipvlan"
	_ "github.com/containernetworking/plugins/plugins/main/loopback"
	_ "github.com/containernetworking/plugins/plugins/main/macvlan"
	_ "github.com/containernetworking/plugins/plugins/main/ptp"
	_ "github.com/containernetworking/plugins/plugins/main/vlan"
	_ "github.com/containernetworking/plugins/plugins/meta/bandwidth"
	_ "github.com/containernetworking/plugins/plugins/meta/firewall"
	_ "github.com/containernetworking/plugins/plugins/meta/portmap"
	_ "github.com/containernetworking/plugins/plugins/meta/sbr"
	_ "github.com/containernetworking/plugins/plugins/meta/tuning"
	_ "github.com/containernetworking/plugins/plugins/meta/vrf"
)
