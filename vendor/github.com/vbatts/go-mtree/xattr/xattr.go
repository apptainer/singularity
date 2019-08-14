// +build linux

package xattr

import (
	"strings"
	"syscall"
)

// Get returns the extended attributes (xattr) on file `path`, for the given `name`.
func Get(path, name string) ([]byte, error) {
	dest := make([]byte, 1024)
	i, err := syscall.Getxattr(path, name, dest)
	if err != nil {
		return nil, err
	}
	return dest[:i], nil
}

// Set sets the extended attributes (xattr) on file `path`, for the given `name` and `value`
func Set(path, name string, value []byte) error {
	return syscall.Setxattr(path, name, value, 0)
}

// List returns a list of all the extended attributes (xattr) for file `path`
func List(path string) ([]string, error) {
	dest := make([]byte, 1024)
	i, err := syscall.Listxattr(path, dest)
	if err != nil {
		return nil, err
	}

	// If the returned list is empty, return nil instead of []string{""}
	str := string(dest[:i])
	if str == "" {
		return nil, nil
	}

	return strings.Split(strings.TrimRight(str, nilByte), nilByte), nil
}

const nilByte = "\x00"
