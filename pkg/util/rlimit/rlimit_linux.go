// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package rlimit

import (
	"fmt"
	"syscall"
)

var resource = map[string]int{
	"RLIMIT_CPU":        0,
	"RLIMIT_FSIZE":      1,
	"RLIMIT_DATA":       2,
	"RLIMIT_STACK":      3,
	"RLIMIT_CORE":       4,
	"RLIMIT_RSS":        5,
	"RLIMIT_NPROC":      6,
	"RLIMIT_NOFILE":     7,
	"RLIMIT_MEMLOCK":    8,
	"RLIMIT_AS":         9,
	"RLIMIT_LOCKS":      10,
	"RLIMIT_SIGPENDING": 11,
	"RLIMIT_MSGQUEUE":   12,
	"RLIMIT_NICE":       13,
	"RLIMIT_RTPRIO":     14,
	"RLIMIT_RTTIME":     15,
}

// Set sets soft and hard resource limit
func Set(res string, cur uint64, max uint64) error {
	var rlim syscall.Rlimit

	resVal, ok := resource[res]
	if !ok {
		return fmt.Errorf("%s is not a valid resource type", res)
	}

	rlim.Cur = cur
	rlim.Max = max

	if err := syscall.Setrlimit(resVal, &rlim); err != nil {
		return fmt.Errorf("failed to set resource limit %s: %s", res, err)
	}

	return nil
}

// Get retrieves soft and hard resource limit
func Get(res string) (cur uint64, max uint64, err error) {
	var rlim syscall.Rlimit

	resVal, ok := resource[res]
	if !ok {
		err = fmt.Errorf("%s is not a valid resource type", res)
		return
	}

	if err = syscall.Getrlimit(resVal, &rlim); err != nil {
		err = fmt.Errorf("failed to get resource limit %s: %s", res, err)
		return
	}

	cur = rlim.Cur
	max = rlim.Max

	return
}
