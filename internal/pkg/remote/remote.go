// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package remote

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
	yaml "gopkg.in/yaml.v2"
)

var (
	// ErrNoDefault indicates no default remote being set
	ErrNoDefault = errors.New("no default remote")
)

var errorCodeMap = map[int]string{
	404: "Invalid Token",
	500: "Internal Server Error",
}

// Config stores the state of remote endpoint configurations
type Config struct {
	DefaultRemote string               `yaml:"Active"`
	Remotes       map[string]*EndPoint `yaml:"Remotes"`
}

// EndPoint descriptes a single remote service
type EndPoint struct {
	URI    string `yaml:"URI,omitempty"`
	Token  string `yaml:"Token,omitempty"`
	System bool   `yaml:"System"` // Was this EndPoint set from system config file
}

// ReadFrom reads remote configuration from io.Reader
// returns Config populated with remotes
func ReadFrom(r io.Reader) (*Config, error) {
	c := &Config{
		Remotes: make(map[string]*EndPoint),
	}

	// read all data from r into b
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read from io.Reader: %s", err)
	}

	if len(b) > 0 {
		// If we had data to read in io.Reader, attempt to unmarshal as YAML.
		// Also, it will fail if the YAML file does not have the expected
		// structure.
		if err := yaml.UnmarshalStrict(b, c); err != nil {
			return nil, fmt.Errorf("failed to decode YAML data from io.Reader: %s", err)
		}
	}
	return c, nil
}

// WriteTo writes the configuration to the io.Writer
// returns and error if write is incomplete
func (c *Config) WriteTo(w io.Writer) (int64, error) {
	yaml, err := yaml.Marshal(c)
	if err != nil {
		return 0, fmt.Errorf("failed to marshall remote config to yaml: %v", err)
	}

	n, err := w.Write(yaml)
	if err != nil {
		return 0, fmt.Errorf("failed to write remote config to io.Writer: %v", err)
	}

	return int64(n), err
}

// SyncFrom updates c with the remotes specified in sys. Typically, this is used
// to sync a globally-configured remote.Config into a user-specific remote.Config.
// Currently, SyncFrom will return a name-collision error if there is an EndPoint
// name which exists in both c & sys, and the EndPoint in c has System == false.
func (c *Config) SyncFrom(sys *Config) error {
	for name, eSys := range sys.Remotes {
		eUsr, err := c.GetRemote(name)
		if err == nil && !eUsr.System { // usr & sys name collision
			return fmt.Errorf("name collision while syncing: %s", name)
		} else if err == nil {
			eUsr.URI = eSys.URI // update URI just in case
			continue
		}

		e := &EndPoint{
			URI:    eSys.URI,
			System: true,
		}

		if err := c.Add(name, e); err != nil {
			return err
		}
	}

	// set system default to user default if no user default specified
	if c.DefaultRemote == "" && sys.DefaultRemote != "" {
		c.DefaultRemote = sys.DefaultRemote
	}

	return nil
}

// SetDefault sets default remote endpoint or returns an error if it does not exist
func (c *Config) SetDefault(name string) error {
	if _, ok := c.Remotes[name]; !ok {
		return fmt.Errorf("%s is not a remote", name)
	}

	c.DefaultRemote = name
	return nil
}

// GetDefault returns default remote endpoint or an error
func (c *Config) GetDefault() (*EndPoint, error) {
	if c.DefaultRemote == "" {
		return nil, ErrNoDefault
	}

	if _, ok := c.Remotes[c.DefaultRemote]; !ok {
		return nil, fmt.Errorf("%s is not a remote", c.DefaultRemote)
	}

	return c.Remotes[c.DefaultRemote], nil
}

// Add a new remote endpoint
// returns an error if it already exists
func (c *Config) Add(name string, e *EndPoint) error {
	if _, ok := c.Remotes[name]; ok {
		return fmt.Errorf("%s is already a remote", name)
	}

	c.Remotes[name] = e
	return nil
}

// Remove a remote endpoint
// if endpoint is the default, the default is cleared
// returns an error if it does not exist
func (c *Config) Remove(name string) error {
	if _, ok := c.Remotes[name]; !ok {
		return fmt.Errorf("%s is not a remote", name)
	}

	if c.DefaultRemote == name {
		c.DefaultRemote = ""
	}

	delete(c.Remotes, name)
	return nil
}

// GetRemote returns a reference to an existing endpoint
// returns error if remote does not exist
func (c *Config) GetRemote(name string) (*EndPoint, error) {
	r, ok := c.Remotes[name]
	if !ok {
		return nil, fmt.Errorf("%s is not a remote", name)
	}
	return r, nil
}

// Rename an existing remote
// returns an error if it does not exist
func (c *Config) Rename(name, newName string) error {
	if _, ok := c.Remotes[name]; !ok {
		return fmt.Errorf("%s is not a remote", name)
	}

	if _, ok := c.Remotes[newName]; ok {
		return fmt.Errorf("%s is already a remote", newName)
	}

	if c.DefaultRemote == name {
		c.DefaultRemote = newName
	}

	c.Remotes[newName] = c.Remotes[name]
	delete(c.Remotes, name)
	return nil
}

// VerifyToken returns an error if a token is not valid
func (e *EndPoint) VerifyToken() error {
	baseURL, err := e.GetServiceURI("token")
	if err != nil {
		return fmt.Errorf("while getting token service uri: %v", err)
	}

	client := &http.Client{
		Timeout: (30 * time.Second),
	}
	req, err := http.NewRequest(http.MethodGet, baseURL+"/v1/token-status", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", e.Token))
	req.Header.Set("User-Agent", useragent.Value())

	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error making request to server:\n\t%v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		convStatus, ok := errorCodeMap[res.StatusCode]
		if !ok {
			convStatus = "Unknown"
		}
		return fmt.Errorf("error response from server: %v", convStatus)
	}

	return nil
}

func getCloudConfig(uri string) ([]byte, error) {
	client := &http.Client{
		Timeout: (30 * time.Second),
	}

	url := "https://" + uri + "/assets/config/config.prod.json"

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", useragent.Value())

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request to server:\n\t%v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error response from server: %v", res.StatusCode)
	}

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("while reading response body: %v", err)
	}
	return b, nil
}

// GetServiceURI returns the URI for the service at the specified SCS endpoint
// Examples of services: consent, build, library, key, token
func (e *EndPoint) GetServiceURI(service string) (string, error) {
	b, err := getCloudConfig(e.URI)
	if err != nil {
		return "", err
	}

	var a map[string]map[string]interface{}
	if err := json.Unmarshal(b, &a); err != nil {
		return "", fmt.Errorf("jsonresp: failed to unmarshal response: %v", err)
	}

	val, ok := a[service+"API"]
	if !ok {
		return "", fmt.Errorf("%v is not a service at endpoint", service)
	}

	uri, ok := val["uri"].(string)
	if !ok {
		return "", fmt.Errorf("%v service at endpoint failed to provide URI in response", service)
	}

	return uri, nil
}

// GetAllServiceURIs returns all available service urls for a given endpoint in a map
func (e *EndPoint) GetAllServiceURIs() (map[string]string, error) {
	b, err := getCloudConfig(e.URI)
	if err != nil {
		return nil, err
	}

	var a map[string]map[string]interface{}
	if err := json.Unmarshal(b, &a); err != nil {
		return nil, fmt.Errorf("jsonresp: failed to unmarshal response: %v", err)
	}

	uris := make(map[string]string)
	for k := range a {
		if strings.HasSuffix(k, "API") {
			if s, ok := a[k]["uri"].(string); ok {
				uris[strings.TrimSuffix(k, "API")] = s
			}
		}
	}

	return uris, nil
}
