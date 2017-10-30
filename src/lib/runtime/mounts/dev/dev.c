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

#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <sys/mount.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <unistd.h>
#include <stdlib.h>
#include <grp.h>

#include "config.h"
#include "util/file.h"
#include "util/util.h"
#include "util/message.h"
#include "util/privilege.h"
#include "util/config_parser.h"
#include "util/registry.h"
#include "util/mount.h"

#include "../../runtime.h"

static int bind_dev(char *tmpdir, char *dev);


int _singularity_runtime_mount_dev(void) {
    char *container_dir = CONTAINER_FINALDIR;

    if ( ( singularity_registry_get("CONTAIN") != NULL ) || ( strcmp("minimal", singularity_config_get_value(MOUNT_DEV)) == 0 ) ) {
        char *sessiondir = singularity_registry_get("SESSIONDIR");
        char *devdir = joinpath(sessiondir, "/dev");

        if ( is_dir(joinpath(container_dir, "/dev")) < 0 ) {
            int ret;

            if ( singularity_registry_get("OVERLAYFS_ENABLED") == NULL ) {
                singularity_message(WARNING, "Not mounting devices as /dev directory does not exist within container\n");
                return(-1);
            }

            singularity_priv_escalate();
            ret = s_mkpath(joinpath(container_dir, "/dev"), 0755);
            singularity_priv_drop();

            if ( ret < 0 ) {
                singularity_message(ERROR, "Could not create /dev inside container\n");
                ABORT(255);
            }
        }

        singularity_message(DEBUG, "Creating temporary staged /dev\n");
        if ( s_mkpath(devdir, 0755) != 0 ) {
            singularity_message(ERROR, "Failed creating the session device directory %s: %s\n", devdir, strerror(errno));
            ABORT(255);
        }

        singularity_message(DEBUG, "Creating temporary staged /dev/shm\n");
        if ( s_mkpath(joinpath(devdir, "/shm"), 0755) != 0 ) {
            singularity_message(ERROR, "Failed creating temporary /dev/shm %s: %s\n", joinpath(devdir, "/shm"), strerror(errno));
            ABORT(255);
        }

        if ( singularity_config_get_bool_char(MOUNT_DEVPTS) > 0 ) {
            struct stat multi_instance_devpts;
            
            if( stat("/dev/pts/ptmx", &multi_instance_devpts) < 0 ) {
                singularity_message(ERROR, "Multiple devpts instances unsupported and \"%s\" configured\n", MOUNT_DEVPTS);
                ABORT(255);
            }
            singularity_message(DEBUG, "Creating staged /dev/pts\n");
            if ( s_mkpath(joinpath(devdir, "/pts"), 0755) != 0 ) {
                singularity_message(ERROR, "Failed creating /dev/pts %s: %s\n", joinpath(devdir, "/pts"), strerror(errno));
                ABORT(255);
            }
            bind_dev(sessiondir, "/dev/tty");
        }

        bind_dev(sessiondir, "/dev/null");
        bind_dev(sessiondir, "/dev/zero");
        bind_dev(sessiondir, "/dev/random");
        bind_dev(sessiondir, "/dev/urandom");

        singularity_priv_escalate();
        singularity_message(DEBUG, "Mounting tmpfs for staged /dev/shm\n");
        if ( singularity_mount("/dev/shm", joinpath(devdir, "/shm"), "tmpfs", MS_NOSUID, "") < 0 ) {
            singularity_message(ERROR, "Failed to mount %s: %s\n", joinpath(devdir, "/shm"), strerror(errno));
            ABORT(255);
        }

        if ( singularity_config_get_bool_char(MOUNT_DEVPTS) > 0 ) {
            struct group *ttygid;
            char *devpts_opts_base="mode=0620,newinstance,ptmxmode=0666,gid=";
            char *devpts_opts;
            unsigned int max_sz, gd, gd_n;

            if ( (ttygid=getgrnam("tty")) == NULL ) {
                singularity_message(ERROR, "Problem resolving 'tty' group GID: %s\n", strerror(errno));
                ABORT(255);
            }

            /* number of digits in gid */
            if ( ttygid->gr_gid == 0 ) {
                gd_n = 1;
            }
            else {
                gd_n = 0;
                gd = ttygid->gr_gid;
                while ( gd ) {
                    gd /= 10;
                    gd_n++;
                }
            }

            /* length of gid string + mount options + terminator + padding */
            max_sz = gd_n + strlen(devpts_opts_base) + 1 + 16;
            if ( (devpts_opts=malloc(max_sz)) == NULL ) {
                    singularity_message(ERROR, "Memory allocation failure: %s\n", strerror(errno));
                    ABORT(255);
            }
            bzero(devpts_opts, max_sz);
            snprintf(devpts_opts, max_sz-1, "%s%d", devpts_opts_base, ttygid->gr_gid);

            singularity_message(DEBUG, "Mounting devpts for staged /dev/pts\n");
            if ( singularity_mount("devpts", joinpath(devdir, "/pts"), "devpts", MS_NOSUID|MS_NOEXEC, devpts_opts) < 0 ) {
                if (errno == EINVAL) {
                    // This is the error when unprivileged on RHEL7.4
                    singularity_message(VERBOSE, "Couldn't mount %s, continuing\n", joinpath(devdir, "/pts"));
                } else {
                    singularity_message(ERROR, "Failed to mount %s: %s\n", joinpath(devdir, "/pts"), strerror(errno));
                    ABORT(255);
                }
            } else {
                singularity_message(DEBUG, "Creating staged /dev/ptmx symlink\n");
                if ( symlink("/dev/pts/ptmx", joinpath(devdir, "/ptmx")) < 0 ) {
                    singularity_message(ERROR, "Failed to create /dev/ptmx symlink: %s\n", strerror(errno));
                    ABORT(255);
                }
            }
            free(devpts_opts);
        }

        singularity_message(DEBUG, "Mounting minimal staged /dev into container\n");
        if ( singularity_mount(devdir, joinpath(container_dir, "/dev"), NULL, MS_BIND|MS_REC, NULL) < 0 ) {
            singularity_priv_drop();
            singularity_message(WARNING, "Could not stage dev tree: '%s' -> '%s': %s\n", devdir, joinpath(container_dir, "/dev"), strerror(errno));
            free(sessiondir);
            free(devdir);
            return(-1);
        }
        singularity_priv_drop();

        free(sessiondir);
        free(devdir);

        return(0);
    }

    singularity_message(DEBUG, "Checking configuration file for 'mount dev'\n");
    if ( singularity_config_get_bool_char(MOUNT_DEV) > 0 ) {
        if ( is_dir(joinpath(container_dir, "/dev")) == 0 ) {
                singularity_priv_escalate();
                singularity_message(VERBOSE, "Bind mounting /dev\n");
                if ( singularity_mount("/dev", joinpath(container_dir, "/dev"), NULL, MS_BIND|MS_NOSUID|MS_REC, NULL) < 0 ) {
                    singularity_message(ERROR, "Could not bind mount container's /dev: %s\n", strerror(errno));
                    ABORT(255);
                }
                singularity_priv_drop();
        } else {
            singularity_message(WARNING, "Not mounting /dev, container has no bind directory\n");
        }
        return(0);
    }

    singularity_message(VERBOSE, "Not mounting /dev inside the container\n");

    return(0);
}


static int bind_dev(char *tmpdir, char *dev) {
    char *path = joinpath(tmpdir, dev);

    if ( ( is_chr(dev) == 0 ) || ( is_blk(dev) == 0 ) ) {
        if ( is_file(path) != 0 ) {
            int ret;
            singularity_message(VERBOSE2, "Creating bind point within container: %s\n", dev);

            singularity_priv_escalate();
            ret = fileput(path, "");
            singularity_priv_drop();

            if ( ret < 0 ) {
                singularity_message(WARNING, "Can not create %s: %s\n", dev, strerror(errno));
                return(-1);
            }
        }
    } else {
        singularity_message(WARNING, "Not setting up contained device: %s\n", dev);
        return(-1);
    }

    singularity_priv_escalate();
    singularity_message(DEBUG, "Mounting device %s at %s\n", dev, path);
    if ( singularity_mount(dev, path, NULL, MS_BIND, NULL) < 0 ) {
        singularity_priv_drop();
        singularity_message(WARNING, "Could not mount %s: %s\n", dev, strerror(errno));
        free(path);
        return(-1);
    }
    singularity_priv_drop();

    free(path);

    return(0);
}
