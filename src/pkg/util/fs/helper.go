/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package fs

import (
	"os"
	"syscall"
)

// IsFile check if name component is regular file
func IsFile(name string) bool {
	info, err := os.Stat(name)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}

// IsDir check if name component is a directory
func IsDir(name string) bool {
	info, err := os.Stat(name)
	if err != nil {
		return false
	}
	return info.Mode().IsDir()
}

// IsLink check if name component is a symlink
func IsLink(name string) bool {
	info, err := os.Lstat(name)
	if err != nil {
		return false
	}
	return (info.Mode()&os.ModeSymlink != 0)
}

// IsOwner check if name component is owned by user identified with uid
func IsOwner(name string, uid uint32) bool {
	info, err := os.Stat(name)
	if err != nil {
		return false
	}
	return (info.Sys().(*syscall.Stat_t).Uid == uid)
}

// IsExec check if name component has executable bit permission set
func IsExec(name string) bool {
	info, err := os.Stat(name)
	if err != nil {
		return false
	}
	return (info.Sys().(*syscall.Stat_t).Mode&syscall.S_IXUSR != 0)
}

// IsSuid check if name component has setuid bit permission set
func IsSuid(name string) bool {
	info, err := os.Stat(name)
	if err != nil {
		return false
	}
	return (info.Sys().(*syscall.Stat_t).Mode&syscall.S_ISUID != 0)
}
