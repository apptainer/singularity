// Copyright (c) 2020, Control Command Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package credential

import (
	"fmt"
	"net/url"
)

// Manager handle login/logout handlers.
var Manager = new(manager)

type manager struct{}

// Login allows to log into a service like a Docker/OCI registry or a keyserver.
func (m *manager) Login(uri, username, password string, insecure bool) (*Config, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	if handler, ok := loginHandlers[u.Scheme]; ok {
		return handler.login(u, username, password, insecure)
	}

	return nil, fmt.Errorf("%s transport is not supported", u.Scheme)
}

// Logout allows to log out from a service like a Docker/OCI registry or a keyserver.
func (m *manager) Logout(uri string) error {
	u, err := url.Parse(uri)
	if err != nil {
		return err
	}

	if handler, ok := loginHandlers[u.Scheme]; ok {
		return handler.logout(u)
	}

	return fmt.Errorf("%s transport is not supported", u.Scheme)
}
