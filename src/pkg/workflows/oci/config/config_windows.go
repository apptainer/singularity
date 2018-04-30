package config

const Platform = "windows"

type RuntimeOciPlatform struct {
	Linux   interface{}
	Solaris interface{}
	Windows RuntimeOciWindows
}
