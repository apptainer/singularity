package config

const Platform = "solaris"

type RuntimeOciPlatform struct {
    Linux           interface{}
    Solaris         RuntimeOciSolaris
    Windows         interface{}
}
