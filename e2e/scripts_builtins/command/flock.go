// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package command

import (
	"context"
	"fmt"
	"sync"

	"github.com/sylabs/singularity/pkg/util/fs/lock"

	"github.com/sylabs/singularity/pkg/stest"
	"mvdan.cc/sh/v3/interp"
)

// holds all locks and protect it with a mutex
var locks = make(map[string]int)
var m sync.Mutex

// flock builtin
// usage:
// flock lock|unlock filepath
func flock(ctx context.Context, mc interp.ModuleCtx, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("flock need two arguments")
	}

	switch args[0] {
	case "lock":
		fd, err := lock.Exclusive(args[1])
		if err != nil {
			return fmt.Errorf("error while locking file %s: %s", args[1], err)
		}
		m.Lock()
		locks[args[1]] = fd
		m.Unlock()
	case "unlock":
		m.Lock()
		fd, ok := locks[args[1]]
		if !ok {
			return nil
		}
		delete(locks, args[1])
		m.Unlock()
		return lock.Release(fd)
	}

	return nil
}

func init() {
	stest.RegisterCommandBuiltin("flock", flock)
}
