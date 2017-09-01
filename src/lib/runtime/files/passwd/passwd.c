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
#include <sys/stat.h>
#include <sys/types.h>
#include <limits.h>
#include <unistd.h>
#include <stdlib.h>
#include <grp.h>
#include <pwd.h>

#include "config.h"
#include "util/file.h"
#include "util/util.h"
#include "util/config_parser.h"
#include "util/message.h"
#include "util/privilege.h"
#include "util/registry.h"

#include "../file-bind.h"
#include "../../runtime.h"


int _singularity_runtime_files_passwd(void) {
    FILE *file_fp;
    char *source_file;
    char *tmp_file;
    char *homedir = singularity_priv_home();
    uid_t uid = singularity_priv_getuid();
    struct passwd *pwent = getpwuid(uid);
    char *containerdir = CONTAINER_FINALDIR;
    char *tmpdir = singularity_registry_get("SESSIONDIR");

    singularity_message(DEBUG, "Called singularity_file_passwd_create()\n");

    if ( uid == 0 ) {
        singularity_message(VERBOSE, "Not updating passwd file, running as root!\n");
        return(0);
    }

    if ( containerdir == NULL ) {
        singularity_message(ERROR, "Failed to obtain container directory\n");
        ABORT(255);
    }

    if ( tmpdir == NULL ) {
        singularity_message(ERROR, "Failed to obtain session directory\n");
        ABORT(255);
    }

    singularity_message(DEBUG, "Checking configuration option: 'config passwd'\n");
    if ( singularity_config_get_bool(CONFIG_PASSWD) <= 0 ) {
        singularity_message(VERBOSE, "Skipping bind of the host's /etc/passwd\n");
        return(0);
    }

    source_file = joinpath(containerdir, "/etc/passwd");
    tmp_file = joinpath(tmpdir, "/passwd");

    singularity_message(VERBOSE2, "Checking for template passwd file: %s\n", source_file);
    if ( is_file(source_file) < 0 ) {
        singularity_message(VERBOSE, "Passwd file does not exist in container, not updating\n");
        return(0);
    }

    singularity_message(VERBOSE2, "Creating template of /etc/passwd\n");
    if ( ( copy_file(source_file, tmp_file) ) < 0 ) {
        singularity_message(ERROR, "Failed copying template passwd file to tmpdir: %s\n", strerror(errno));
        ABORT(255);
    }

    singularity_message(VERBOSE, "Creating template passwd file and appending user data: %s\n", tmp_file);
    if ( ( file_fp = fopen(tmp_file, "a") ) == NULL ) { // Flawfinder: ignore
        singularity_message(ERROR, "Could not open template passwd file %s: %s\n", tmp_file, strerror(errno));
        ABORT(255);
    }

    fprintf(file_fp, "%s:x:%d:%d:%s:%s:%s\n", pwent->pw_name, pwent->pw_uid, pwent->pw_gid, pwent->pw_gecos, homedir, pwent->pw_shell);
    fclose(file_fp);


    container_file_bind(tmp_file, "/etc/passwd");

    // set HOME to the homedir, because it might be different than outside
    envar_set("HOME", homedir, 1);

    return(0);
}
