// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"encoding/json"
	"fmt"

	"github.com/sylabs/singularity/pkg/util/unix"
)

// OciState query container state
func OciState(containerID string, args *OciArgs) error {
	// query instance files and returns state
	state, err := getState(containerID)
	if err != nil {
		return err
	}
	if args.SyncSocketPath != "" {
		data, err := json.Marshal(state)
		if err != nil {
			return fmt.Errorf("failed to marshal state data: %s", err)
		} else if err := unix.WriteSocket(args.SyncSocketPath, data); err != nil {
			return err
		}
	} else {
		c, err := json.MarshalIndent(state, "", "\t")
		if err != nil {
			return err
		}
		fmt.Println(string(c))
	}
	return nil
}
