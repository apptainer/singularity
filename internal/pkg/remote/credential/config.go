// Copyright (c) 2020, Control Command Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package credential

const (
	// BasicPrefix is the prefix for the HTTP basic authentication.
	BasicPrefix = "Basic "
	// TokenPrefix is the prefix for the HTTP token/bearer authentication.
	TokenPrefix = "Bearer "
)

// Config holds credential configuration for a service.
type Config struct {
	URI string `yaml:"URI"`
	// Can take the form of:
	// - "Basic <base64 encoded username:passwd>"
	// - "Bearer <token>"
	// An empty value means there no authentication at all
	// or that credentials are stored elsewhere
	Auth     string `yaml:"Auth,omitempty"`
	Insecure bool   `yaml:"Insecure"`
}
