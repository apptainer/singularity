package config

import(
    "fmt"
)

type RuntimeOciVersion interface {
    Get() string
    Set(name string)
}

type DefaultRuntimeOciVersion struct {
    RuntimeOciSpec *RuntimeOciSpec
}

func (c *DefaultRuntimeOciVersion) Get() string {
    fmt.Println("Get version")
    return c.RuntimeOciSpec.Version
}

func (c *DefaultRuntimeOciVersion) Set(version string) {
    fmt.Println("Set version to", version)
    c.RuntimeOciSpec.Version = version
}
