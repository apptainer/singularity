// Copyright (c) 2020, Control Command Inc. All rights reserved.
// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package util

import (
	"fmt"
	"net"
	"net/url"
)

const (
	hkpPort = "11371"
)

const (
	httpScheme  = "http"
	httpsScheme = "https"
	hkpScheme   = "hkp"
	hkpsScheme  = "hkps"
)

// NormalizeKeyserverURI is normalizing a URI string by converting
// protocol scheme hkp:// and hkps:// to their corresponding http://
// and https:// protocol scheme and returns the parsed URI with the
// corresponding scheme.
func NormalizeKeyserverURI(uri string) (*url.URL, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case httpScheme, httpsScheme:
	case hkpScheme:
		u.Scheme = httpScheme
		if u.Port() == "" {
			u.Host = net.JoinHostPort(u.Hostname(), hkpPort)
		}
	case hkpsScheme:
		u.Scheme = httpsScheme
	default:
		return nil, fmt.Errorf("unsupported keyserver protocol scheme %q", u.Scheme)
	}

	return u, nil
}

// SameKeyserver returns if two URIs point to the same keyserver or not, URI are
// also normalized meaning by example that hkp://localhost is equivalent to
// http://localhost:11371.
func SameKeyserver(u1, u2 string) bool {
	uri1, err := NormalizeKeyserverURI(u1)
	if err != nil {
		return false
	}
	uri2, err := NormalizeKeyserverURI(u2)
	if err != nil {
		return false
	}
	return Equal(uri1, uri2)
}

// SameURI returns if two URIs point to the same service or not.
func SameURI(u1, u2 string) bool {
	uri1, err := url.Parse(u1)
	if err != nil {
		return false
	}
	uri2, err := url.Parse(u2)
	if err != nil {
		return false
	}
	return Equal(uri1, uri2)
}

// Equal returns if both URLs have the same hostname, port and scheme or not.
func Equal(u1, u2 *url.URL) bool {
	if u1.Host == "" || u2.Host == "" {
		return false
	}
	return u1.Host == u2.Host && u1.Scheme == u2.Scheme
}
