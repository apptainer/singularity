// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package fakeroot

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity/internal/pkg/util/user"
	"github.com/sylabs/singularity/pkg/util/fs/lock"
)

const (
	// SubUIDFile is the default path to the subuid file.
	SubUIDFile = "/etc/subuid"
	// SubGIDFile is the default path to the subgid file.
	SubGIDFile = "/etc/subgid"
	// validRangeCount is the valid fakeroot range count.
	validRangeCount = uint32(65536)
	// StartMax is the maximum possible range start.
	startMax = uint32(4294967296 - 131072)
	// StartMin is the minimum possible range start.
	startMin = uint32(65536)
	// disabledPrefix is the character prefix marking an entry as disabled.
	disabledPrefix = '!'
	// fieldSeparator is the character separating entry's fields.
	fieldSeparator = ":"
	// minFields is the minimum number of fields for a valid entry.
	minFields = 3
	// maxUID is the highest UID.
	maxUID = ^uint32(0)
)

// Entry represents an entry line of subuid/subgid configuration file.
type Entry struct {
	line     string
	UID      uint32
	Start    uint32
	Count    uint32
	disabled bool
	invalid  bool
}

// Config holds all entries found in the corresponding configuration
// file and manages its configuration.
type Config struct {
	entries       []*Entry
	file          *os.File
	readOnly      bool
	requireUpdate bool
	getUserFn     func(string) (*user.User, error)
}

// GetUserFn defines the user lookup function prototype.
type GetUserFn func(string) (*user.User, error)

// GetConfig parses a subuid/subgid configuration file and returns
// a Config holding all mapping entries, it allows to pass a custom
// function getUserFn used to lookup in a custom user database, if
// there is no custom function, the default one is used.
func GetConfig(filename string, edit bool, getUserFn GetUserFn) (*Config, error) {
	var err error

	config := &Config{
		readOnly:  !edit,
		getUserFn: user.GetPwNam,
	}

	// mainly for mocking
	if getUserFn != nil {
		config.getUserFn = getUserFn
	}

	flags := os.O_RDONLY
	if !config.readOnly {
		flags = os.O_CREATE | os.O_RDWR
		umask := syscall.Umask(0)
		defer syscall.Umask(umask)
	}

	config.file, err = os.OpenFile(filename, flags, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open: %s: %s", filename, err)
	}

	config.entries = make([]*Entry, 0)

	scanner := bufio.NewScanner(config.file)
	for scanner.Scan() {
		config.parseEntry(scanner.Text())
	}

	return config, nil
}

// parseEntry parses a line and adds an entry.
func (c *Config) parseEntry(line string) {
	e := new(Entry)
	e.line = line

	fields := strings.Split(line, fieldSeparator)
	// entry doesn't have the right number of fields,
	// don't add it to the list of entries that need to be removed
	// from the file during the close operation
	if len(fields) < minFields {
		return
	}

	defer func() {
		c.entries = append(c.entries, e)
	}()

	start, err := strconv.ParseUint(fields[1], 10, 32)
	if err != nil {
		e.invalid = true
	} else {
		e.Start = uint32(start)
	}

	count, err := strconv.ParseUint(fields[2], 10, 32)
	if err != nil || count == 0 {
		e.invalid = true
	} else {
		e.Count = uint32(count)
	}

	username := fields[0]

	// include disabled users
	if username[0] == disabledPrefix {
		username = username[1:]
		e.disabled = true
	}

	uid, err := strconv.Atoi(username)
	if err == nil {
		e.UID = uint32(uid)
	} else {
		// try with username, if there is an error
		// we still consider the entry as valid and
		// just associate it with the maximal UID
		u, err := c.getUserFn(username)
		if err != nil {
			e.UID = maxUID
		} else {
			e.UID = u.UID
		}
	}
}

// Close closes the configuration file handle, if there is any pending
// updates and the configuration was opened for writing, all entries
// are written before into the configuration file before closing it.
func (c *Config) Close() error {
	defer c.file.Close()

	if !c.requireUpdate || c.readOnly {
		return nil
	}

	var buf bytes.Buffer
	filename := c.file.Name()

	for _, entry := range c.entries {
		buf.WriteString(entry.line + "\n")
	}

	fd, err := lock.Exclusive(filename)
	if err != nil {
		return fmt.Errorf("error while acquiring lock in %s: %s", filename, err)
	}
	defer lock.Release(fd)

	if err := c.file.Truncate(0); err != nil {
		return fmt.Errorf("error while truncating %s to 0: %s", filename, err)
	}
	if _, err := c.file.Seek(0, os.SEEK_SET); err != nil {
		return fmt.Errorf("error while resetting file offset: %s", err)
	}
	if _, err := c.file.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("error while writing configuration file %s: %s", filename, err)
	}

	return nil
}

// AddUser adds a user mapping entry, it will automatically
// find the first available range. It doesn't return any error
// if the user is already present and ignores the operation.
func (c *Config) AddUser(username string) error {
	_, err := c.GetUserEntry(username, false)
	if err == nil {
		return nil
	}

	u, err := c.getUserFn(username)
	if err != nil {
		return fmt.Errorf("could not retrieve user information for %s: %s", username, err)
	}
	for i := startMax; i >= startMin; i -= validRangeCount {
		current := i
		available := true
		for _, entry := range c.entries {
			if entry.invalid {
				continue
			}
			start := entry.Start
			end := entry.Start + entry.Count - 1
			if current >= start && current <= end {
				available = false
				break
			}
		}
		if available {
			c.requireUpdate = true
			line := fmt.Sprintf("%d:%d:%d", u.UID, current, validRangeCount)
			c.entries = append(
				c.entries,
				&Entry{
					UID:      u.UID,
					Start:    current,
					Count:    validRangeCount,
					disabled: false,
					line:     line,
				})
			return nil
		}
	}
	return fmt.Errorf("no range available")
}

// RemoveUser removes a user mapping entry. It returns an error
// if there is no entry for the user.
func (c *Config) RemoveUser(username string) error {
	e, err := c.GetUserEntry(username, false)
	if err != nil {
		return err
	}
	for i, entry := range c.entries {
		if entry.invalid {
			continue
		} else if entry == e {
			c.requireUpdate = true
			c.entries = append(c.entries[:i], c.entries[i+1:]...)
			break
		}
	}
	return nil
}

// EnableUser enables a previously disabled user mapping entry.
// It returns an error if there is no entry for the user but will
// ignore the operation if the user entry is already enabled.
func (c *Config) EnableUser(username string) error {
	e, err := c.GetUserEntry(username, false)
	if err != nil {
		return err
	}
	e.disabled = false
	if e.line[0] == disabledPrefix {
		c.requireUpdate = true
		e.line = e.line[1:]
	}
	return nil
}

// DisableUser disables a user entry mapping entry. It returns an
// error if there is no entry for the user but will ignore the
// operation if the user entry is already disabled.
func (c *Config) DisableUser(username string) error {
	e, err := c.GetUserEntry(username, false)
	if err != nil {
		return err
	}
	e.disabled = true
	if e.line[0] != disabledPrefix {
		c.requireUpdate = true
		e.line = fmt.Sprintf("%c%s", disabledPrefix, e.line)
	}
	return nil
}

// GetUserEntry returns a user entry associated to a user and returns
// an error if there is no entry for this user.
func (c *Config) GetUserEntry(username string, reportBadEntry bool) (*Entry, error) {
	entryCount := 0

	u, err := c.getUserFn(username)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve user information for %s: %s", username, err)
	}
	for _, entry := range c.entries {
		if entry.invalid {
			continue
		}
		if entry.UID == u.UID {
			if entry.Count == validRangeCount {
				return entry, nil
			}
			entryCount++
		}
	}
	if reportBadEntry && entryCount > 0 {
		return nil, fmt.Errorf(
			"mapping entries for user %s found in %s but all with a range count different from %d",
			username, c.file.Name(), validRangeCount,
		)
	}
	return nil, fmt.Errorf("no mapping entry found in %s for %s", c.file.Name(), username)
}

// GetIDRange determines UID/GID mappings based on configuration
// file provided in path.
func GetIDRange(path string, uid uint32) (*specs.LinuxIDMapping, error) {
	config, err := GetConfig(path, false, nil)
	if err != nil {
		return nil, err
	}
	defer config.Close()

	userinfo, err := user.GetPwUID(uid)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve user with UID %d: %s", uid, err)
	}
	e, err := config.GetUserEntry(userinfo.Name, true)
	if err != nil {
		return nil, err
	}
	if e.disabled {
		return nil, fmt.Errorf("your fakeroot mapping has been disabled by the administrator")
	}
	return &specs.LinuxIDMapping{
		ContainerID: 1,
		HostID:      e.Start,
		Size:        e.Count,
	}, nil
}
