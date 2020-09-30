// Copyright (c) 2020, Control Command Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package endpoint

import (
	"fmt"
	"net/http"
	"time"

	golog "github.com/go-log/log"
	buildclient "github.com/sylabs/scs-build-client/client"
	keyclient "github.com/sylabs/scs-key-client/client"
	libclient "github.com/sylabs/scs-library-client/client"
	"github.com/sylabs/singularity/pkg/sylog"
	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
)

func (ep *Config) KeyserverClientConfig(uri string, op KeyserverOp) (*keyclient.Config, error) {
	// empty uri means to use the default endpoint
	isDefault := uri == ""

	config := &keyclient.Config{
		BaseURL:   uri,
		UserAgent: useragent.Value(),
	}

	if err := ep.UpdateKeyserversConfig(); err != nil {
		return nil, err
	}

	var primaryKeyserver *ServiceConfig

	for _, kc := range ep.Keyservers {
		if kc.Skip {
			continue
		}
		primaryKeyserver = kc
		break
	}

	// shouldn't happen
	if primaryKeyserver == nil {
		return nil, fmt.Errorf("no primary keyserver configured")
	}

	var keyservers []*ServiceConfig

	if isDefault {
		config.BaseURL = primaryKeyserver.URI

		if op == KeyserverVerifyOp {
			// verify operation can query multiple keyserver, the token
			// is automatically set by the custom client
			keyservers = ep.Keyservers
		} else {
			// use the primary keyserver
			keyservers = []*ServiceConfig{
				primaryKeyserver,
			}
		}
	} else {
		keyservers = []*ServiceConfig{
			{
				URI:      uri,
				External: true,
			},
		}
	}

	config.HTTPClient = newClient(keyservers, op)

	return config, nil
}

func (ep *Config) LibraryClientConfig(uri string) (*libclient.Config, error) {
	// empty uri means to use the default endpoint
	isDefault := uri == ""

	config := &libclient.Config{
		BaseURL:   uri,
		UserAgent: useragent.Value(),
		Logger:    (golog.Logger)(sylog.DebugLogger{}),
	}

	if isDefault {
		libURI, err := ep.GetServiceURI(Library)
		if err != nil {
			return nil, fmt.Errorf("unable to get library service URI: %v", err)
		}
		config.AuthToken = ep.Token
		config.BaseURL = libURI
	}

	return config, nil
}

func (ep *Config) BuilderClientConfig(uri string) (*buildclient.Config, error) {
	// empty uri means to use the default endpoint
	isDefault := uri == ""

	config := &buildclient.Config{
		BaseURL:   uri,
		UserAgent: useragent.Value(),
		Logger:    (golog.Logger)(sylog.DebugLogger{}),
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	if isDefault {
		buildURI, err := ep.GetServiceURI(Builder)
		if err != nil {
			return nil, fmt.Errorf("unable to get builder service URI: %v", err)
		}
		config.AuthToken = ep.Token
		config.BaseURL = buildURI
	}

	return config, nil
}
