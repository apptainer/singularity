/* 
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 *
 * Copyright (c) 2015-2017, Gregory M. Kurtzer. All rights reserved.
 * 
 * Copyright (c) 2016-2017, The Regents of the University of California,
 * through Lawrence Berkeley National Laboratory (subject to receipt of any
 * required approvals from the U.S. Dept. of Energy).  All rights reserved.
 * 
 * This software is licensed under a customized 3-clause BSD license.  Please
 * consult LICENSE file distributed with the sources of this project regarding
 * your rights to use or distribute this software.
 * 
 * NOTICE.  This Software was developed under funding from the U.S. Department of
 * Energy and the U.S. Government consequently retains certain rights. As such,
 * the U.S. Government has been granted for itself and others acting on its
 * behalf a paid-up, nonexclusive, irrevocable, worldwide license in the Software
 * to reproduce, distribute copies to the public, prepare derivative works, and
 * perform publicly and display publicly, and to permit other to do so. 
 * 
*/

#define _GNU_SOURCE
#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/mount.h>
#include <unistd.h>
#include <stdlib.h>
#include <sched.h>

#include "util/file.h"
#include "util/util.h"
#include "util/registry.h"
#include "util/message.h"
#include "util/config_parser.h"
#include "util/privilege.h"
#include "util/setns.h"
#include "util/mount.h"


static int enabled = -1;


int _singularity_runtime_ns_mnt(void) {
    int slave;

    slave = singularity_config_get_bool(MOUNT_SLAVE);

    singularity_priv_escalate();
#ifdef NS_CLONE_FS
    singularity_message(DEBUG, "Virtualizing FS namespace\n");
    if ( unshare(CLONE_FS) < 0 ) {
        singularity_message(ERROR, "Could not virtualize file system namespace: %s\n", strerror(errno));
        ABORT(255);
    }
#endif

    singularity_message(DEBUG, "Virtualizing mount namespace\n");
    if ( unshare(CLONE_NEWNS) < 0 ) {
        singularity_message(ERROR, "Could not virtualize mount namespace: %s\n", strerror(errno));
        ABORT(255);
    }

    // Privatize the mount namespaces
    //
#ifdef SINGULARITY_MS_SLAVE
    singularity_message(DEBUG, "Making mounts %s\n", (slave ? "slave" : "private"));
    // The strange formatting here is to avoid SonarQube complaints about bitwise or of signed operands.
    unsigned mount_flags = MS_REC;
    mount_flags |= (unsigned)(slave ? MS_SLAVE : MS_PRIVATE);
    if ( singularity_mount(NULL, "/", NULL, mount_flags, NULL) < 0 ) {
        singularity_message(ERROR, "Could not make mountspaces %s: %s\n", (slave ? "slave" : "private"), strerror(errno));
        ABORT(255);
    }
#else
    if ( slave > 0 ) {
        singularity_message(WARNING, "Requested option 'mount slave' is not available on this host, using private\n");
    }
    singularity_message(DEBUG, "Making mounts private\n");
    if ( singularity_mount(NULL, "/", NULL, MS_PRIVATE | MS_REC, NULL) < 0 ) {
        singularity_message(ERROR, "Could not make mountspaces %s: %s\n", (slave ? "slave" : "private"), strerror(errno));
        ABORT(255);
    }
#endif

    singularity_priv_drop();
    enabled = 0;
    return(0);
}

int _singularity_runtime_ns_mnt_join(void) {
    int ns_fd = atoi(singularity_registry_get("DAEMON_NS_FD"));
    int mnt_fd;

    /* Attempt to open /proc/[MNT]/ns/mnt */
    singularity_priv_escalate();
    mnt_fd = openat(ns_fd, "mnt", O_RDONLY);

    if( mnt_fd == -1 ) {
        singularity_message(ERROR, "Could not open mount NS fd: %s\n", strerror(errno));
        ABORT(255);
    }
    
    singularity_message(DEBUG, "Attempting to join mount namespace\n");
    if ( setns(mnt_fd, CLONE_NEWNS) < 0 ) {
        singularity_message(ERROR, "Could not join mount namespace: %s\n", strerror(errno));
        ABORT(255);
    }
    singularity_priv_drop();
    singularity_message(DEBUG, "Successfully joined mount namespace\n");

    close(ns_fd);
    return(0);    
}

/*
int singularity_ns_mnt_enabled(void) {
    singularity_message(DEBUG, "Checking MNT namespace enabled: %d\n", enabled);
    return(enabled);
}
*/
