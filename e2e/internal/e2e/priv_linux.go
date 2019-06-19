// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

/*
#define _GNU_SOURCE
#include <unistd.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <errno.h>

#define SIZE    128

// getProcInfo returns the parent PID, UID, and GID associated with the
// supplied PID.
static pid_t getProcInfo(pid_t pid, uid_t *uid, gid_t *gid) {
	FILE *status;
	char procPath[SIZE];
	char *line = NULL;
	size_t len = 0;
	pid_t ppid = 1;

	memset(procPath, 0, SIZE);
	snprintf(procPath, SIZE-1, "/proc/%d/status", pid);

	status = fopen(procPath, "r");
	if ( status == NULL ) {
		return 1;
	}

	while ( getline(&line, &len, status) != -1 ) {
		if ( ppid == 1 ) {
			sscanf(line, "PPid:\t%d", &ppid);
		}
		if ( *uid == 0 ) {
			sscanf(line, "Uid:\t%d", uid);
		}
		if ( *gid == 0 ) {
			sscanf(line, "Gid:\t%d", gid);
		}
	}

	free(line);
	fclose(status);

	return ppid;
}

// getUnprivIDs searches recursively up the process parent chain to find a
// process with a non-root UID, then returns the UID and GID of that process.
static int getUnprivIDs(pid_t pid, uid_t *uid, gid_t *gid) {
	if ( pid == 1 ) {
		return -1;
	}
	pid_t ppid = getProcInfo(pid, uid, gid);
	if ( *uid == 0 ) {
		return getUnprivIDs(ppid, uid, gid);
	}
	return 0;
}

__attribute__((constructor)) static void init(void) {
	uid_t uid = 0;
	gid_t gid = 0;

	if ( getUnprivIDs(getppid(), &uid, &gid) < 0 ) {
		fprintf(stderr, "failed to retrieve user information\n");
		exit(1);
	}
	if ( uid == 0 || gid == 0 ) {
		fprintf(stderr, "failed to retrieve user information\n");
		exit(1);
	}
	if ( setresgid(gid, gid, 0) < 0 ) {
		fprintf(stderr, "setresgid failed: %s\n", strerror(errno));
		exit(1);
	}
	if ( setresuid(uid, uid, 0) < 0 ) {
		fprintf(stderr, "setresuid failed: %s\n", strerror(errno));
		exit(1);
	}
}
*/
import "C"

import (
	"os"
	"runtime"
	"syscall"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/client/cache"

	"github.com/sylabs/singularity/internal/pkg/util/user"
)

var (
	// uid user running test.
	uid = os.Getuid()
	// gid group running test.
	gid = os.Getgid()
)

// Privileged wraps the supplied test function with calls to ensure
// the test is run with elevated privileges.
func Privileged(f func(*testing.T)) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()

		runtime.LockOSThread()

		if err := syscall.Setresuid(0, 0, uid); err != nil {
			t.Fatalf("privileges escalation failed: %s", err)
		}
		if err := syscall.Setresgid(0, 0, gid); err != nil {
			t.Fatalf("privileges escalation failed: %s", err)
		}
		// NEED FIX: it shouldn't be set/restored globally, only
		// when executing singularity command with privileges.
		os.Setenv(cache.DirEnv, cacheDirPriv)

		defer func() {
			if err := syscall.Setresgid(gid, gid, 0); err != nil {
				t.Fatalf("privileges drop failed: %s", err)
			}
			if err := syscall.Setresuid(uid, uid, 0); err != nil {
				t.Fatalf("privileges drop failed: %s", err)
			}
			// NEED FIX: see above comment
			os.Setenv(cache.DirEnv, cacheDirUnpriv)
			runtime.UnlockOSThread()
		}()

		f(t)
	}
}

// CurrentUser returns the current user account information. Use of user.Current is
// not safe with e2e tests as the user information is cached after the first call,
// so it will always return the same user information which could be wrong if
// user.Current was first called in unprivileged context and called after in a
// privileged context as it will return information of unprivileged user.
func CurrentUser(t *testing.T) *user.User {
	u, err := user.GetPwUID(uint32(os.Getuid()))
	if err != nil {
		t.Fatalf("failed to retrieve user information")
	}
	return u
}
