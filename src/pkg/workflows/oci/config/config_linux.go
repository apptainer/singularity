package config

const Platform = "linux"

type RuntimeOciPlatform struct {
	Linux   RuntimeOciLinux
	Solaris interface{}
	Windows interface{}
}
