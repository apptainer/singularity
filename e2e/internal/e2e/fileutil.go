// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
)

var (
	// cacheDirPriv is the directory the cachedir gets set to when running privileged.
	cacheDirPriv = ""
	// cacheDirUnpriv is the directory the cachedir gets set to when running unprivileged.
	cacheDirUnpriv = ""
)

// WriteTempFile creates and populates a temporary file in the specified
// directory or in os.TempDir if dir is ""
// returns the file name or an error
func WriteTempFile(dir, pattern, content string) (string, error) {
	tmpfile, err := ioutil.TempFile(dir, pattern)
	if err != nil {
		return "", err
	}

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		return "", err
	}

	if err := tmpfile.Close(); err != nil {
		return "", err
	}

	return tmpfile.Name(), nil
}

// MakeCacheDirs creates cache directories for privileged and unprivileged
// tests. Also set SINGULARITY_CACHEDIR environment variable for unprivileged
// context.
func MakeCacheDirs(baseDir string) error {
	if cacheDirPriv == "" {
		dir, err := fs.MakeTmpDir(baseDir, "privcache-", 0755)
		if err != nil {
			return fmt.Errorf("failed to create privileged cache directory: %s", err)
		}
		cacheDirPriv = dir
	}
	if cacheDirUnpriv == "" {
		dir, err := fs.MakeTmpDir(baseDir, "unprivcache-", 0755)
		if err != nil {
			return fmt.Errorf("failed to create unprivileged cache directory: %s", err)
		}
		cacheDirUnpriv = dir
		os.Setenv(cache.DirEnv, cacheDirUnpriv)
	}
	return nil
}
