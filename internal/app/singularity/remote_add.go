// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/sylabs/singularity/internal/pkg/remote"
)

func isValidString(str string) bool {
	if str == "" {
		return false
	}
	return true
}

func isValidName(name string) bool {
	return isValidString(name)
}

func isValidURI(uri string) bool {
	return isValidString(uri)
}

// RemoteAdd adds remote to configuration
func RemoteAdd(configFile, name, uri string, global bool) (err error) {
	// Clean up the name and uri string
	name = strings.TrimSpace(name)
	uri = strings.TrimSpace(uri)

	// Explicit handling of corner cases: name and uri must be valid strings
	if isValidName(name) == false {
		return fmt.Errorf("invalid name: %s", name)
	}
	if isValidURI(uri) == false {
		return fmt.Errorf("invalid URI: %s", uri)
	}

	c := &remote.Config{}

	// system config should be world readable
	perm := os.FileMode(0600)
	if global {
		perm = os.FileMode(0644)
	}

	// opening config file
	file, err := os.OpenFile(configFile, os.O_RDWR|os.O_CREATE, perm)
	if err != nil {
		return fmt.Errorf("while opening remote config file: %s", err)
	}
	defer file.Close()

	// read file contents to config struct
	c, err = remote.ReadFrom(file)
	if err != nil {
		return fmt.Errorf("while parsing remote config data: %s", err)
	}

	u, err := url.Parse(uri)
	if err != nil {
		return err
	}
	e := remote.EndPoint{URI: path.Join(u.Host + u.Path), System: global}

	if err := c.Add(name, &e); err != nil {
		return err
	}

	// truncating file before writing new contents and syncing to commit file
	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("while truncating remote config file: %s", err)
	}

	if n, err := file.Seek(0, os.SEEK_SET); err != nil || n != 0 {
		return fmt.Errorf("failed to reset %s cursor: %s", file.Name(), err)
	}

	if _, err := c.WriteTo(file); err != nil {
		return fmt.Errorf("while writing remote config to file: %s", err)
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to flush remote config file %s: %s", file.Name(), err)
	}

	return nil
}
