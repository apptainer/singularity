/* 
 * Copyright (c) 2017-2018, SyLabs, Inc. All rights reserved.
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

#define _GNU_SOURCE 1

#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/types.h>
#include <errno.h>
#include <string.h>
#include <fcntl.h>

#include "util/util.h"
#include "util/file.h"
#include "util/registry.h"
#include "lib/image/image.h"
#include "util/suid.h"
#include "util/sessiondir.h"
#include "util/cleanupd.h"
#include "lib/runtime/runtime.h"
#include "util/config_parser.h"
#include "util/privilege.h"

#include "action-lib/include.h"

#include "slurm/spank.h"

SPANK_PLUGIN(singularity, 1);

#ifndef SYSCONFDIR
#define SYSCONFDIR "/etc"
#endif

#define INT_MAX_STRING_SIZE 30

// These should only be set post-fork and before exec;
// this means reading / writing to these should be thread-safe.
static char job_uid_str[INT_MAX_STRING_SIZE]; //Flawfinder: ignore
static char job_gid_str[INT_MAX_STRING_SIZE]; //Flawfinder: ignore
static char *job_image = NULL;
static char *job_bindpath = NULL;

static int setup_container_environment(spank_t spank)
{

    uid_t job_uid = -1;
    gid_t job_gid = -1;
    char *job_cwd = NULL;
    char *bindpath = NULL;
    int argc;
    char **argv = NULL;
    int i;

     setenv("SINGULARITY_MESSAGELEVEL", "1", 0); //Don't overwrite if exists

    if (ESPANK_SUCCESS != spank_get_item(spank, S_JOB_UID, &job_uid)) {
        slurm_error("spank/%s: Failed to get job's target UID", plugin_name);
        return -1;
    }
    if (INT_MAX_STRING_SIZE <=
        snprintf(job_uid_str, INT_MAX_STRING_SIZE, "%u", job_uid)) { // Flawfinder: ignore
        slurm_error("spank/%s: Failed to serialize job's UID to string",
                    plugin_name);
        return -1;
    }
    if (setenv("SINGULARITY_TARGET_UID", job_uid_str, 1) < 0) {
        slurm_error("spank/%s: Failed to setenv(\"SINGULARITY_TARGET_UID\")",
                    plugin_name);
        return -1;
    }

    if (ESPANK_SUCCESS != spank_get_item(spank, S_JOB_GID, &job_gid)) {
        slurm_error("spank/%s: Failed to get job's target GID", plugin_name);
        return -1;
    }
    if (INT_MAX_STRING_SIZE <=
        snprintf(job_gid_str, INT_MAX_STRING_SIZE, "%u", job_gid)) { // Flawfinder: ignore
        slurm_error("spank/%s: Failed to serialize job's GID to string",
                    plugin_name);
        return -1;
    }
    if (setenv("SINGULARITY_TARGET_GID", job_gid_str, 1) < 0) {
        slurm_error("spank/%s: Failed to setenv(\"SINGULARITY_TARGET_GID\")",
                    plugin_name);
        return -1;
    }

    job_cwd = get_current_dir_name();
    if (!job_cwd) {
        slurm_error("spank/%s: Failed to determine job's correct PWD: %s",
                    plugin_name, strerror(errno));
        return -1;
    }
    if (setenv("SINGULARITY_TARGET_PWD", job_cwd, 1) < 0) {
        slurm_error("spank/%s: Failed to setenv(\"SINGULARITY_TARGET_PWD\")",
                    plugin_name);
        return -1;
    }
    /* setenv() makes a copy */
    free(job_cwd);

    if (!job_image) {
        slurm_error("spank/%s: Unable to determine job's image file.",
                    plugin_name);
        return -1;
    }
    if (setenv("SINGULARITY_IMAGE", job_image, 1) < 0) {
        slurm_error("spank/%s: Failed to setenv(\"SINGULARITY_IMAGE\")",
                    plugin_name);
        return -1;
    }

    if ((job_bindpath) &&
        (setenv("SINGULARITY_BINDPATH", job_bindpath, 1) < 0)) {
        slurm_error("spank/%s: Failed to setenv(\"SINGULARITY_BINDPATH\")",
                    plugin_name);
        return -1;
    }

    return 0;
}

static int setup_container_cwd() {
    singularity_message(DEBUG, "Trying to change directory to where we started\n");
    char *target_pwd = singularity_registry_get("TARGET_PWD");

    if (!target_pwd || (chdir(target_pwd) < 0)) {
        singularity_message(ERROR, "Failed to change into correct directory "
                            "(%s) inside container.",
                            target_pwd ? target_pwd : "UNKNOWN");
        return -1;
    }
    free(target_pwd);
    return 0;
}

static int setup_container(spank_t spank)
{
    int rc;
    struct image_object image;
    char *command = NULL;

    if ((rc = setup_container_environment(spank)) != 0) { return rc; }

    /*
     * Ugg, singularity_* calls tend to call ABORT(255), which translates to
     * exit(255), all over the place.  The slurm SPANK hook API may not
     * expect such sudden death of the pending slurm task.  I've left
     * a bunch of following "return rc;" commented out, as the failure
     * conditions from singularity_* calls isn't clear to me.
     */

    // Before we do anything, check privileges and drop permission
    singularity_priv_init();
    singularity_priv_drop();

    singularity_message(VERBOSE, "Running SLURM/Singularity integration "
                        "plugin\n");

    if ((rc = singularity_config_init()) != 0) {
         return rc;
    }


    singularity_priv_init();
//TODO    singularity_suid_init(argv);

    singularity_registry_init();
    singularity_priv_userns();
    singularity_priv_drop();

    singularity_cleanupd();

    singularity_runtime_ns(SR_NS_ALL);

    singularity_sessiondir();

    image = singularity_image_init(singularity_registry_get("IMAGE")); 

    if ( singularity_registry_get("WRITABLE") == NULL ) {
        singularity_image_open(&image, O_RDONLY);
    } else {
        singularity_image_open(&image, O_RDWR);
    }  

    singularity_image_check(&image);
    singularity_image_bind(&image);
    singularity_image_mount(&image, singularity_runtime_rootfs(NULL));

    action_ready(singularity_runtime_rootfs(NULL));

    singularity_runtime_overlayfs();
    singularity_runtime_mounts();
    singularity_runtime_files();
    singularity_runtime_enter();

    singularity_runtime_environment();

    singularity_priv_drop_perm();

 
    if ((rc = setup_container_cwd()) < 0) { 
       singularity_message(ERROR, "Could not obtain current directory.\n");
       return rc; 
    }

    envar_set("SINGULARITY_CONTAINER", singularity_image_name(&image), 1); // Legacy PS1 support
    envar_set("SINGULARITY_NAME", singularity_image_name(&image), 1);
    envar_set("SINGULARITY_SHELL", singularity_registry_get("SHELL"), 1);

    command = singularity_registry_get("COMMAND");
    singularity_message(LOG, "USER=%s, IMAGE='%s', COMMAND='%s'\n", singularity_priv_getuser(), singularity_image_name(&image), singularity_registry_get("COMMAND"));

    // At this point, the current process is in the runtime container environment.
    // Return control flow back to SLURM: when execv is invoked, it'll be done from
    // within the container.

    return 0;
}

// TODO: When run on the submit host, we should evaluate the URL and create/cache docker images as necessary.
static int determine_image(int val, const char *optarg, int remote)
{
    if (val) {}  // Suppresses unused error...
    // TODO: could do some basic path validation here in order to prevent an ABORT() later.
    job_image = strdup(optarg);

    return job_image == NULL;
}

static int determine_bind(int val, const char *optarg, int remote)
{
    if (!job_bindpath) {
        job_bindpath = strdup(optarg);
    }
    return job_bindpath == NULL;
}

/// SPANK plugin functions.

int slurm_spank_init(spank_t spank, int ac, char **av)
{
    int i;
    struct spank_option image_opt,
                        bind_opt;

    memset(&image_opt, '\0', sizeof(image_opt));
    image_opt.name = "singularity-image";
    image_opt.arginfo = "[path]";
    image_opt.usage = "Specify a path to a Singularity image, directory tree, "
                      "or Docker image";
    image_opt.has_arg = 1;
    image_opt.val = 0;
    image_opt.cb = determine_image;
    if (ESPANK_SUCCESS != spank_option_register(spank, &image_opt)) {
        slurm_error("spank/%s: Unable to register a new option.",
                    plugin_name);
        return -1;
    }

    memset(&bind_opt, '\0', sizeof(bind_opt));
    bind_opt.name = "singularity-bind";
    bind_opt.arginfo = "[path || src:dest],...";
    bind_opt.usage = "Specify a user-bind path specification.  Can either be "
                     "a path or a src:dest pair, specifying the bind mount to "
                     "perform";
    bind_opt.has_arg = 1;
    bind_opt.val = 0;
    bind_opt.cb = determine_bind;
    if (ESPANK_SUCCESS != spank_option_register(spank, &bind_opt)) {
        slurm_error("spank/%s: Unable to register a new option.",
                    plugin_name);
        return -1;
    }

    // Make this a no-op except when starting the task.
    if (spank_context() == S_CTX_ALLOCATOR || (spank_remote(spank) != 1)) {
        return 0;
    }

    for (i = 0; i < ac; i++) {
        if (strncmp ("default_image=", av[i], 14) == 0) {
            const char *optarg = av[i] + 14;
            job_image = strdup(optarg);
        } else {
            slurm_error ("spank/%s: Invalid option: %s", av[i], plugin_name);
        }
    }

    return 0;
}

int slurm_spank_task_init_privileged(spank_t spank, int ac, char *argv[])
{
    if (job_image) {
        return setup_container(spank);
    }
    return 0;
}
