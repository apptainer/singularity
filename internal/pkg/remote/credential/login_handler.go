// Copyright (c) 2020, Control Command Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package credential

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"time"

	auth "github.com/deislabs/oras/pkg/auth/docker"
	"github.com/sylabs/singularity/pkg/syfs"
	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
)

// loginHandlers contains the registered handlers by scheme.
var loginHandlers = make(map[string]loginHandler)

// loginHandler interface implements login and logout for a specific scheme.
type loginHandler interface {
	login(url *url.URL, username, password string, insecure bool) (*Config, error)
	logout(url *url.URL) error
}

func init() {
	oh := &ociHandler{}
	loginHandlers["oras"] = oh
	loginHandlers["docker"] = oh

	kh := &keyserverHandler{}
	loginHandlers["http"] = kh
	loginHandlers["https"] = kh
}

// ociHandler handle login/logout for services with docker:// and oras:// scheme.
type ociHandler struct{}

func (h *ociHandler) login(u *url.URL, username, password string, insecure bool) (*Config, error) {
	cli, err := auth.NewClient(syfs.DockerConf())
	if err != nil {
		return nil, err
	}
	if err := cli.Login(context.TODO(), u.Host+u.Path, username, password, insecure); err != nil {
		return nil, err
	}
	return &Config{
		URI:      u.String(),
		Insecure: insecure,
	}, nil
}

func (h *ociHandler) logout(u *url.URL) error {
	cli, err := auth.NewClient(syfs.DockerConf())
	if err != nil {
		return err
	}
	return cli.Logout(context.TODO(), u.Host+u.Path)
}

// keyserverHandler handle login/logout for keyserver service.
type keyserverHandler struct{}

func (h *keyserverHandler) login(u *url.URL, username, password string, insecure bool) (*Config, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	if insecure {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	if username == "" {
		req.Header.Set("Authorization", TokenPrefix+password)
	} else {
		req.SetBasicAuth(username, password)
	}

	auth := req.Header.Get("Authorization")
	req.Header.Set("User-Agent", useragent.Value())

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request to server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error response from server: %s", resp.Status)
	}

	return &Config{
		URI:      u.String(),
		Auth:     auth,
		Insecure: insecure,
	}, nil
}

func (h *keyserverHandler) logout(u *url.URL) error {
	return nil
}
