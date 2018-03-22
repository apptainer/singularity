package config

import(
    "fmt"
)

type RuntimeOciHostname interface {
    Get() string
    Set(hostname string)
}

type DefaultRuntimeOciHostname struct {
    RuntimeOciSpec *RuntimeOciSpec
}

func (c *DefaultRuntimeOciHostname) Get() string {
    fmt.Println("Get hostname")
    return c.RuntimeOciSpec.Hostname
}

func (c *DefaultRuntimeOciHostname) Set(hostname string) {
    fmt.Println("Set hostname to", hostname)
    c.RuntimeOciSpec.Hostname = hostname
}
