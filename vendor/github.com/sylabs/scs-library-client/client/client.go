// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the LICENSE.md file
// distributed with the sources of this project regarding your rights to use or distribute this
// software.

package client

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/go-log/log"
)

// Config contains the client configuration.
type Config struct {
	// Base URL of the service.
	BaseURL string
	// Auth token to include in the Authorization header of each request (if supplied).
	AuthToken string
	// User agent to include in each request (if supplied).
	UserAgent string
	// HTTPClient to use to make HTTP requests (if supplied).
	HTTPClient *http.Client
	// Logger to be used when output is generated
	Logger log.Logger
}

// DefaultConfig is a configuration that uses default values.
var DefaultConfig = &Config{}

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
	// Logger to be used when output is generated
	Logger log.Logger
}

const defaultBaseURL = "https://library.sylabs.io"

// NewClient sets up a new Cloud-Library Service client with the specified base URL and auth token.
func NewClient(cfg *Config) (*Client, error) {
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
	if baseURL.Scheme != "http" && baseURL.Scheme != "https" {
		return nil, fmt.Errorf("unsupported protocol scheme %q", baseURL.Scheme)
	}

	c := &Client{
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

	if cfg.Logger != nil {
		c.Logger = cfg.Logger
	} else {
		c.Logger = log.DefaultLogger
	}

	return c, nil
}

// newRequest initializes HTTP request and sets up headers based on configuration
func (c *Client) newRequest(method, path, rawQuery string, body io.Reader) (*http.Request, error) {
	u := c.BaseURL.ResolveReference(&url.URL{
		Path:     path,
		RawQuery: rawQuery,
	})
	r, err := http.NewRequest(method, u.String(), body)
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
