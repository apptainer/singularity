// Copyright (c) 2020, Control Command Inc. All rights reserved.
// Copyright (c) 2019-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"errors"
	"fmt"
	"os"

	"github.com/hpcng/singularity/internal/pkg/remote"
	"github.com/hpcng/singularity/internal/pkg/remote/endpoint"
	"github.com/hpcng/singularity/internal/pkg/util/auth"
	"github.com/hpcng/singularity/internal/pkg/util/interactive"
	"github.com/hpcng/singularity/pkg/sylog"
)

type LoginArgs struct {
	Name      string
	Username  string
	Password  string
	Tokenfile string
	Insecure  bool
}

// ErrLoginAborted is raised when the login process has been aborted by the user
var ErrLoginAborted = errors.New("user aborted login")

// RemoteLogin logs in remote by setting API token
// If the supplied remote name is an empty string, it will attempt
// to use the default remote.
func RemoteLogin(usrConfigFile string, args *LoginArgs) (err error) {
	c := &remote.Config{}

	// opening config file
	file, err := os.OpenFile(usrConfigFile, os.O_RDWR|os.O_CREATE, 0o600)
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
		// endpoints (sylabs cloud, singularity enterprise etc.)
		err := endPointLogin(r, args)
		if err == ErrLoginAborted {
			return nil
		}
		if err != nil {
			return err
		}
	} else {
		// services (oci registry, single keyserver etc.)
		if args.Tokenfile != "" {
			return fmt.Errorf("--tokenfile is only supported for login to a remote endpoint, not OCI (docker/oras) or keyservers")
		}
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

	sylog.Infof("Token stored in %s", file.Name())
	return nil
}

// endPointLogin implements the flow to set a new token against a remote endpoing config.
// A token may be provided with a file, or through interactive prompts.
func endPointLogin(ep *endpoint.Config, args *LoginArgs) error {
	var (
		token string
		err   error
	)
	// Non-interactive with a token file
	if args.Tokenfile != "" {
		token, err = auth.ReadToken(args.Tokenfile)
		if err != nil {
			return fmt.Errorf("while reading tokenfile: %s", err)
		}
	} else {
		// Interactive login
		// If a token is already set, prompt to see if we want to replace it
		if ep.Token != "" {
			input, err := interactive.AskYNQuestion("n", "An access token is already set for this remote. Replace it? [N/y] ")
			if err != nil {
				return fmt.Errorf("while reading input: %s", err)
			}
			if input == "n" {
				return ErrLoginAborted
			}
		}

		webURL, err := ep.GetURL()
		if err != nil {
			return err
		}

		fmt.Printf("Generate an access token at %s/auth/tokens, and paste it here.\n", webURL)
		fmt.Println("Token entered will be hidden for security.")
		token, err = interactive.AskQuestionNoEcho("Access Token: ")
		if err != nil {
			return err
		}
		// No token was entered
		if token == "" {
			return ErrLoginAborted
		}
	}

	// We now have a token to check... *before* we assign it to the endpoint config
	if err := ep.VerifyToken(token); err != nil {
		return fmt.Errorf("while verifying token: %v", err)
	}
	// Token is verified, update the endpoint config with it
	ep.Token = token
	return nil
}
