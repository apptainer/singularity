package config

import (
	"github.com/opencontainers/runtime-spec/specs-go"
)

type RuntimeOciSpec specs.Spec

type RuntimeOciConfig struct {
	RuntimeOciSpec
	Version     RuntimeOciVersion
	Process     RuntimeOciProcess
	Root        RuntimeOciRoot
	Hostname    RuntimeOciHostname
	Mounts      RuntimeOciMounts
	Hooks       RuntimeOciHooks
	Annotations RuntimeOciAnnotations
	RuntimeOciPlatform
}
