// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package config

import (
	"fmt"
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/singularityware/singularity/src/pkg/util/capabilities"
)

// ProcessPlatform describes the platform process interface.
type ProcessPlatform interface {
	GetBoundingCapabilities() []string
	SetBoundingCapabilities([]string) error
	AddBoundingCapability(capability string) error
	DelBoundingCapability(capability string) error

	GetEffectiveCapabilities() []string
	SetEffectiveCapabilities([]string) error
	AddEffectiveCapability(capability string) error
	DelEffectiveCapability(capability string) error

	GetInheritableCapabilities() []string
	SetInheritableCapabilities([]string) error
	AddInheritableCapability(capability string) error
	DelInheritableCapability(capability string) error

	GetPermittedCapabilities() []string
	SetPermittedCapabilities([]string) error
	AddPermittedCapability(capability string) error
	DelPermittedCapability(capability string) error

	GetAmbientCapabilities() []string
	SetAmbientCapabilities([]string) error
	AddAmbientCapability(capability string) error
	DelAmbientCapability(capability string) error

	GetNoNewPrivileges() bool
	SetNoNewPrivileges(enable bool)

	GetApparmorProfile() string
	SetApparmorProfile(profile string)

	GetSelinuxLabel() string
	SetSelinuxLabel(label string)

	GetOOMScoreAdj() *int
	SetOOMScoreAdj(score int)

	GetRlimits() []specs.POSIXRlimit
	SetRlimits(limits []specs.POSIXRlimit) error
	AddRlimit(rtype string, hard uint64, soft uint64) error
	DelRlimit(rtype string) error
}

var boundingCapabilities = map[string]int{}
var effectiveCapabilities = map[string]int{}
var permittedCapabilities = map[string]int{}
var inheritableCapabilities = map[string]int{}
var ambientCapabilities = map[string]int{}

func addCapability(capability string, set *[]string, capabilityMap map[string]int) error {
	uppercap := strings.ToUpper(capability)
	if strings.HasPrefix(uppercap, "CAP_") == false {
		uppercap = "CAP_" + uppercap
	}
	if capabilities.Map[uppercap] == nil {
		return fmt.Errorf("no capability found for %s", capability)
	}
	if _, present := capabilityMap[uppercap]; present {
		return nil
	}
	capabilityMap[capability] = len(*set)
	*set = append(*set, capability)
	return nil
}

func delCapability(capability string, set *[]string, capabilityMap map[string]int) error {
	uppercap := strings.ToUpper(capability)
	if strings.HasPrefix(uppercap, "CAP_") == false {
		uppercap = "CAP_" + uppercap
	}
	if capabilities.Map[uppercap] != nil {
		if idx, present := capabilityMap[uppercap]; present {
			tmp := *set
			*set = append(tmp[:idx], tmp[idx+1:]...)
			delete(capabilityMap, uppercap)
			return nil
		}
	}
	return fmt.Errorf("no capability found for %s", uppercap)
}

func (c *DefaultRuntimeOciProcess) initCapabilities() {
	c.init()
	if c.RuntimeOciSpec.Process.Capabilities == nil {
		c.RuntimeOciSpec.Process.Capabilities = &specs.LinuxCapabilities{}
	}
}

// GetBoundingCapabilities retrieves the bounding capabilities.
func (c *DefaultRuntimeOciProcess) GetBoundingCapabilities() []string {
	c.initCapabilities()
	return c.RuntimeOciSpec.Process.Capabilities.Bounding
}

// SetBoundingCapabilities sets the bounding capabilities.
func (c *DefaultRuntimeOciProcess) SetBoundingCapabilities(capabilities []string) error {
	c.initCapabilities()
	for _, capability := range capabilities {
		if err := c.AddBoundingCapability(capability); err != nil {
			return err
		}
	}
	return nil
}

// AddBoundingCapability adds a bounding capability.
func (c *DefaultRuntimeOciProcess) AddBoundingCapability(capability string) error {
	c.initCapabilities()
	return addCapability(capability, &c.RuntimeOciSpec.Process.Capabilities.Bounding, boundingCapabilities)
}

// DelBoundingCapability deletes a bounding capability.
func (c *DefaultRuntimeOciProcess) DelBoundingCapability(capability string) error {
	c.initCapabilities()
	return delCapability(capability, &c.RuntimeOciSpec.Process.Capabilities.Bounding, boundingCapabilities)
}

// GetEffectiveCapabilities retrieves the effective capabilities.
func (c *DefaultRuntimeOciProcess) GetEffectiveCapabilities() []string {
	c.initCapabilities()
	return c.RuntimeOciSpec.Process.Capabilities.Effective
}

// SetEffectiveCapabilities sets the effective capabilities.
func (c *DefaultRuntimeOciProcess) SetEffectiveCapabilities(capabilities []string) error {
	c.initCapabilities()
	for _, capability := range capabilities {
		if err := c.AddEffectiveCapability(capability); err != nil {
			return err
		}
	}
	return nil
}

// AddEffectiveCapability adds an effective capability.
func (c *DefaultRuntimeOciProcess) AddEffectiveCapability(capability string) error {
	c.initCapabilities()
	return addCapability(capability, &c.RuntimeOciSpec.Process.Capabilities.Effective, effectiveCapabilities)
}

// DelEffectiveCapability deletes an effective capability.
func (c *DefaultRuntimeOciProcess) DelEffectiveCapability(capability string) error {
	c.initCapabilities()
	return delCapability(capability, &c.RuntimeOciSpec.Process.Capabilities.Effective, effectiveCapabilities)
}

// GetInheritableCapabilities retrieves the inheritable capabilities.
func (c *DefaultRuntimeOciProcess) GetInheritableCapabilities() []string {
	c.initCapabilities()
	return []string{}
}

// SetInheritableCapabilities sets the inheritable capabilities.
func (c *DefaultRuntimeOciProcess) SetInheritableCapabilities(capabilities []string) error {
	c.initCapabilities()
	for _, capability := range capabilities {
		if err := c.AddInheritableCapability(capability); err != nil {
			return err
		}
	}
	return nil
}

// AddInheritableCapability adds an inheritable capability.
func (c *DefaultRuntimeOciProcess) AddInheritableCapability(capability string) error {
	c.initCapabilities()
	return addCapability(capability, &c.RuntimeOciSpec.Process.Capabilities.Inheritable, inheritableCapabilities)
}

// DelInheritableCapability deletes an inheritable capability.
func (c *DefaultRuntimeOciProcess) DelInheritableCapability(capability string) error {
	c.initCapabilities()
	return delCapability(capability, &c.RuntimeOciSpec.Process.Capabilities.Inheritable, inheritableCapabilities)
}

// GetPermittedCapabilities retrieves the permitted capabilities.
func (c *DefaultRuntimeOciProcess) GetPermittedCapabilities() []string {
	c.initCapabilities()
	return []string{}
}

// SetPermittedCapabilities sets the permitted capabilities.
func (c *DefaultRuntimeOciProcess) SetPermittedCapabilities(capabilities []string) error {
	c.initCapabilities()
	for _, capability := range capabilities {
		if err := c.AddPermittedCapability(capability); err != nil {
			return err
		}
	}
	return nil
}

// AddPermittedCapability adds a permitted capability.
func (c *DefaultRuntimeOciProcess) AddPermittedCapability(capability string) error {
	c.initCapabilities()
	return addCapability(capability, &c.RuntimeOciSpec.Process.Capabilities.Permitted, permittedCapabilities)
}

// DelPermittedCapability deletes a permitted capability.
func (c *DefaultRuntimeOciProcess) DelPermittedCapability(capability string) error {
	c.initCapabilities()
	return delCapability(capability, &c.RuntimeOciSpec.Process.Capabilities.Permitted, permittedCapabilities)
}

// GetAmbientCapabilities retrieves the ambient capabilities.
func (c *DefaultRuntimeOciProcess) GetAmbientCapabilities() []string {
	c.initCapabilities()
	return []string{}
}

// SetAmbientCapabilities sets the ambient capabilities.
func (c *DefaultRuntimeOciProcess) SetAmbientCapabilities(capabilities []string) error {
	c.initCapabilities()
	for _, capability := range capabilities {
		if err := c.AddAmbientCapability(capability); err != nil {
			return err
		}
	}
	return nil
}

// AddAmbientCapability adds an ambient capability.
func (c *DefaultRuntimeOciProcess) AddAmbientCapability(capability string) error {
	c.initCapabilities()
	return addCapability(capability, &c.RuntimeOciSpec.Process.Capabilities.Ambient, ambientCapabilities)
}

// DelAmbientCapability deletes an ambient capability.
func (c *DefaultRuntimeOciProcess) DelAmbientCapability(capability string) error {
	c.initCapabilities()
	return delCapability(capability, &c.RuntimeOciSpec.Process.Capabilities.Ambient, ambientCapabilities)
}

// GetNoNewPrivileges gets the no new privileges flag.
func (c *DefaultRuntimeOciProcess) GetNoNewPrivileges() bool {
	c.init()
	return c.RuntimeOciSpec.Process.NoNewPrivileges
}

// SetNoNewPrivileges sets the no new privileges flag.
func (c *DefaultRuntimeOciProcess) SetNoNewPrivileges(enable bool) {
	c.init()
	c.RuntimeOciSpec.Process.NoNewPrivileges = enable
}

// GetApparmorProfile gets the apparmor profile.
func (c *DefaultRuntimeOciProcess) GetApparmorProfile() string {
	c.init()
	return c.RuntimeOciSpec.Process.ApparmorProfile
}

// SetApparmorProfile sets the apparmor profile.
func (c *DefaultRuntimeOciProcess) SetApparmorProfile(profile string) {
	c.init()
	c.RuntimeOciSpec.Process.ApparmorProfile = profile
}

// GetSelinuxLabel gets the selinux label.
func (c *DefaultRuntimeOciProcess) GetSelinuxLabel() string {
	c.init()
	return c.RuntimeOciSpec.Process.SelinuxLabel
}

// SetSelinuxLabel sets the selinux label.
func (c *DefaultRuntimeOciProcess) SetSelinuxLabel(label string) {
	c.init()
	c.RuntimeOciSpec.Process.SelinuxLabel = label
}

// GetOOMScoreAdj gets the OOM score adjustment value.
func (c *DefaultRuntimeOciProcess) GetOOMScoreAdj() *int {
	c.init()
	return c.RuntimeOciSpec.Process.OOMScoreAdj
}

// SetOOMScoreAdj sets the OOM score adjustment value.
func (c *DefaultRuntimeOciProcess) SetOOMScoreAdj(score int) {
	c.init()
	c.RuntimeOciSpec.Process.OOMScoreAdj = &score
}

// GetRlimits gets the POSIX rlimits.
func (c *DefaultRuntimeOciProcess) GetRlimits() []specs.POSIXRlimit {
	c.init()
	return c.RuntimeOciSpec.Process.Rlimits
}

// SetRlimits sets the POSIX rlimits.
func (c *DefaultRuntimeOciProcess) SetRlimits(limits []specs.POSIXRlimit) error {
	c.init()
	return nil
}

// AddRlimit adds a POSIX rlimit.
func (c *DefaultRuntimeOciProcess) AddRlimit(rtype string, hard uint64, soft uint64) error {
	c.init()
	return nil
}

// DelRlimit deletes a POSIX rlimit.
func (c *DefaultRuntimeOciProcess) DelRlimit(rtype string) error {
	c.init()
	return nil
}
