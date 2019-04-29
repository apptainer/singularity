// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package instance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/util/user"
)

const (
	// OciSubDir represents directory where OCI instance files are stored
	OciSubDir = "oci"
	// SingSubDir represents directory where Singularity instance files are stored
	SingSubDir = "sing"
	// LogSubDir represents directory where Singularity instance log files are stored
	LogSubDir = "logs"
)

const (
	instancePath    = ".singularity/instances"
	authorizedChars = `^[a-zA-Z0-9._-]+$`
	prognameFormat  = "Singularity instance: %s [%s]"
)

// File represents an instance file storing instance information
type File struct {
	Path   string `json:"-"`
	Pid    int    `json:"pid"`
	PPid   int    `json:"ppid"`
	Name   string `json:"name"`
	User   string `json:"user"`
	Image  string `json:"image"`
	Config []byte `json:"config"`
	UserNs bool   `json:"userns"`
}

// ProcName returns processus name based on instance name
// and username
func ProcName(name string, username string) (string, error) {
	if err := CheckName(name); err != nil {
		return "", fmt.Errorf("while checking instance name: %s", err)
	}
	if username == "" {
		return "", fmt.Errorf("while getting instance processus name: empty username")
	}
	return fmt.Sprintf(prognameFormat, username, name), nil
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
func getPath(username string, subDir string) (string, error) {
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

	hostname, err := os.Hostname()
	if err != nil {
		return path, err
	}

	path = filepath.Join(pw.Dir, instancePath, subDir, hostname, pw.Name)
	return path, nil
}

// GetDir returns directory where instances file will be stored
func GetDir(name string, subDir string) (string, error) {
	if err := CheckName(name); err != nil {
		return "", err
	}
	path, err := getPath("", subDir)
	if err != nil {
		return "", err
	}
	return filepath.Join(path, name), nil
}

// Get returns the instance file corresponding to instance name
func Get(name string, subDir string) (*File, error) {
	if err := CheckName(name); err != nil {
		return nil, err
	}
	list, err := List("", name, subDir)
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
func Add(name string, subDir string) (*File, error) {
	if err := CheckName(name); err != nil {
		return nil, err
	}
	_, err := Get(name, subDir)
	if err == nil {
		return nil, fmt.Errorf("instance %s already exists", name)
	}
	i := &File{Name: name}
	i.Path, err = getPath("", subDir)
	if err != nil {
		return nil, err
	}
	jsonFile := name + ".json"
	i.Path = filepath.Join(i.Path, name, jsonFile)
	return i, nil
}

// List returns instance files matching username and/or name pattern
func List(username string, name string, subDir string) ([]*File, error) {
	list := make([]*File, 0)

	path, err := getPath(username, subDir)
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
		} else if err != nil {
			return nil, err
		}
		f := &File{}
		if err := json.NewDecoder(r).Decode(f); err != nil {
			r.Close()
			return nil, err
		}
		r.Close()
		f.Path = file
		list = append(list, f)
	}

	return list, nil
}

// Delete deletes instance file
func (i *File) Delete() error {
	return os.RemoveAll(filepath.Dir(i.Path))
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
	file, err := os.OpenFile(i.Path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY|syscall.O_NOFOLLOW, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.Write(b); err != nil {
		return fmt.Errorf("failed to write instance file %s: %s", i.Path, err)
	}

	return file.Sync()
}

// SetLogFile replaces stdout/stderr streams and redirect content
// to log file
func SetLogFile(name string, uid int, subDir string) (*os.File, *os.File, error) {
	path, err := getPath("", subDir)
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

	stderr, err := os.OpenFile(stderrPath, os.O_RDWR|os.O_CREATE|os.O_APPEND|syscall.O_NOFOLLOW, 0644)
	if err != nil {
		return nil, nil, err
	}

	stdout, err := os.OpenFile(stdoutPath, os.O_RDWR|os.O_CREATE|os.O_APPEND|syscall.O_NOFOLLOW, 0644)
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
