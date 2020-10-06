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
	"github.com/sylabs/singularity/internal/pkg/util/interactive"
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

// ensurePassword ensures password is not empty, if it is, a prompt
// is displayed asking user to provide a password, the entered password
// is then returned by this function. If password is not empty this
// function just return the password provided as argument.
func ensurePassword(password string) (string, error) {
	if password == "" {
		question := "Password (or token when username is empty): "
		input, err := interactive.AskQuestionNoEcho(question)
		if err != nil {
			return "", fmt.Errorf("failed to read password: %s", err)
		}
		if input == "" {
			return "", fmt.Errorf("A password is required")
		}
		return input, nil
	}
	return password, nil
}

// ociHandler handle login/logout for services with docker:// and oras:// scheme.
type ociHandler struct{}

func (h *ociHandler) login(u *url.URL, username, password string, insecure bool) (*Config, error) {
	if username == "" {
		return nil, fmt.Errorf("Docker/OCI registry requires a username")
	}
	pass, err := ensurePassword(password)
	if err != nil {
		return nil, err
	}
	cli, err := auth.NewClient(syfs.DockerConf())
	if err != nil {
		return nil, err
	}
	if err := cli.Login(context.TODO(), u.Host+u.Path, username, pass, insecure); err != nil {
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
	pass, err := ensurePassword(password)
	if err != nil {
		return nil, err
	}

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
		req.Header.Set("Authorization", TokenPrefix+pass)
	} else {
		req.SetBasicAuth(username, pass)
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
