/* 
 * Copyright (c) 2015-2016, Gregory M. Kurtzer. All rights reserved.
 * 
 * “Singularity” Copyright (c) 2016, The Regents of the University of California,
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


#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/mount.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/file.h>
#include <errno.h> 
#include <string.h>
#include <fcntl.h>
#include <libgen.h>
#include <limits.h>
#include <pwd.h>


#include "config.h"
#include "mounts.h"
#include "file.h"
#include "util.h"
#include "loop-control.h"
#include "message.h"
#include "config_parser.h"

#ifndef MS_REC
#define MS_REC 16384
#endif


int mount_image(char * loop_device, char * mount_point, int writable) {

    message(DEBUG, "Called mount_image(%s, %s, %d)\n", loop_device, mount_point, writable);

    message(DEBUG, "Checking mount point is present\n");
    if ( is_dir(mount_point) < 0 ) {
        message(ERROR, "Mount point is not available: %s\n", mount_point);
        ABORT(255);
    }

    message(DEBUG, "Checking loop is a block device\n");
    if ( is_blk(loop_device) < 0 ) {
        message(ERROR, "Loop device is not a block dev: %s\n", loop_device);
        ABORT(255);
    }

    if ( writable > 0 ) {
        message(DEBUG, "Trying to mount read/write as ext4 with discard option\n");
        if ( mount(loop_device, mount_point, "ext4", MS_NOSUID, "discard") < 0 ) {
            message(DEBUG, "Trying to mount read/write as ext4 without discard option\n");
            if ( mount(loop_device, mount_point, "ext4", MS_NOSUID, "") < 0 ) {
                message(DEBUG, "Trying to mount read/write as ext3\n");
                if ( mount(loop_device, mount_point, "ext3", MS_NOSUID, "") < 0 ) {
                    message(ERROR, "Failed to mount (rw) '%s' at '%s': %s\n", loop_device, mount_point, strerror(errno));
                    ABORT(255);
                }
            }
        }
    } else {
        message(DEBUG, "Trying to mount read only as ext4 with discard option\n");
        if ( mount(loop_device, mount_point, "ext4", MS_NOSUID|MS_RDONLY, "discard") < 0 ) {
            message(DEBUG, "Trying to mount read only as ext4 without discard option\n");
            if ( mount(loop_device, mount_point, "ext4", MS_NOSUID|MS_RDONLY, "") < 0 ) {
                message(DEBUG, "Trying to mount read only as ext3\n");
                if ( mount(loop_device, mount_point, "ext3", MS_NOSUID|MS_RDONLY, "") < 0 ) {
                    message(ERROR, "Failed to mount (ro) '%s' at '%s': %s\n", loop_device, mount_point, strerror(errno));
                    ABORT(255);
                }
            }
        }
    }

    message(DEBUG, "Returning mount_image(%s, %s, %d) = 0\n", loop_device, mount_point, writable);

    return(0);
}

static int create_bind_dir(const char *dest_orig, const char *tmp_dir, int isfile) {
    char *dest = strdup(dest_orig);
    if ( !dest ) {
        message(ERROR, "Failed to allocate memory for destination strings.\n");
        ABORT(255)
    }
    char *last_slash = dest + strlen(dest) - 1;
    while ( last_slash > dest ) {
        if ( *last_slash != '/' ) {break;}
        *last_slash = '\0';
        last_slash--;
    }
    message(DEBUG, "Calling create_bind_dir(%s, %s, %d)\n", dest, tmp_dir, isfile);
    
    char *dest_copy = strdup(dest);
    if ( !dest_copy ) {
        message(ERROR, "Failed to allocate memory for destination strings.\n");
        ABORT(255)
    }
    last_slash = strrchr(dest_copy, '/');
    if (last_slash == NULL) {
        message(ERROR, "Ran out of '/' prefixes\n");
        ABORT(255);
    }
    *last_slash = '\0';
    if ( !is_dir(dest_copy) ) {
        // Parent directory exists; create a temporary directory inside tmp_dir,
        // bind mount the temporary directory on top of the existing parent dir.  Note
        // this has the unfortunate side-effect of squashing any othe files inside this
        // parent directory.
        char new_tmp_dir[PATH_MAX];
        if ( snprintf(new_tmp_dir, PATH_MAX, "%s/bind_bootstrap_XXXXXX", tmp_dir) >= PATH_MAX) {
            message(ERROR, "Overly long temporary pathname: %s\n", tmp_dir);
            return 1;
        }
        if (mkdtemp(new_tmp_dir) == NULL) {
            message(ERROR, "Failed to create new temporary directory %s: %s\n", new_tmp_dir, strerror(errno));
            return 1; 
        }
        if ( chmod(new_tmp_dir, 0755) ) {
            message(ERROR, "Failed to chmod temporary directory %s: %s\n", new_tmp_dir, strerror(errno));
            return 1;
        }
        int new_len = PATH_MAX - strlen(new_tmp_dir) - 1;
        if (snprintf(new_tmp_dir + strlen(new_tmp_dir), new_len, "/%s", last_slash + 1) >= new_len) {
            message(ERROR, "Overly long path name in temp dir: %s/%s\n", new_tmp_dir, last_slash + 1);
            return 1;
        }
        if ( mkdir(new_tmp_dir, 0755) == -1 ) {
            message(ERROR, "Failed to create new directory %s inside temp: %s", new_tmp_dir, strerror(errno));
            return 1;
        }
        *strrchr(new_tmp_dir, '/') = '\0';
        if ( mount(new_tmp_dir, dest_copy, NULL, MS_BIND|MS_NOSUID|MS_REC, NULL) < 0 ) {
            message(ERROR, "When creating temp directory, Could not bind %s: %s\n", dest, strerror(errno));
            return 1;
        }
        message(DEBUG, "Created top-level graft directory: %s\n", new_tmp_dir);
    } else {
        struct stat filestat;
        if ( stat(dest_copy, &filestat) == 0 ) {
            // Not a directory, but path exists (file or other)
            message(ERROR, "Cannot create bind directory as %s in path already exists.\n", dest_copy);
            return 1;
        }
        if ( create_bind_dir(dest_copy, tmp_dir, 0) ) {
            return 1;
        }
        // Now we know the parent path exists.
        if (isfile) {
            int fd;
            if ( -1 == ( fd = open(dest, O_CREAT|O_RDWR|O_CLOEXEC|O_EXCL, 0600)  ) ) {
                message(ERROR, "Failed to create stub file %s: %s\n", dest, strerror(errno));
                return 1;
            }
            close(fd);
        } else {
            if ( -1 == ( mkdir(dest, 0755) ) ) {
                message(ERROR, "Failed to make top-level stub directory %s: %s\n", dest, strerror(errno));
                return 1;
            }
        }
    }
    free(dest);
    free(dest_copy);
    return 0;
}

void mount_bind(char * source, char * dest, int writable, const char *tmp_dir) {

    message(DEBUG, "Called mount_bind(%s, %d, %d, %s)\n", source, dest, writable, tmp_dir);

    message(DEBUG, "Checking that source exists and is a file or directory\n");
    if ( is_dir(source) != 0 && is_file(source) != 0 ) {
        message(ERROR, "Bind source path is not a file or directory: '%s'\n", source);
        ABORT(255);
    }

    message(DEBUG, "Checking that destination exists and is a file or directory\n");
    if ( is_dir(dest) != 0 && is_file(dest) != 0 ) {
        if ( create_bind_dir(dest, tmp_dir, is_dir(source)) != 0 ) {
            message(ERROR, "Container bind path is not a file or directory: '%s'\n", dest);
            ABORT(255);
        }
    }

    message(DEBUG, "Calling mount(%s, %s, ...)\n", source, dest);
    if ( mount(source, dest, NULL, MS_BIND|MS_NOSUID|MS_REC, NULL) < 0 ) {
        message(ERROR, "Could not bind %s: %s\n", dest, strerror(errno));
        ABORT(255);
    }

    if ( writable <= 0 ) {
        message(VERBOSE2, "Making mount read only: %s\n", dest);
        if ( mount(NULL, dest, NULL, MS_BIND|MS_REC|MS_REMOUNT|MS_RDONLY, NULL) < 0 ) {
            message(ERROR, "Could not bind read only %s: %s\n", dest, strerror(errno));
            ABORT(255);
        }
    }

}


void mount_home(char *rootpath) {
    char *homedir;
    char *homedir_base;
    struct passwd *pw;

    // TODO: Functionize this
    pw = getpwuid(getuid());

    message(DEBUG, "Obtaining user's homedir\n");
    homedir = pw->pw_dir;

    if ( ( homedir_base = container_basedir(rootpath, homedir) ) != NULL ) {
        if ( is_dir(homedir_base) == 0 ) {
            if ( is_dir(joinpath(rootpath, homedir_base)) == 0 ) {
                message(VERBOSE, "Mounting home directory base path: %s\n", homedir_base);
                if ( mount(homedir_base, joinpath(rootpath, homedir_base), NULL, MS_BIND|MS_NOSUID|MS_REC, NULL) < 0 ) {
                    ABORT(255);
                }
            } else {
                message(WARNING, "Container bind point does not exist: '%s' (homedir_base)\n", homedir_base);
            }
        } else {
            message(WARNING, "Home directory base source path does not exist: %s\n", homedir_base);
        }
    }

}


void bind_paths(char *rootpath) {
    char *tmp_config_string;
    message(DEBUG, "Checking configuration file for 'bind path'\n");
    config_rewind();
    while ( ( tmp_config_string = config_get_key_value("bind path") ) != NULL ) {
        char *source = strtok(tmp_config_string, ",");
        char *dest = strtok(NULL, ",");
        chomp(source);
        if ( dest == NULL ) {
            dest = strdup(source);
        } else {
            if ( dest[0] == ' ' ) {
                dest++;
            }
            chomp(dest);
        }

        message(VERBOSE2, "Found 'bind path' = %s, %s\n", source, dest);

// TODO: Make sure this isn't already mounted
//        if ( ( homedir_base != NULL ) && ( strncmp(dest, homedir_base, strlength(homedir_base, 256)) == 0 )) {
//            // Skipping path as it was already mounted as homedir_base
//            message(VERBOSE2, "Skipping '%s' as it is part of home path and already mounted\n", dest);
//            continue;
//        }

        if ( ( is_file(source) != 0 ) && ( is_dir(source) != 0 ) ) {
            message(WARNING, "Non existant 'bind path' source: '%s'\n", source);
            continue;
        }
        if ( ( is_file(joinpath(rootpath, dest)) != 0 ) && ( is_dir(joinpath(rootpath, dest)) != 0 ) ) {
            message(WARNING, "Non existant 'bind point' in container: '%s'\n", dest);
            continue;
        }

        message(VERBOSE, "Binding '%s' to '%s/%s'\n", source, rootpath, dest);
        if ( mount(source, joinpath(rootpath, dest), NULL, MS_BIND|MS_NOSUID|MS_REC, NULL) < 0 ) {
            ABORT(255);
        }
//        message(VERBOSE2, "Making mount read only: %s\n", dest);
//        if ( mount(NULL, dest, NULL, MS_BIND|MS_REC|MS_REMOUNT|MS_RDONLY, NULL) < 0 ) {
//            message(ERROR, "Could not bind read only %s: %s\n", dest, strerror(errno));
//            ABORT(255);
//        }
    }

}


