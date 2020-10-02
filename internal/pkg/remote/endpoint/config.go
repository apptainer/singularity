// Copyright (c) 2020, Control Command Inc. All rights reserved.
// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package endpoint

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/sylabs/singularity/internal/pkg/remote/credential"
	"github.com/sylabs/singularity/pkg/syfs"
)

var cacheDuration = 720 * time.Hour

var DefaultEndpointConfig = &Config{
	URI:    SCSDefaultCloudURI,
	System: true,
}

// Config describes a single remote endpoint.
type Config struct {
	URI        string           `yaml:"URI,omitempty"`
	Token      string           `yaml:"Token,omitempty"`
	System     bool             `yaml:"System"`    // Was this EndPoint set from system config file
	Exclusive  bool             `yaml:"Exclusive"` // true if the endpoint must be used exclusively
	Keyservers []*ServiceConfig `yaml:"Keyservers,omitempty"`

	// for internal purpose
	credentials []*credential.Config
	services    map[string][]Service
}

func (e *Config) SetCredentials(creds []*credential.Config) {
	e.credentials = creds
}

type ServiceConfig struct {
	// for internal purpose
	credential *credential.Config

	URI      string `yaml:"URI"`
	Skip     bool   `yaml:"Skip"`
	External bool   `yaml:"External"`
	Insecure bool   `yaml:"Insecure"`
}

func cacheDir() string {
	cacheDir := syfs.RemoteCacheDir()
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		if err := os.Mkdir(cacheDir, 0700); err != nil {
			return ""
		}
	}
	return cacheDir
}

func getCachedConfig(uri string) io.ReadCloser {
	dir := cacheDir()
	if dir == "" {
		return nil
	}
	config := filepath.Join(dir, uri+".json")
	fi, err := os.Stat(config)
	if err != nil {
		return nil
	} else if fi.ModTime().Add(cacheDuration).Before(time.Now()) {
		return nil
	}
	rc, err := os.Open(config)
	if err != nil {
		return nil
	}
	return rc
}

func updateCachedConfig(uri string, data []byte) {
	dir := cacheDir()
	if dir == "" {
		return
	}
	config := filepath.Join(dir, uri+".json")
	ioutil.WriteFile(config, data, 0600)
}
