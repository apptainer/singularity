// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"os"

	"github.com/sylabs/singularity/internal/pkg/remote"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/auth"
	"github.com/sylabs/singularity/pkg/sypgp"
)

// RemoteLogin logs in remote by setting API token
func RemoteLogin(usrConfigFile, sysConfigFile, name, tokenfile string) (err error) {
	c := &remote.Config{}

	// opening config file
	file, err := os.OpenFile(usrConfigFile, os.O_RDWR|os.O_CREATE, 0600)
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

	r, err := c.GetRemote(name)
	if err != nil {
		return err
	}

	if tokenfile != "" {
		var authWarning string
		r.Token, authWarning = auth.ReadToken(tokenfile)
		if authWarning != "" {
			return fmt.Errorf("while reading tokenfile: %s", authWarning)
		}
	} else {
		fmt.Printf("Generate an API Key at https://%s/auth/tokens, and paste here:\n", r.URI)
		r.Token, err = sypgp.AskQuestionNoEcho("API Key: ")
		if err != nil {
			return err
		}
	}

	if err := r.VerifyToken(); err != nil {
		return fmt.Errorf("while verifying token: %v", err)
	}

	sylog.Infof("API Key Verified!")

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
