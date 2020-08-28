// Copyright (c) 2020, Control Command Inc. All rights reserved.
// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package endpoint

import (
	"github.com/sylabs/singularity/internal/pkg/remote/credential"
)

var DefaultEndpointConfig = &Config{
	URI:    SCSDefaultCloudURI,
	System: true,
}

// Config describes a single remote endpoint.
type Config struct {
	URI        string           `yaml:"URI,omitempty"`
	Token      string           `yaml:"Token,omitempty"`
	System     bool             `yaml:"System"`    // Was this EndPoint set from system config file
	Exclusive  bool             `yaml:"Exclusive"` // true if the endpoint must be used exclusively
	Keyservers []*ServiceConfig `yaml:"Keyservers,omitempty"`

	// for internal purpose
	credentials []*credential.Config
	services    map[string][]Service
}

func (e *Config) SetCredentials(creds []*credential.Config) {
	e.credentials = creds
}

type ServiceConfig struct {
	// for internal purpose
	credential *credential.Config

	URI      string `yaml:"URI"`
	Skip     bool   `yaml:"Skip"`
	External bool   `yaml:"External"`
	Insecure bool   `yaml:"Insecure"`
}
