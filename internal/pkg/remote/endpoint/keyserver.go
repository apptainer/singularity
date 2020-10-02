// Copyright (c) 2020, Control Command Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package endpoint

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/sylabs/singularity/internal/pkg/remote/credential"
	remoteutil "github.com/sylabs/singularity/internal/pkg/remote/util"
	"github.com/sylabs/singularity/pkg/sylog"
)

// KeyserverOp represents a keyserver operation type.
type KeyserverOp uint8

const (
	// KeyserverPushOp represents a key push operation.
	KeyserverPushOp KeyserverOp = iota
	// KeyserverPullOp represents a key pull operation.
	KeyserverPullOp
	// KeyserverSearch represents a key search operation.
	KeyserverSearchOp
	// KeyserverVerify represents a key verification operation.
	KeyserverVerifyOp
)

// AddKeyserver adds a keyserver for the corresponding remote endpoint.
func (ep *Config) AddKeyserver(uri string, order uint32, insecure bool) error {
	if err := ep.UpdateKeyserversConfig(); err != nil {
		return err
	}

	matchIndex := -1
	maxOrder := uint32(1)

	for i, kc := range ep.Keyservers {
		if remoteutil.SameKeyserver(kc.URI, uri) {
			matchIndex = i
		}
		if kc.Skip {
			continue
		}
		maxOrder++
	}

	if order == 0 {
		order = maxOrder
	} else if order > maxOrder {
		return fmt.Errorf("order is out of range: maximum is %d", maxOrder)
	}

	var kc *ServiceConfig

	if matchIndex >= 0 {
		kc = ep.Keyservers[matchIndex]
		if !kc.External && kc.Skip {
			kc.Skip = false
		} else {
			return fmt.Errorf("%s is already configured", uri)
		}
		// remove it first
		ep.Keyservers = append(ep.Keyservers[:matchIndex], ep.Keyservers[matchIndex+1:]...)
	} else {
		kc = &ServiceConfig{
			External: true,
			URI:      uri,
			Insecure: insecure,
		}
	}

	// insert it as specified by the order
	ep.Keyservers = append(ep.Keyservers[:order-1], append([]*ServiceConfig{kc}, ep.Keyservers[order-1:]...)...)

	return nil
}

// RemoveKeyserver removes a previously added keyserver.
func (ep *Config) RemoveKeyserver(uri string) error {
	if err := ep.UpdateKeyserversConfig(); err != nil {
		return err
	}

	total := 0
	for _, kc := range ep.Keyservers {
		if kc.Skip {
			continue
		}
		total++
	}

	for i, kc := range ep.Keyservers {
		if remoteutil.SameKeyserver(kc.URI, uri) && !kc.Skip {
			if total == 1 {
				return fmt.Errorf("the primary keyserver %s can't be removed", uri)
			}
			if kc.External {
				ep.Keyservers = append(ep.Keyservers[:i], ep.Keyservers[i+1:]...)
			} else {
				// SCS keyserver is just marked as skipped
				kc.Skip = true
			}
			return nil
		}
	}

	return fmt.Errorf("keyserver %s is not configured", uri)
}

// UpdateKeyserversConfig updates the keyserver configuration for the
// corresponding remote endpoint.
func (ep *Config) UpdateKeyserversConfig() error {
	if len(ep.Keyservers) == 0 {
		// current remote keyserver
		uri, err := ep.GetServiceURI(Keyserver)
		if err != nil {
			return err
		}
		ep.Keyservers = append(ep.Keyservers, &ServiceConfig{
			URI: uri,
			credential: &credential.Config{
				URI:  uri,
				Auth: credential.TokenPrefix + ep.Token,
			},
		})
		return nil
	}
	for _, kc := range ep.Keyservers {
		if kc.credential != nil {
			continue
		} else if !kc.External {
			// associated current endpoint token to the SCS key service
			kc.credential = &credential.Config{
				URI:  kc.URI,
				Auth: credential.TokenPrefix + ep.Token,
			}
		} else {
			// attempt to find credentials in the credential store
			for _, cred := range ep.credentials {
				if remoteutil.SameKeyserver(cred.URI, kc.URI) {
					kc.credential = cred
					break
				}
			}
		}
	}
	return nil
}

type keyserverTransport struct {
	keyservers []*ServiceConfig
	op         KeyserverOp
	client     *http.Client
}

func (c *keyserverTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	for i, k := range c.keyservers {
		if k.Skip {
			continue
		}

		cloneReq := req.Clone(ctx)

		if i > 0 {
			u, err := remoteutil.NormalizeKeyserverURI(k.URI)
			if err != nil {
				return nil, err
			}
			cloneReq.URL.Scheme = u.Scheme
			cloneReq.URL.Host = u.Host
			cloneReq.URL.User = u.User
		}

		sylog.Debugf("Querying keyserver %s", cloneReq.URL)

		cloneReq.Header.Del("Authorization")
		if k.credential != nil && k.credential.Auth != "" {
			cloneReq.Header.Set("Authorization", k.credential.Auth)
		}

		tr, ok := c.client.Transport.(*http.Transport)
		if ok {
			tr.TLSClientConfig.InsecureSkipVerify = k.Insecure
		}

		resp, err := c.client.Do(cloneReq)
		if err != nil {
			if i < len(c.keyservers)-1 {
				continue
			}
			return resp, err
		}

		if resp.StatusCode/100 != 2 && i < len(c.keyservers)-1 {
			resp.Body.Close()
			continue
		}

		return resp, err
	}

	return nil, fmt.Errorf("no keyserver configured")
}

var defaultClient = &http.Client{
	Timeout: 5 * time.Second,
	Transport: &http.Transport{
		DisableKeepAlives: true,
		TLSClientConfig:   &tls.Config{},
	},
}

func newClient(keyservers []*ServiceConfig, op KeyserverOp) *http.Client {
	return &http.Client{
		Transport: &keyserverTransport{
			keyservers: keyservers,
			op:         op,
			client:     defaultClient,
		},
	}
}
