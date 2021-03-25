// Copyright (c) 2020, Control Command Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package endpoint

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	golog "github.com/go-log/log"
	buildclient "github.com/sylabs/scs-build-client/client"
	keyclient "github.com/sylabs/scs-key-client/client"
	libclient "github.com/sylabs/scs-library-client/client"
	remoteutil "github.com/sylabs/singularity/internal/pkg/remote/util"
	"github.com/sylabs/singularity/pkg/sylog"
	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
)

func (ep *Config) KeyserverClientOpts(uri string, op KeyserverOp) ([]keyclient.Option, error) {
	// empty uri means to use the default endpoint
	isDefault := uri == ""

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
		uri = primaryKeyserver.URI

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
	} else if ep.Exclusive {
		available := make([]string, 0)
		found := false
		for _, kc := range ep.Keyservers {
			if kc.Skip {
				continue
			}
			available = append(available, kc.URI)
			if remoteutil.SameKeyserver(uri, kc.URI) {
				found = true
				break
			}
		}
		if !found {
			list := strings.Join(available, ", ")
			return nil, fmt.Errorf(
				"endpoint is set as exclusive by the system administrator: only %q can be used",
				list,
			)
		}
	} else {
		keyservers = []*ServiceConfig{
			{
				URI:      uri,
				External: true,
			},
		}
	}

	co := []keyclient.Option{
		keyclient.OptBaseURL(uri),
		keyclient.OptUserAgent(useragent.Value()),
		keyclient.OptHTTPClient(newClient(keyservers, op)),
	}
	return co, nil
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
	} else if ep.Exclusive {
		libURI, err := ep.GetServiceURI(Library)
		if err != nil {
			return nil, fmt.Errorf("unable to get library service URI: %v", err)
		}
		if !remoteutil.SameURI(uri, libURI) {
			return nil, fmt.Errorf(
				"endpoint is set as exclusive by the system administrator: only %q can be used",
				libURI,
			)
		}
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
	} else if ep.Exclusive {
		buildURI, err := ep.GetServiceURI(Builder)
		if err != nil {
			return nil, fmt.Errorf("unable to get builder service URI: %v", err)
		}
		if !remoteutil.SameURI(uri, buildURI) {
			return nil, fmt.Errorf(
				"endpoint is set as exclusive by the system administrator: only %q can be used",
				buildURI,
			)
		}
	}

	return config, nil
}
