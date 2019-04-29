// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package files

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

const filenameExpansionScript = `for n in %[1]s ; do
	printf "$n\0"
done
`

func expandPath(path string) ([]string, error) {
	var output, stderr bytes.Buffer
	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf(filenameExpansionScript, path))
	cmd.Stdout = &output
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s: %s", err, stderr.String())
	}

	// parse expanded output and ignore empty strings from consecutive null bytes
	var paths []string
	for _, s := range strings.Split(output.String(), "\x00") {
		if s == "" {
			continue
		}
		paths = append(paths, s)
	}

	return paths, nil
}

// AddPrefix prepends the supplied prefix to the path, ensuring a trailing '/' in the path
// since that is meaningful to the 'cp' command
func AddPrefix(prefix, path string) string {
	fullPath := filepath.Join(prefix, path)
	// append a slash if path ended with a trailing '/', second check
	// makes sure we don't return a double slash
	if strings.HasSuffix(path, "/") && !strings.HasSuffix(fullPath, "/") {
		fullPath += "/"
	}

	return fullPath
}
