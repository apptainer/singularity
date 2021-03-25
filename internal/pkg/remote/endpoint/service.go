// Copyright (c) 2020, Control Command Inc. All rights reserved.
// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package endpoint

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	jsonresp "github.com/sylabs/json-resp"
	"github.com/sylabs/singularity/internal/pkg/remote/credential"
	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
)

const defaultTimeout = 10 * time.Second

const (
	SCSDefaultCloudURI     = "cloud.sylabs.io"
	SCSDefaultLibraryURI   = "https://library.sylabs.io"
	SCSDefaultKeyserverURI = "https://keys.sylabs.io"
	SCSDefaultBuilderURI   = "https://build.sylabs.io"
)

// SCS cloud services
const (
	Consent   = "consent"
	Token     = "token"
	Library   = "library"
	Keystore  = "keystore" // alias for keyserver
	Keyserver = "keyserver"
	Builder   = "builder"
)

var errorCodeMap = map[int]string{
	404: "Invalid Credentials",
	500: "Internal Server Error",
}

var (
	// ErrStatusNotSupported represents the error returned by
	// a service which doesn't support SCS status check.
	ErrStatusNotSupported = errors.New("status not supported")
)

// Service defines a simple service interface which can be exposed
// to retrieve service URI and check the service status.
type Service interface {
	URI() string
	Status() (string, error)
}

func newService(config *ServiceConfig) Service {
	return &service{cfg: config}
}

type service struct {
	cfg *ServiceConfig
}

// URI returns the service URI.
func (s *service) URI() string {
	return s.cfg.URI
}

// Status checks the service status and returns the version
// of the corresponding service. An ErrStatusNotSupported is
// returned if the service doesn't support this check.
func (s *service) Status() (version string, err error) {
	if s.cfg.External {
		return "", ErrStatusNotSupported
	}

	client := &http.Client{
		Timeout: (30 * time.Second),
	}

	req, err := http.NewRequest(http.MethodGet, s.cfg.URI+"/version", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", useragent.Value())

	res, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request to server: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error response from server: %v", res.StatusCode)
	}

	var vRes struct {
		Version string `json:"version"`
	}

	if err := jsonresp.ReadResponse(res.Body, &vRes); err != nil {
		return "", err
	}

	return vRes.Version, nil
}

func (ep *Config) GetAllServices() (map[string][]Service, error) {
	if ep.services != nil {
		return ep.services, nil
	}

	ep.services = make(map[string][]Service)

	client := &http.Client{
		Timeout: defaultTimeout,
	}

	url := "https://" + ep.URI + "/assets/config/config.prod.json"

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", useragent.Value())

	cacheReader := getCachedConfig(ep.URI)
	reader := cacheReader

	if cacheReader == nil {
		res, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("error making request to server: %s", err)
		} else if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("error response from server: %s", err)
		}
		reader = res.Body
	}
	defer reader.Close()

	b, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("while reading response body: %v", err)
	}

	var a map[string]map[string]interface{}

	if err := json.Unmarshal(b, &a); err != nil {
		return nil, fmt.Errorf("jsonresp: failed to unmarshal response: %v", err)
	}

	if reader != cacheReader {
		updateCachedConfig(ep.URI, b)
	}

	for k, v := range a {
		s := strings.TrimSuffix(k, "API")
		uri, ok := v["uri"].(string)
		if !ok {
			continue
		}

		serviceConfig := &ServiceConfig{
			URI: uri,
			credential: &credential.Config{
				URI:  uri,
				Auth: credential.TokenPrefix + ep.Token,
			},
		}

		if s == Keystore {
			s = Keyserver
		}

		ep.services[s] = []Service{
			newService(serviceConfig),
		}
	}

	return ep.services, nil
}

// GetServiceURI returns the URI for the service at the specified SCS endpoint
// Examples of services: consent, build, library, key, token
func (ep *Config) GetServiceURI(service string) (string, error) {
	// don't grab remote URI if the endpoint is the
	// default public Sylabs Cloud Service
	if ep.URI == SCSDefaultCloudURI {
		switch service {
		case Library:
			return SCSDefaultLibraryURI, nil
		case Builder:
			return SCSDefaultBuilderURI, nil
		case Keyserver:
			return SCSDefaultKeyserverURI, nil
		}
	}

	services, err := ep.GetAllServices()
	if err != nil {
		return "", err
	}

	s, ok := services[service]
	if !ok || len(s) == 0 {
		return "", fmt.Errorf("%v is not a service at endpoint", service)
	} else if s[0].URI() == "" {
		return "", fmt.Errorf("%v service at endpoint failed to provide URI in response", service)
	}

	return s[0].URI(), nil
}
