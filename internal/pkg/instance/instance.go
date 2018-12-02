// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package instance

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/util/fs/proc"
	"github.com/sylabs/singularity/internal/pkg/util/user"
)

const (
	privPath        = "/var/run/singularity/instances"
	unprivPath      = ".singularity/instances"
	authorizedChars = `^[a-zA-Z0-9._-]+$`
	prognameFormat  = "Singularity instance: %s [%s]"
)

// File represents an instance file storing instance information
type File struct {
	Path       string `json:"-"`
	Pid        int    `json:"pid"`
	PPid       int    `json:"ppid"`
	Name       string `json:"name"`
	User       string `json:"user"`
	Image      string `json:"image"`
	Privileged bool   `json:"privileged"`
	Config     []byte `json:"config"`
}

// ProcName returns processus name based on instance name
// and username
func ProcName(name string, username string) string {
	return fmt.Sprintf(prognameFormat, username, name)
}

// ExtractName extracts instance name from an instance:// URI
func ExtractName(name string) string {
	return strings.Replace(name, "instance://", "", 1)
}

// CheckName checks if name is a valid instance name
func CheckName(name string) error {
	r := regexp.MustCompile(authorizedChars)
	if !r.MatchString(name) {
		return fmt.Errorf("%s is not a valid instance name", name)
	}
	return nil
}

// getPath returns the path where searching for instance files
func getPath(privileged bool, username string) (string, error) {
	path := ""
	var pw *user.User
	var err error

	if username == "" {
		if pw, err = user.GetPwUID(uint32(os.Getuid())); err != nil {
			return path, err
		}
	} else {
		if pw, err = user.GetPwNam(username); err != nil {
			return path, err
		}
	}

	if privileged {
		path = filepath.Join(privPath, pw.Name)
		return path, nil
	}

	containerID, hostID, err := proc.ReadIDMap("/proc/self/uid_map")
	if containerID == 0 && containerID != hostID {
		if pw, err = user.GetPwUID(hostID); err != nil {
			return path, err
		}
	}

	hostname, err := os.Hostname()
	if err != nil {
		return path, err
	}

	path = filepath.Join(pw.Dir, unprivPath, hostname, pw.Name)
	return path, nil
}

func getDir(privileged bool, name string) (string, error) {
	if err := CheckName(name); err != nil {
		return "", err
	}
	path, err := getPath(privileged, "")
	if err != nil {
		return "", err
	}
	return filepath.Join(path, name), nil
}

// GetDirPrivileged returns directory where instances file will be stored
// if instance is run with privileges
func GetDirPrivileged(name string) (string, error) {
	return getDir(true, name)
}

// GetDirUnprivileged returns directory where instances file will be stored
// if instance is run without privileges
func GetDirUnprivileged(name string) (string, error) {
	return getDir(false, name)
}

// Get returns the instance file corresponding to instance name
func Get(name string) (*File, error) {
	if err := CheckName(name); err != nil {
		return nil, err
	}
	list, err := List("", name)
	if err != nil {
		return nil, err
	}
	if len(list) != 1 {
		return nil, fmt.Errorf("no instance found with name %s", name)
	}
	return list[0], nil
}

// Add creates an instance file for a named instance in a privileged
// or unprivileged path
func Add(name string, privileged bool) (*File, error) {
	if err := CheckName(name); err != nil {
		return nil, err
	}
	_, err := Get(name)
	if err == nil {
		return nil, fmt.Errorf("instance %s already exists", name)
	}
	i := &File{Name: name, Privileged: privileged}
	i.Path, err = getPath(privileged, "")
	if err != nil {
		return nil, err
	}
	jsonFile := name + ".json"
	i.Path = filepath.Join(i.Path, name, jsonFile)
	return i, nil
}

// List returns instance files matching username and/or name pattern
func List(username string, name string) ([]*File, error) {
	list := make([]*File, 0)
	privileged := true

	for {
		path, err := getPath(privileged, username)
		if err != nil {
			return nil, err
		}
		pattern := filepath.Join(path, name, name+".json")
		files, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			r, err := os.Open(file)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return nil, err
			}
			b, err := ioutil.ReadAll(r)
			r.Close()
			if err != nil {
				return nil, err
			}
			f := &File{Path: file}
			if err := json.Unmarshal(b, f); err != nil {
				return nil, err
			}
			list = append(list, f)
		}
		privileged = !privileged
		if privileged {
			break
		}
	}

	return list, nil
}

// PrivilegedPath returns if instance file is stored in privileged path or not
func (i *File) PrivilegedPath() bool {
	return strings.HasPrefix(i.Path, privPath)
}

// Delete deletes instance file
func (i *File) Delete() error {
	path := filepath.Dir(i.Path)
	return os.RemoveAll(path)
}

// Update stores instance information in associated instance file
func (i *File) Update() error {
	b, err := json.Marshal(i)
	if err != nil {
		return err
	}
	path := filepath.Dir(i.Path)

	oldumask := syscall.Umask(0)
	defer syscall.Umask(oldumask)

	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}
	file, err := os.OpenFile(i.Path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	b = append(b, '\n')
	if n, err := file.Write(b); err != nil || n != len(b) {
		return fmt.Errorf("failed to write instance file %s: %s", i.Path, err)
	}

	return file.Sync()
}

// SetLogFile replaces stdout/stderr streams and redirect content
// to log file
func SetLogFile(name string, uid int) (*os.File, *os.File, error) {
	path, err := getPath(false, "")
	if err != nil {
		return nil, nil, err
	}
	stderrPath := filepath.Join(path, name+".err")
	stdoutPath := filepath.Join(path, name+".out")

	oldumask := syscall.Umask(0)
	defer syscall.Umask(oldumask)

	if err := os.MkdirAll(filepath.Dir(stderrPath), 0755); err != nil {
		return nil, nil, err
	}
	if err := os.MkdirAll(filepath.Dir(stdoutPath), 0755); err != nil {
		return nil, nil, err
	}

	stderr, err := os.OpenFile(stderrPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, nil, err
	}

	stdout, err := os.OpenFile(stdoutPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, nil, err
	}

	if uid != os.Getuid() || uid == 0 {
		if err := stderr.Chown(uid, os.Getgid()); err != nil {
			return nil, nil, err
		}
		if err := stdout.Chown(uid, os.Getgid()); err != nil {
			return nil, nil, err
		}
	}

	return stdout, stderr, nil
}
