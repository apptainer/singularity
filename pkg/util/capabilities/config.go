// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package capabilities

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/sylabs/singularity/internal/pkg/sylog"
)

// Caplist defines a map of users/groups with associated list of capabilities
type Caplist map[string][]string

// Config is the in memory representation of the user/group capability
// authorizations as set by an admin
type Config struct {
	Users  Caplist `json:"users,omitempty"`
	Groups Caplist `json:"groups,omitempty"`
}

// ReadFrom reads a capability configuration from an io.Reader and returns a capability
// config with the set of authorized user/group capabilities
func ReadFrom(r io.Reader) (*Config, error) {
	c := &Config{
		Users:  make(Caplist),
		Groups: make(Caplist),
	}

	// read all data from r into b
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read from io.Reader: %s", err)
	}

	if len(b) > 0 {
		// if we had data to read in io.Reader, attempt to unmarshal as JSON
		if err := json.Unmarshal(b, c); err != nil {
			return nil, fmt.Errorf("failed to decode JSON data from io.Reader: %s", err)
		}
	} else {
		// if no data in io.Reader, populate c with empty data
		data, err := json.Marshal(c)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize data")
		}
		json.Unmarshal(data, c)
	}

	return c, nil
}

// WriteTo writes the capability config into the provided io.Writer. If writing to the
// same file as passed to ReadFrom(io.Reader), the file should be truncated should seek to 0
// before passing the file as the io.Writer
func (c *Config) WriteTo(w io.Writer) (int64, error) {
	json, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		return 0, fmt.Errorf("failed to marshall capability data to json: %v", err)
	}

	n, err := w.Write(json)
	if err != nil {
		return int64(n), fmt.Errorf("failed to write capability json to io.Writer: %v", err)
	}

	return int64(n), err
}

func (c *Config) checkCaps(caps []string) error {
	for _, c := range caps {
		if _, ok := Map[c]; !ok {
			return fmt.Errorf("unknown capability %s", c)
		}
	}
	return nil
}

// AddUserCaps adds an authorized capability set to user
func (c *Config) AddUserCaps(user string, caps []string) error {
	if err := c.checkCaps(caps); err != nil {
		return err
	}
	for _, cap := range caps {
		present := false
		for _, c := range c.Users[user] {
			if c == cap {
				present = true
			}
		}
		if !present {
			c.Users[user] = append(c.Users[user], cap)
		} else {
			sylog.Warningf("Won't add capability '%s', already assigned to user %s", cap, user)
		}
	}
	return nil
}

// AddGroupCaps adds an authorized capability set to group
func (c *Config) AddGroupCaps(group string, caps []string) error {
	if err := c.checkCaps(caps); err != nil {
		return err
	}
	for _, cap := range caps {
		present := false
		for _, c := range c.Groups[group] {
			if c == cap {
				present = true
			}
		}
		if !present {
			c.Groups[group] = append(c.Groups[group], cap)
		} else {
			sylog.Warningf("Won't add capability '%s', already assigned to group %s", cap, group)
		}
	}
	return nil
}

// DropUserCaps drops a set of capabilities for user
func (c *Config) DropUserCaps(user string, caps []string) error {
	if err := c.checkCaps(caps); err != nil {
		return err
	}
	if _, ok := c.Users[user]; !ok {
		return fmt.Errorf("user '%s' doesn't have any capability assigned", user)
	}
	for _, cap := range caps {
		dropped := false
		for i := len(c.Users[user]) - 1; i >= 0; i-- {
			if c.Users[user][i] == cap {
				c.Users[user] = append(c.Users[user][:i], c.Users[user][i+1:]...)
				dropped = true
				break
			}
		}
		if !dropped {
			sylog.Warningf("Won't drop capability '%s', not assigned to user %s", cap, user)
		}
	}
	if len(c.Users[user]) == 0 {
		delete(c.Users, user)
	}
	return nil
}

// DropGroupCaps drops a set of capabilities for group
func (c *Config) DropGroupCaps(group string, caps []string) error {
	if err := c.checkCaps(caps); err != nil {
		return err
	}
	if _, ok := c.Groups[group]; !ok {
		return fmt.Errorf("group '%s' doesn't have any capability assigned", group)
	}
	for _, cap := range caps {
		dropped := false
		for i := len(c.Groups[group]) - 1; i >= 0; i-- {
			if c.Groups[group][i] == cap {
				c.Groups[group] = append(c.Groups[group][:i], c.Groups[group][i+1:]...)
				dropped = true
				break
			}
		}
		if !dropped {
			sylog.Warningf("Won't drop capability '%s', not assigned to group %s", cap, group)
		}
	}
	if len(c.Groups[group]) == 0 {
		delete(c.Groups, group)
	}
	return nil
}

// ListUserCaps returns a capability list authorized for user
func (c *Config) ListUserCaps(user string) []string {
	return c.Users[user]
}

// ListGroupCaps returns a capability list authorized for group
func (c *Config) ListGroupCaps(group string) []string {
	return c.Groups[group]
}

// ListAllCaps returns capability list for both authorized users and groups
func (c *Config) ListAllCaps() (Caplist, Caplist) {
	return c.Users, c.Groups
}

// CheckUserCaps checks if provided capability list for user are whether
// or not authorized by returning two lists, the first one containing
// authorized capabilities and the second one containing unauthorized
// capabilities
func (c *Config) CheckUserCaps(user string, caps []string) (authorized []string, unauthorized []string) {
	for _, ca := range caps {
		present := false
		for _, userCap := range c.ListUserCaps(user) {
			if userCap == ca {
				authorized = append(authorized, ca)
				present = true
				break
			}
		}
		if !present {
			unauthorized = append(unauthorized, ca)
		}
	}
	return authorized, unauthorized
}

// CheckGroupCaps checks if provided capability list for group are whether
// or not authorized by returning two lists, the first one containing
// authorized capabilities and the second one containing unauthorized
// capabilities
func (c *Config) CheckGroupCaps(group string, caps []string) (authorized []string, unauthorized []string) {
	for _, ca := range caps {
		present := false
		for _, groupCap := range c.ListGroupCaps(group) {
			if groupCap == ca {
				authorized = append(authorized, ca)
				present = true
				break
			}
		}
		if !present {
			unauthorized = append(unauthorized, ca)
		}
	}
	return authorized, unauthorized
}
