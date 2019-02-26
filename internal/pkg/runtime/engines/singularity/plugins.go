// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

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
	_ "github.com/containernetworking/plugins/plugins/meta/flannel"
	_ "github.com/containernetworking/plugins/plugins/meta/portmap"
	_ "github.com/containernetworking/plugins/plugins/meta/tuning"
)
