// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the LICENSE.md file
// distributed with the sources of this project regarding your rights to use or distribute this
// software.

package client

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
)

const (
	schemeHTTP  = "http"
	schemeHTTPS = "https"
	schemeHKP   = "hkp"
	schemeHKPS  = "hkps"
)

var (
	// ErrTLSRequired is returned when an auth token is supplied with a non-TLS BaseURL.
	ErrTLSRequired = errors.New("TLS required when auth token provided")
)

// Config contains the client configuration.
type Config struct {
	// Base URL of the service (https://keys.sylabs.io is used if not supplied).
	BaseURL string
	// Auth token to include in the Authorization header of each request (if supplied).
	AuthToken string
	// User agent to include in each request (if supplied).
	UserAgent string
	// HTTPClient to use to make HTTP requests (if supplied).
	HTTPClient *http.Client
}

// DefaultConfig is a configuration that uses default values.
var DefaultConfig = &Config{}

// PageDetails includes pagination details.
type PageDetails struct {
	// Maximum number of results per page (server may ignore or return fewer).
	Size int
	// Token for next page (advanced with each request, empty for last page).
	Token string
}

// Client describes the client details.
type Client struct {
	// Base URL of the service.
	BaseURL *url.URL
	// Auth token to include in the Authorization header of each request (if supplied).
	AuthToken string
	// User agent to include in each request (if supplied).
	UserAgent string
	// HTTPClient to use to make HTTP requests.
	HTTPClient *http.Client
}

// normalizeURL normalizes the scheme of the supplied URL. If an unsupported scheme is provided, an
// error is returned.
func normalizeURL(u *url.URL) (*url.URL, error) {
	switch u.Scheme {
	case schemeHTTP, schemeHTTPS:
		return u, nil
	case schemeHKP:
		// The HKP scheme is HTTP and implies port 11371.
		newURL := *u
		newURL.Scheme = schemeHTTP
		if u.Port() == "" {
			newURL.Host = net.JoinHostPort(u.Hostname(), "11371")
		}
		return &newURL, nil
	case schemeHKPS:
		// The HKPS scheme is HTTPS and implies port 443.
		newURL := *u
		newURL.Scheme = schemeHTTPS
		return &newURL, nil
	default:
		return nil, fmt.Errorf("unsupported protocol scheme %q", u.Scheme)
	}
}

const defaultBaseURL = "https://keys.sylabs.io"

// NewClient sets up a new Key Service client with the specified base URL and auth token.
func NewClient(cfg *Config) (c *Client, err error) {
	if cfg == nil {
		cfg = DefaultConfig
	}

	// Determine base URL
	bu := defaultBaseURL
	if cfg.BaseURL != "" {
		bu = cfg.BaseURL
	}
	baseURL, err := url.Parse(bu)
	if err != nil {
		return nil, err
	}
	baseURL, err = normalizeURL(baseURL)
	if err != nil {
		return nil, err
	}

	// If auth token is used, verify TLS.
	if cfg.AuthToken != "" && baseURL.Scheme != schemeHTTPS && baseURL.Hostname() != "localhost" {
		return nil, ErrTLSRequired
	}

	c = &Client{
		BaseURL:   baseURL,
		AuthToken: cfg.AuthToken,
		UserAgent: cfg.UserAgent,
	}

	// Set HTTP client
	if cfg.HTTPClient != nil {
		c.HTTPClient = cfg.HTTPClient
	} else {
		c.HTTPClient = http.DefaultClient
	}

	return c, nil
}

// newRequest returns a new Request given a method, path, query, and optional body.
func (c *Client) newRequest(method, path, rawQuery string, body io.Reader) (r *http.Request, err error) {
	u := c.BaseURL.ResolveReference(&url.URL{
		Path:     path,
		RawQuery: rawQuery,
	})
	u, err = normalizeURL(u)
	if err != nil {
		return nil, err
	}

	// If auth token is used, verify TLS.
	if c.AuthToken != "" && u.Scheme != schemeHTTPS && u.Hostname() != "localhost" {
		return nil, ErrTLSRequired
	}

	r, err = http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}
	if v := c.AuthToken; v != "" {
		r.Header.Set("Authorization", fmt.Sprintf("BEARER %s", v))
	}
	if v := c.UserAgent; v != "" {
		r.Header.Set("User-Agent", v)
	}

	return r, nil
}
