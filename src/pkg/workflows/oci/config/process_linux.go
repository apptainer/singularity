package config

import (
	"fmt"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/singularityware/singularity/src/pkg/util/capabilities"
	"strings"
)

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

func (c *DefaultRuntimeOciProcess) GetBoundingCapabilities() []string {
	c.initCapabilities()
	return c.RuntimeOciSpec.Process.Capabilities.Bounding
}

func (c *DefaultRuntimeOciProcess) SetBoundingCapabilities(capabilities []string) error {
	c.initCapabilities()
	for _, capability := range capabilities {
		if err := c.AddBoundingCapability(capability); err != nil {
			return err
		}
	}
	return nil
}

func (c *DefaultRuntimeOciProcess) AddBoundingCapability(capability string) error {
	c.initCapabilities()
	return addCapability(capability, &c.RuntimeOciSpec.Process.Capabilities.Bounding, boundingCapabilities)
}

func (c *DefaultRuntimeOciProcess) DelBoundingCapability(capability string) error {
	c.initCapabilities()
	return delCapability(capability, &c.RuntimeOciSpec.Process.Capabilities.Bounding, boundingCapabilities)
}

func (c *DefaultRuntimeOciProcess) GetEffectiveCapabilities() []string {
	c.initCapabilities()
	return c.RuntimeOciSpec.Process.Capabilities.Effective
}

func (c *DefaultRuntimeOciProcess) SetEffectiveCapabilities(capabilities []string) error {
	c.initCapabilities()
	for _, capability := range capabilities {
		if err := c.AddEffectiveCapability(capability); err != nil {
			return err
		}
	}
	return nil
}

func (c *DefaultRuntimeOciProcess) AddEffectiveCapability(capability string) error {
	c.initCapabilities()
	return addCapability(capability, &c.RuntimeOciSpec.Process.Capabilities.Effective, effectiveCapabilities)
}

func (c *DefaultRuntimeOciProcess) DelEffectiveCapability(capability string) error {
	c.initCapabilities()
	return delCapability(capability, &c.RuntimeOciSpec.Process.Capabilities.Effective, effectiveCapabilities)
}

func (c *DefaultRuntimeOciProcess) GetInheritableCapabilities() []string {
	c.initCapabilities()
	return []string{}
}

func (c *DefaultRuntimeOciProcess) SetInheritableCapabilities(capabilities []string) error {
	c.initCapabilities()
	for _, capability := range capabilities {
		if err := c.AddInheritableCapability(capability); err != nil {
			return err
		}
	}
	return nil
}

func (c *DefaultRuntimeOciProcess) AddInheritableCapability(capability string) error {
	c.initCapabilities()
	return addCapability(capability, &c.RuntimeOciSpec.Process.Capabilities.Inheritable, inheritableCapabilities)
}

func (c *DefaultRuntimeOciProcess) DelInheritableCapability(capability string) error {
	c.initCapabilities()
	return delCapability(capability, &c.RuntimeOciSpec.Process.Capabilities.Inheritable, inheritableCapabilities)
}

func (c *DefaultRuntimeOciProcess) GetPermittedCapabilities() []string {
	c.initCapabilities()
	return []string{}
}

func (c *DefaultRuntimeOciProcess) SetPermittedCapabilities(capabilities []string) error {
	c.initCapabilities()
	for _, capability := range capabilities {
		if err := c.AddPermittedCapability(capability); err != nil {
			return err
		}
	}
	return nil
}

func (c *DefaultRuntimeOciProcess) AddPermittedCapability(capability string) error {
	c.initCapabilities()
	return addCapability(capability, &c.RuntimeOciSpec.Process.Capabilities.Permitted, permittedCapabilities)
}

func (c *DefaultRuntimeOciProcess) DelPermittedCapability(capability string) error {
	c.initCapabilities()
	return delCapability(capability, &c.RuntimeOciSpec.Process.Capabilities.Permitted, permittedCapabilities)
}

func (c *DefaultRuntimeOciProcess) GetAmbientCapabilities() []string {
	c.initCapabilities()
	return []string{}
}

func (c *DefaultRuntimeOciProcess) SetAmbientCapabilities(capabilities []string) error {
	c.initCapabilities()
	for _, capability := range capabilities {
		if err := c.AddAmbientCapability(capability); err != nil {
			return err
		}
	}
	return nil
}

func (c *DefaultRuntimeOciProcess) AddAmbientCapability(capability string) error {
	c.initCapabilities()
	return addCapability(capability, &c.RuntimeOciSpec.Process.Capabilities.Ambient, ambientCapabilities)
}

func (c *DefaultRuntimeOciProcess) DelAmbientCapability(capability string) error {
	c.initCapabilities()
	return delCapability(capability, &c.RuntimeOciSpec.Process.Capabilities.Ambient, ambientCapabilities)
}

func (c *DefaultRuntimeOciProcess) GetNoNewPrivileges() bool {
	c.init()
	return c.RuntimeOciSpec.Process.NoNewPrivileges
}

func (c *DefaultRuntimeOciProcess) SetNoNewPrivileges(enable bool) {
	c.init()
	c.RuntimeOciSpec.Process.NoNewPrivileges = enable
}

func (c *DefaultRuntimeOciProcess) GetApparmorProfile() string {
	c.init()
	return c.RuntimeOciSpec.Process.ApparmorProfile
}

func (c *DefaultRuntimeOciProcess) SetApparmorProfile(profile string) {
	c.init()
	c.RuntimeOciSpec.Process.ApparmorProfile = profile
}

func (c *DefaultRuntimeOciProcess) GetSelinuxLabel() string {
	c.init()
	return c.RuntimeOciSpec.Process.SelinuxLabel
}

func (c *DefaultRuntimeOciProcess) SetSelinuxLabel(label string) {
	c.init()
	c.RuntimeOciSpec.Process.SelinuxLabel = label
}

func (c *DefaultRuntimeOciProcess) GetOOMScoreAdj() *int {
	c.init()
	return c.RuntimeOciSpec.Process.OOMScoreAdj
}

func (c *DefaultRuntimeOciProcess) SetOOMScoreAdj(score int) {
	c.init()
	c.RuntimeOciSpec.Process.OOMScoreAdj = &score
}

func (c *DefaultRuntimeOciProcess) GetRlimits() []specs.POSIXRlimit {
	c.init()
	return c.RuntimeOciSpec.Process.Rlimits
}

func (c *DefaultRuntimeOciProcess) SetRlimits(limits []specs.POSIXRlimit) error {
	c.init()
	return nil
}

func (c *DefaultRuntimeOciProcess) AddRlimit(rtype string, hard uint64, soft uint64) error {
	c.init()
	return nil
}

func (c *DefaultRuntimeOciProcess) DelRlimit(rtype string) error {
	c.init()
	return nil
}
