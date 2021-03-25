// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package init

/*
#define _GNU_SOURCE
#include <unistd.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <errno.h>
#include <sched.h>
#include <sys/mount.h>
#include <sys/types.h>
#include <sys/wait.h>

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
	if ( snprintf(procPath, SIZE-1, "/proc/%d/status", pid) > SIZE-1 ) {
		// set returned PID to 1 to trigger error from getUnprivIDs call
		return 1;
	}

	status = fopen(procPath, "r");
	if ( status == NULL ) {
		// set returned PID to 1 to trigger error from getUnprivIDs call
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
	// PID 1 here means we didn't find a process containing
	// identity of the original user or an error occurred in
	// getProcInfo
	if ( pid == 1 ) {
		return -1;
	}
	pid_t ppid = getProcInfo(pid, uid, gid);
	if ( *uid == 0 || *gid == 0 ) {
		return getUnprivIDs(ppid, uid, gid);
	}
	return 0;
}

// create and use a PID namespace if possible to avoid leaving some processes
// once tests are done. Child process won't catch orphaned child processes like
// instances, and we can't really catch them correctly to avoid conflicts during
// `cmd.Wait()` calls. But this is not a big deal compared to detached processes
// that could keep running on host machine after the tests execution.
static void create_pid_namespace(void) {
	if ( unshare(CLONE_NEWPID) == 0 ) {
		pid_t forked = fork();
		if ( forked > 0 ) {
			// parent process will wait that tests execution finished
			int status, exit_status = 0;
			pid_t child;

			child = waitpid(forked, &status, 0);
			if ( child < 0 ) {
				fprintf(stderr, "unexpected error while waiting children: %s\n", strerror(errno));
				exit(1);
			}

			if ( WIFEXITED(status) ) {
				if ( WEXITSTATUS(status) != 0 ) {
					exit_status = WEXITSTATUS(status);
				}
			} else if ( WIFSIGNALED(status) ) {
				kill(getpid(), WTERMSIG(status));
				exit_status = 128 + WTERMSIG(status);
			}
			exit(exit_status);
		}

		// mount a new proc filesystem for the new PID namespace
		if ( mount(NULL, "/proc", "proc", MS_NOSUID|MS_NODEV, NULL) < 0 ) {
			fprintf(stderr, "failed to set private mount propagation: %s\n", strerror(errno));
			exit(1);
		}
		// return to the child process
	}
}

// create and use a mount namespace in order to bind a temporary
// filesystem on top of home directories and not screw them up by
// accident during tests execution.
static void create_mount_namespace(void) {
	if ( unshare(CLONE_FS) < 0 ) {
		fprintf(stderr, "failed to unshare filesystem: %s\n", strerror(errno));
		exit(1);
	}
	if ( unshare(CLONE_NEWNS) < 0 ) {
		fprintf(stderr, "failed to create mount namespace: %s\n", strerror(errno));
		exit(1);
	}
	if ( mount(NULL, "/", NULL, MS_PRIVATE|MS_REC, NULL) < 0 ) {
		fprintf(stderr, "failed to set private mount propagation: %s\n", strerror(errno));
		exit(1);
	}
}

// This is the CGO init constructor called before executing any Go code
// in e2e/e2e_test.go.
__attribute__((constructor)) static void init(void) {
	uid_t uid = 0;
	gid_t gid = 0;

	if ( getuid() != 0 ) {
		fprintf(stderr, "tests must be executed as root user\n");
		fprintf(stderr, "%d %d", uid, gid);
		exit(1);
	}
	if ( getUnprivIDs(getppid(), &uid, &gid) < 0 ) {
		fprintf(stderr, "failed to retrieve user information\n");
		exit(1);
	}
	if ( uid == 0 || gid == 0 ) {
		fprintf(stderr, "failed to retrieve user information\n");
		exit(1);
	}

	create_mount_namespace();
	create_pid_namespace();

	// set original user identity and retain privileges for
	// Privileged method
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
