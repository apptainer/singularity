// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package files

import (
	"path/filepath"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
)

// joinKeepSlash joins path to prefix, ensuring that if path ends with a "/" it
// is preserved in the result, as may be required when calling out to commands
// for which this is meaningful.
func joinKeepSlash(prefix, path string) string {
	fullPath := filepath.Join(prefix, path)
	// append a slash if path ended with a trailing '/', second check
	// makes sure we don't return a double slash
	if strings.HasSuffix(path, "/") && !strings.HasSuffix(fullPath, "/") {
		fullPath += "/"
	}
	return fullPath
}

// secureJoinKeepSlash joins path to prefix, but guarantees the resulting path is under prefix.
// If path ends with a "/" it is preserved in the result, as may be required when calling
// out to commands for which this is meaningful.
func secureJoinKeepSlash(prefix, path string) (string, error) {
	fullPath, err := securejoin.SecureJoin(prefix, path)
	if err != nil {
		return "", err
	}
	// append a slash if path ended with a trailing '/', second check
	// makes sure we don't return a double slash
	if strings.HasSuffix(path, "/") && !strings.HasSuffix(fullPath, "/") {
		fullPath += "/"
	}
	return fullPath, nil
}
