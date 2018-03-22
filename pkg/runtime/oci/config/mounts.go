package config

import (
    "github.com/opencontainers/runtime-spec/specs-go"
)

type RuntimeOciMounts interface {
    GetSpec() *specs.Mount

    GetMounts() []specs.Mount
    SetMounts(mounts []specs.Mount) error
    AddMount(destination string, mounttype string, source string, options []string) error
    DelMount(destination string) error
}
