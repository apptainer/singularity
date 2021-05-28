// Copyright (c) 2020, Control Command Inc. All rights reserved.
// Copyright (c) 2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	registry "github.com/adigunhammedolalekan/registry-auth"
)

const (
	// DefaultUsername is the default E2E username.
	DefaultUsername = "e2e"
	// DefaultPassword is the default E2E password.
	DefaultPassword  = "e2e"
	privateNamespace = "private"
)

type dockerAuthHandler struct {
	srv *registry.AuthServer
}

type authnz struct {
	username string
	password string
	sync.Mutex
}

const (
	noAuthUsername = "no-auth"
)

func (a *authnz) Authenticate(username, password string) error {
	// deferring unlock in Authorize to be sure username
	// and password are associated to the same request,
	// fortunately docker-registry-auth package doesn't
	// generate error between Authenticate and Authorize
	// calls, so a deadlock is not possible
	a.Lock()

	if username == noAuthUsername {
		a.username = ""
		a.password = ""
	} else {
		a.username = username
		a.password = password
	}

	return nil
}

func (a *authnz) Authorize(req *registry.AuthorizationRequest) ([]string, error) {
	// release previous lock
	defer a.Unlock()

	requireAuth := false

	if strings.HasPrefix(req.Name, privateNamespace) || req.Type == "" {
		requireAuth = true
	}
	if requireAuth {
		if a.username != DefaultUsername || a.password != DefaultPassword {
			return nil, fmt.Errorf("unauthorized")
		}
	}

	return []string{"pull", "push"}, nil
}

func startAuthServer(crt, key string) error {
	authnz := new(authnz)

	opt := &registry.Option{
		Certfile:        crt,
		Keyfile:         key,
		TokenExpiration: time.Now().Add(1 * time.Hour).Unix(),
		TokenIssuer:     "E2E",
		Authenticator:   authnz,
		Authorizer:      authnz,
	}

	srv, err := registry.NewAuthServer(opt)
	if err != nil {
		return err
	}

	http.Handle("/auth", &dockerAuthHandler{srv: srv})
	return http.ListenAndServe(":5001", nil)
}

func (d *dockerAuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, _, ok := r.BasicAuth()
	if !ok {
		// pass a non empty username meaning there is no authentication
		// credentials, this is required as the docker-registry-auth package
		// doesn't allow empty credentials, Authorize will reset the username
		// to an empty value
		r.SetBasicAuth(noAuthUsername, "")
	}
	d.srv.ServeHTTP(w, r)
}
