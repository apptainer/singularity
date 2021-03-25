// Copyright (c) 2020, Control Command Inc. All rights reserved.
// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package remote

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/remote/credential"
	"github.com/sylabs/singularity/internal/pkg/remote/endpoint"
	remoteutil "github.com/sylabs/singularity/internal/pkg/remote/util"
	"github.com/sylabs/singularity/pkg/syfs"
	"github.com/sylabs/singularity/pkg/sylog"
	yaml "gopkg.in/yaml.v2"
)

var (
	// ErrNoDefault indicates no default remote being set
	ErrNoDefault = errors.New("no default remote")
)

const (
	// DefaultRemoteName is the default remote name
	DefaultRemoteName = "SylabsCloud"
)

// DefaultRemoteConfig holds the default remote configuration
// if there is no remote.yaml present both in user home directory
// and in system location.
var DefaultRemoteConfig = &Config{
	DefaultRemote: DefaultRemoteName,
	Remotes: map[string]*endpoint.Config{
		DefaultRemoteName: endpoint.DefaultEndpointConfig,
	},
}

// SystemConfigPath holds the path to the remote system configuration.
var SystemConfigPath = filepath.Join(buildcfg.SYSCONFDIR, "singularity", syfs.RemoteConfFile)

// Config stores the state of remote endpoint configurations
type Config struct {
	DefaultRemote string                      `yaml:"Active"`
	Remotes       map[string]*endpoint.Config `yaml:"Remotes"`
	Credentials   []*credential.Config        `yaml:"Credentials,omitempty"`

	// set to true when this is the system configuration
	system bool
}

// ReadFrom reads remote configuration from io.Reader
// returns Config populated with remotes
func ReadFrom(r io.Reader) (*Config, error) {
	c := &Config{
		Remotes: make(map[string]*endpoint.Config),
	}

	// check if the reader point to the remote system configuration
	if f, ok := r.(*os.File); ok {
		c.system = f.Name() == SystemConfigPath
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
			eUsr.Exclusive = eSys.Exclusive
			if eSys.Exclusive {
				c.DefaultRemote = name
			}
			eUsr.Keyservers = eSys.Keyservers
			continue
		}

		if eSys.Exclusive {
			c.DefaultRemote = name
		}
		e := &endpoint.Config{
			URI:        eSys.URI,
			System:     true,
			Exclusive:  eSys.Exclusive,
			Keyservers: eSys.Keyservers,
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

// SetDefault sets default remote endpoint or returns an error if it does not exist.
// A remote endpoint can also be set as exclusive.
func (c *Config) SetDefault(name string, exclusive bool) error {
	r, ok := c.Remotes[name]
	if !ok {
		return fmt.Errorf("%s is not a remote", name)
	}
	if !c.system && exclusive {
		return fmt.Errorf("exclusive can't be set by user")
	} else if name != c.DefaultRemote {
		for n, r := range c.Remotes {
			if r.Exclusive && !c.system {
				return fmt.Errorf(
					"could not use %s: remote %s has been set exclusive by the system administrator",
					name, n,
				)
			}
		}
	}

	dr, ok := c.Remotes[c.DefaultRemote]
	if ok && c.DefaultRemote != name && exclusive {
		dr.Exclusive = false
	}
	r.Exclusive = exclusive

	c.DefaultRemote = name
	return nil
}

// GetDefault returns default remote endpoint or an error
func (c *Config) GetDefault() (*endpoint.Config, error) {
	if c.DefaultRemote == "" {
		return nil, ErrNoDefault
	}
	return c.GetRemote(c.DefaultRemote)
}

// Add a new remote endpoint
// returns an error if it already exists
func (c *Config) Add(name string, e *endpoint.Config) error {
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
	if r, ok := c.Remotes[name]; !ok {
		return fmt.Errorf("%s is not a remote", name)
	} else if r.System && !c.system {
		return fmt.Errorf("%s is global and can't be removed", name)
	}

	if c.DefaultRemote == name {
		c.DefaultRemote = ""
	}

	delete(c.Remotes, name)
	return nil
}

// GetRemote returns a reference to an existing endpoint
// returns error if remote does not exist
func (c *Config) GetRemote(name string) (*endpoint.Config, error) {
	r, ok := c.Remotes[name]
	if !ok {
		return nil, fmt.Errorf("%s is not a remote", name)
	}
	r.SetCredentials(c.Credentials)
	return r, nil
}

// Login validates and stores credentials for a service like Docker/OCI registries
// and keyservers.
func (c *Config) Login(uri, username, password string, insecure bool) error {
	_, err := remoteutil.NormalizeKeyserverURI(uri)
	// if there is no error, we consider it as a keyserver
	if err == nil {
		var keyserverConfig *endpoint.ServiceConfig

		for _, ep := range c.Remotes {
			if keyserverConfig != nil {
				break
			}
			for _, kc := range ep.Keyservers {
				if !kc.External {
					continue
				}
				if remoteutil.SameKeyserver(uri, kc.URI) {
					keyserverConfig = kc
					break
				}
			}
		}
		if keyserverConfig == nil {
			return fmt.Errorf("no external keyserver configuration found for %s", uri)
		} else if keyserverConfig.Insecure && !insecure {
			sylog.Warningf("%s is configured as insecure, forcing insecure flag for login", uri)
			insecure = true
		} else if !keyserverConfig.Insecure && insecure {
			insecure = false
		}
	}

	credConfig, err := credential.Manager.Login(uri, username, password, insecure)
	if err != nil {
		return err
	}
	c.Credentials = append(c.Credentials, credConfig)
	return nil
}

// Logout removes previously stored credentials for a service.
func (c *Config) Logout(uri string) error {
	if err := credential.Manager.Logout(uri); err != nil {
		return err
	}
	for i, cred := range c.Credentials {
		if remoteutil.SameURI(cred.URI, uri) {
			c.Credentials = append(c.Credentials[:i], c.Credentials[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("%s is not configured", uri)
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
