// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"os"

	"github.com/sylabs/singularity/internal/pkg/remote"
)

func syncSysConfig(cUsr *remote.Config, sysConfigFile string) error {
	// opening system config file
	f, err := os.OpenFile(sysConfigFile, os.O_RDONLY, 0600)
	if err != nil && os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("while opening remote config file: %s", err)
	}
	defer f.Close()

	// read file contents to config struct
	cSys, err := remote.ReadFrom(f)
	if err != nil {
		return fmt.Errorf("while parsing remote config data: %s", err)
	}

	// sync cUsr with system config cSys
	if err := cUsr.SyncFrom(cSys); err != nil {
		return err
	}

	return nil

}

// RemoteUse sets remote to use
func RemoteUse(usrConfigFile, sysConfigFile, name string, global bool) (err error) {
	c := &remote.Config{}

	// system config should be world readable
	perm := os.FileMode(0600)
	if global {
		perm = os.FileMode(0644)
	}

	// opening config file
	file, err := os.OpenFile(usrConfigFile, os.O_RDWR|os.O_CREATE, perm)
	if err != nil {
		return fmt.Errorf("while opening remote config file: %s", err)
	}
	defer file.Close()

	// read file contents to config struct
	c, err = remote.ReadFrom(file)
	if err != nil {
		return fmt.Errorf("while parsing remote config data: %s", err)
	}

	if err := syncSysConfig(c, sysConfigFile); err != nil {
		return err
	}

	if err := c.SetDefault(name); err != nil {
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
