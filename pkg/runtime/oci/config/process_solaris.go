package config

import (
    "github.com/opencontainers/runtime-spec/specs-go"
)

type ProcessPlatform interface {
    GetRlimits() []specs.POSIXRlimit
    SetRlimits(limits []specs.POSIXRlimit) error
    AddRlimit(rtype string, hard uint64, soft uint64) error
    DelRlimit(rtype string) error
}
