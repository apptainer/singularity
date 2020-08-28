// Copyright (c) 2020, Control Command Inc. All rights reserved.
// Copyright (c) 2019-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"os"

	"github.com/sylabs/singularity/internal/pkg/remote"
	"github.com/sylabs/singularity/internal/pkg/remote/endpoint"
	"github.com/sylabs/singularity/internal/pkg/util/auth"
)

type LoginArgs struct {
	Name      string
	Username  string
	Password  string
	Tokenfile string
	Insecure  bool
}

// RemoteLogin logs in remote by setting API token
// If the supplied remote name is an empty string, it will attempt
// to use the default remote.
func RemoteLogin(usrConfigFile string, args *LoginArgs) (err error) {
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

	if err := syncSysConfig(c); err != nil {
		return err
	}

	var r *endpoint.Config
	if args.Name == "" {
		r, err = c.GetDefault()
	} else {
		r, err = c.GetRemote(args.Name)
	}

	if r != nil {
		// endpoint
		if args.Tokenfile != "" {
			var authWarning string
			r.Token, authWarning = auth.ReadToken(args.Tokenfile)
			if authWarning != "" {
				return fmt.Errorf("while reading tokenfile: %s", authWarning)
			}
		}
		if err := r.VerifyToken(); err != nil {
			return fmt.Errorf("while verifying token: %v", err)
		}
	} else {
		// services
		if err := c.Login(args.Name, args.Username, args.Password, args.Insecure); err != nil {
			return fmt.Errorf("while login to %s: %s", args.Name, err)
		}
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
