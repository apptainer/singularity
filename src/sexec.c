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

#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/file.h>
#include <sys/mount.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <sys/param.h>
#ifdef SINGULARITY_NO_NEW_PRIVS
#include <sys/prctl.h>
#include <ctype.h>
#endif
#include <errno.h> 
#include <signal.h>
#include <sched.h>
#include <string.h>
#include <fcntl.h>  
#include <grp.h>
#include <libgen.h>
#include <pwd.h>

#include "config.h"
#include "mounts.h"
#include "loop-control.h"
#include "util.h"
#include "file.h"
#include "container_files.h"
#include "config_parser.h"
#include "container_actions.h"
#include "privilege.h"
#include "message.h"
#include "namespaces.h"


#ifndef LOCALSTATEDIR
#define LOCALSTATEDIR "/etc"
#endif

#ifndef SYSCONFDIR
#define SYSCONFDIR "/etc"
#endif

#ifndef MS_PRIVATE
#define MS_PRIVATE (1<<18)
#endif
#ifndef MS_REC
#define MS_REC 16384
#endif

pid_t exec_fork_pid = 0;

// TODO: This is broke, and needs some love!
void sighandler(int sig) {
    signal(sig, sighandler);

    if ( exec_fork_pid > 0 ) {
        fprintf(stderr, "Singularity is sending SIGKILL to child pid: %d\n", exec_fork_pid);

        kill(exec_fork_pid, SIGKILL);
    }
}



int main(int argc, char ** argv) {
    FILE *loop_fp = NULL;
    FILE *containerimage_fp = NULL;
    FILE *daemon_fp = NULL;
    char *containerimage;
    char *containername;
    char *containerdir;
    char *command;
    char *scratch_dir = NULL;
    char *sessiondir;
    char *sessiondir_prefix;
    char *loop_dev_lock = NULL;
    char *loop_dev_cache = NULL;
    char *loop_dev = 0;
    char *config_path;
    char cwd[PATH_MAX]; // Flawfinder: ignore
    int cwd_fd = 0;
    int sessiondirlock_fd = 0;
    int containerimage_fd = 0;
    int loop_dev_lock_fd = 0;
    int daemon_pid = -1;
    int retval = 0;
    uid_t uid;
    pid_t namespace_fork_pid = 0;
    int container_is_image = -1;
    int container_is_dir = -1;
    mode_t process_mask = umask(0); // Flawfinder: ignore (we must reset umask to ensure appropriate permissions)



//****************************************************************************//
// Init
//****************************************************************************//

    signal(SIGINT, sighandler);
    signal(SIGQUIT, sighandler);
    signal(SIGTERM, sighandler);
    signal(SIGKILL, sighandler);

    // Get all user/group info
    uid = getuid();

    message(VERBOSE3, "Initalizing privilege cache.\n");
    priv_init();

    message(VERBOSE3, "Checking if we can escalate privileges properly.\n");
    priv_escalate();

    message(VERBOSE3, "Setting privileges back to calling user\n");
    priv_drop();

    // Figure out where we start
    message(DEBUG, "Obtaining file descriptor to current directory\n");
    if ( (cwd_fd = open(".", O_RDONLY)) < 0 ) { // Flawfinder: ignore (need current directory FD)
        message(ERROR, "Could not open cwd fd (%s)!\n", strerror(errno));
        ABORT(1);
    }
    message(DEBUG, "Getting current working directory path string\n");
    if ( getcwd(cwd, PATH_MAX) == NULL ) {
        message(ERROR, "Could not obtain current directory path: %s\n", strerror(errno));
        ABORT(1);
    }

    message(DEBUG, "Obtaining SINGULARITY_COMMAND from environment\n");
    if ( ( command = getenv("SINGULARITY_COMMAND") ) == NULL ) { // Flawfinder: ignore (we need the command, and check exact match below)
        message(ERROR, "SINGULARITY_COMMAND undefined!\n");
        ABORT(1);
    }
    unsetenv("SINGULARITY_COMMAND");

    message(DEBUG, "Obtaining SINGULARITY_IMAGE from environment\n");
    if ( ( containerimage = getenv("SINGULARITY_IMAGE") ) == NULL ) { // Flawfinder: ignore (we need the image name, and open it as the calling user)
        message(ERROR, "SINGULARITY_IMAGE undefined!\n");
        ABORT(1);
    }

    message(DEBUG, "Checking container image is a file: %s\n", containerimage);
    if ( is_file(containerimage) == 0 ) {
        message(DEBUG, "Container is a file\n");
        container_is_image = 1;
    } else if ( is_dir(containerimage) == 0 ) {
#ifdef SINGULARITY_NO_NEW_PRIVS
        message(DEBUG, "Container is a directory\n");
        if ( strcmp(containerimage, "/") == 0 ) {
            message(ERROR, "Bad user... I have notified the powers that be.\n");
            message(LOG, "User ID '%d' requested '/' as the container!\n", getuid());
            ABORT(1);
        }
        container_is_dir = 1;
#else
        message(ERROR, "This build of Singularity does not support container directories\n");
        ABORT(1);
#endif
    } else {
        message(ERROR, "Container image path is invalid: %s\n", containerimage);
        ABORT(1);
    }

    message(DEBUG, "Building configuration file location\n");
    config_path = (char *) malloc(strlength(SYSCONFDIR, 128) + 30);
    snprintf(config_path, strlen(SYSCONFDIR) + 30, "%s/singularity/singularity.conf", SYSCONFDIR); // Flawfinder: ignore
    message(DEBUG, "Config location: %s\n", config_path);

    message(DEBUG, "Checking Singularity configuration is a file: %s\n", config_path);
    if ( is_file(config_path) != 0 ) {
        message(ERROR, "Configuration file not found: %s\n", config_path);
        ABORT(255);
    }

    message(DEBUG, "Checking Singularity configuration file is owned by root\n");
    if ( is_owner(config_path, 0) != 0 ) {
        message(ERROR, "Configuration file is not owned by root: %s\n", config_path);
        ABORT(255);
    }

    message(DEBUG, "Opening Singularity configuration file\n");
    if ( config_open(config_path) < 0 ) {
        ABORT(255);
    }

    // TODO: Offer option to only run containers owned by root (so root can approve
    // containers)
//    if ( uid == 0 && is_owner(containerimage, 0) < 0 ) {
//        message(ERROR, "Root should only run containers that root owns!\n");
//        ABORT(1);
//    }

    message(DEBUG, "Checking Singularity configuration for 'sessiondir prefix'\n");
    config_rewind();
    if ( ( sessiondir_prefix = config_get_key_value("sessiondir prefix") ) != NULL ) {
        sessiondir = strjoin(sessiondir_prefix, file_id(containerimage));
    } else {
        sessiondir = strjoin("/tmp/.singularity-session-", file_id(containerimage));
    }
    message(DEBUG, "Set sessiondir to: %s\n", sessiondir);

    
    containername = basename(strdup(containerimage));
    message(DEBUG, "Set containername to: %s\n", containername);

    config_rewind();
    if ( ( containerdir = config_get_key_value("container dir") ) == NULL ) {
        containerdir = strdup("/var/singularity/mnt");
    }
    message(DEBUG, "Set image mount path to: %s\n", containerdir);

    message(LOG, "Command=%s, Container=%s, CWD=%s, Arg1=%s\n", command, containerimage, cwd, argv[1]);

    if (container_is_image > 0 ) {
        message(DEBUG, "Checking if we are opening image as read/write\n");
        if ( getenv("SINGULARITY_WRITABLE") == NULL ) { // Flawfinder: ignore (only checking for existance of getenv)
            message(DEBUG, "Opening image as read only: %s\n", containerimage);
            if ( ( containerimage_fp = fopen(containerimage, "r") ) == NULL ) { // Flawfinder: ignore 
                message(ERROR, "Could not open image read only %s: %s\n", containerimage, strerror(errno));
                ABORT(255);
            }

            containerimage_fd = fileno(containerimage_fp);
            message(DEBUG, "Setting shared lock on file descriptor: %d\n", containerimage_fd);
            if ( flock(containerimage_fd, LOCK_SH | LOCK_NB) < 0 ) {
                message(ERROR, "Could not obtained shared lock on image\n");
                ABORT(5);
            }
        } else {
            message(DEBUG, "Opening image as read/write: %s\n", containerimage);
            if ( ( containerimage_fp = fopen(containerimage, "r+") ) == NULL ) { // Flawfinder: ignore
                message(ERROR, "Could not open image read/write %s: %s\n", containerimage, strerror(errno));
                ABORT(255);
            }

            containerimage_fd = fileno(containerimage_fp);
            message(DEBUG, "Setting exclusive lock on file descriptor: %d\n", containerimage_fd);
            if ( flock(containerimage_fd, LOCK_EX | LOCK_NB) < 0 ) {
                message(ERROR, "Could not obtained exclusive lock on image\n");
                ABORT(5);
            }
        }
    }

    message(DEBUG, "Checking for namespace daemon pidfile\n");
    if ( is_file(joinpath(sessiondir, "daemon.pid")) == 0 ) {
        FILE *test_daemon_fp;

        if ( ( test_daemon_fp = fopen(joinpath(sessiondir, "daemon.pid"), "r") ) == NULL ) { // Flawfinder: ignore
            message(ERROR, "Could not open daemon pid file %s: %s\n", joinpath(sessiondir, "daemon.pid"), strerror(errno));
            ABORT(255);
        }

        message(DEBUG, "Checking if namespace daemon is running\n");
        if ( flock(fileno(test_daemon_fp), LOCK_SH | LOCK_NB) != 0 ) {
            if ( fscanf(test_daemon_fp, "%d", &daemon_pid) <= 0 ) {
                message(ERROR, "Could not read daemon process ID\n");
                ABORT(255);
            }

        } else {
            message(WARNING, "Singularity namespace daemon pid exists, but daemon not alive?\n");
        }
        fclose(test_daemon_fp);
    }

    // Create temporary scratch directories for use inside the chroot.
    // We do this as the user, but will later bind-mount as root.
    config_rewind();
    int user_scratch = 0;
    user_scratch = getenv("SINGULARITY_USER_SCRATCH") != NULL;
    // USER_SCRATCH is only allowed in the case of NO_NEW_PRIVS.
    if ( user_scratch && !config_get_key_bool("allow user scratch", 1) ) {
        message(ERROR, "The sysadmin has disabled support for user-specified scratch directories.\n");
        ABORT(255);
    }
    config_rewind();
#ifndef SINGULARITY_NO_NEW_PRIVS
    // NOTE: we allow 'bind scratch' without NO_NEW_PRIVS as that is setup by
    // the sysadmin; however, we don't allow user-specified scratch!
    if ( user_scratch ) {
        message(ERROR, "User-specified scratch directories requested, but support was not compiled in!\n");
        ABORT(255);
    }
#endif
    if ( ( config_get_key_value("bind scratch") != NULL ) || user_scratch ) {
        message(DEBUG, "Creating a scratch directory for this container.\n");
        config_rewind();
        char *tmp_config_string = config_get_key_value("scratch dir");
        tmp_config_string = tmp_config_string ? tmp_config_string : getenv("_CONDOR_SCRATCH_DIR");
        tmp_config_string = tmp_config_string ? tmp_config_string : getenv("TMPDIR");
        tmp_config_string = tmp_config_string ? tmp_config_string : "/tmp";
        char tmp_path[PATH_MAX];
        if ( snprintf(tmp_path, PATH_MAX, "%s/.singularity-scratchdir.XXXXXX", tmp_config_string) >= PATH_MAX ) {
            message(ERROR, "Overly-long pathname for scratch directory: %s\n", tmp_config_string);
            ABORT(255);
        }
        if ( ( scratch_dir = strdup(tmp_path) ) == NULL ) {
            message(ERROR, "Memory allocation failure when creating scratch directory: %s\n", strerror(errno));
            ABORT(255);
        }
        if ( ( scratch_dir = mkdtemp(scratch_dir) ) == NULL ) {
            message(ERROR, "Creation of temproary scratch directory %s failed: %s\n", scratch_dir, strerror(errno));
            ABORT(255);
        }
        message(DEBUG, "Using scratch directory '%s'\n", scratch_dir);
    }

//****************************************************************************//
// We are now running with escalated privileges until we exec
//****************************************************************************//

    message(VERBOSE3, "Entering privileged runtime\n");
    priv_escalate();

    message(VERBOSE, "Creating/Verifying session directory: %s\n", sessiondir);
    if ( s_mkpath(sessiondir, 0755) < 0 ) {
        message(ERROR, "Failed creating session directory: %s\n", sessiondir);
        ABORT(255);
    }
    if ( is_dir(sessiondir) < 0 ) {
        message(ERROR, "Temporary directory does not exist %s: %s\n", sessiondir, strerror(errno));
        ABORT(255);
    }
    if ( is_owner(sessiondir, 0) < 0 ) {
        message(ERROR, "Container working directory has wrong ownership: %s\n", sessiondir);
        ABORT(255);
    }

    message(DEBUG, "Opening sessiondir file descriptor\n");
    if ( ( sessiondirlock_fd = open(sessiondir, O_RDONLY) ) < 0 ) { // Flawfinder: ignore
        message(ERROR, "Could not obtain file descriptor on %s: %s\n", sessiondir, strerror(errno));
        ABORT(255);
    }
    message(DEBUG, "Setting shared flock() on session directory\n");
    if ( flock(sessiondirlock_fd, LOCK_SH | LOCK_NB) < 0 ) {
        message(ERROR, "Could not obtain shared lock on %s: %s\n", sessiondir, strerror(errno));
        ABORT(255);
    }

    message(DEBUG, "Caching info into sessiondir\n");
    if ( fileput(joinpath(sessiondir, "image"), containername) < 0 ) {
        message(ERROR, "Could not write container name to %s\n", joinpath(sessiondir, "image"));
        ABORT(255);
    }

    if ( container_is_image > 0 ) {
        message(DEBUG, "Checking for set loop device\n");
        loop_dev_lock = joinpath(sessiondir, "loop_dev.lock");
        loop_dev_cache = joinpath(sessiondir, "loop_dev");
        if ( ( loop_dev_lock_fd = open(loop_dev_lock, O_CREAT | O_RDWR, 0644) ) < 0 ) { // Flawfinder: ignore
            message(ERROR, "Could not open loop_dev_lock %s: %s\n", loop_dev_lock, strerror(errno));
            ABORT(255);
        }

        message(DEBUG, "Requesting exclusive flock() on loop_dev lockfile\n");
        if ( flock(loop_dev_lock_fd, LOCK_EX | LOCK_NB) == 0 ) {
            message(DEBUG, "We have exclusive flock() on loop_dev lockfile\n");

            message(DEBUG, "Binding container to loop interface\n");
            if ( ( loop_fp = loop_bind(containerimage_fp, &loop_dev, 1) ) == NULL ) {
                message(ERROR, "Could not bind image to loop!\n");
                ABORT(255);
            }

            message(DEBUG, "Writing loop device name to loop_dev: %s\n", loop_dev);
            if ( fileput(loop_dev_cache, loop_dev) < 0 ) {
                message(ERROR, "Could not write to loop_dev_cache %s: %s\n", loop_dev_cache, strerror(errno));
                ABORT(255);
            }

            message(DEBUG, "Resetting exclusive flock() to shared on loop_dev lockfile\n");
            flock(loop_dev_lock_fd, LOCK_SH | LOCK_NB);

        } else {
            message(DEBUG, "Unable to get exclusive flock() on loop_dev lockfile\n");

            message(DEBUG, "Waiting to obtain shared lock on loop_dev lockfile\n");
            flock(loop_dev_lock_fd, LOCK_SH);

            message(DEBUG, "Exclusive lock on loop_dev lockfile released, getting loop_dev\n");
            if ( ( loop_dev = filecat(loop_dev_cache) ) == NULL ) {
                message(ERROR, "Could not retrieve loop_dev_cache from %s\n", loop_dev_cache);
                ABORT(255);
            }

            message(DEBUG, "Attaching loop file pointer to loop_dev\n");
            if ( ( loop_fp = loop_attach(loop_dev) ) == NULL ) {
                message(ERROR, "Could not obtain file pointer to loop device!\n");
                ABORT(255);
            }
        }

    }

    message(DEBUG, "Creating container image mount path: %s\n", containerdir);
    if ( s_mkpath(containerdir, 0755) < 0 ) {
        message(ERROR, "Failed creating image directory %s\n", containerdir);
        ABORT(255);
    }
    if ( is_owner(containerdir, 0) < 0 ) {
        message(ERROR, "Container directory is not root owned: %s\n", containerdir);
        ABORT(255);
    }



    // Manage the daemon bits early
    if ( strcmp(command, "start") == 0 ) {
#ifdef NO_SETNS
        message(ERROR, "This host does not support joining existing name spaces\n");
        ABORT(1);
#else
        int daemon_fd;

        message(DEBUG, "Namespace daemon function requested\n");

        message(DEBUG, "Creating namespace daemon pidfile: %s\n", joinpath(sessiondir, "daemon.pid"));
        if ( is_file(joinpath(sessiondir, "daemon.pid")) == 0 ) {
            if ( ( daemon_fp = fopen(joinpath(sessiondir, "daemon.pid"), "r+") ) == NULL ) { // Flawfinder: ignore
                message(ERROR, "Could not open daemon pid file for writing %s: %s\n", joinpath(sessiondir, "daemon.pid"), strerror(errno));
                ABORT(255);
            }
        } else {
            if ( ( daemon_fp = fopen(joinpath(sessiondir, "daemon.pid"), "w") ) == NULL ) { // Flawfinder: ignore
                message(ERROR, "Could not open daemon pid file for writing %s: %s\n", joinpath(sessiondir, "daemon.pid"), strerror(errno));
                ABORT(255);
            }
        }

        message(VERBOSE, "Creating daemon.comm fifo\n");
        if ( is_fifo(joinpath(sessiondir, "daemon.comm")) < 0 ) {
            if ( mkfifo(joinpath(sessiondir, "daemon.comm"), 0664) < 0 ) {
                message(ERROR, "Could not create communication fifo: %s\n", strerror(errno));
                ABORT(255);
            }
        }

        daemon_fd = fileno(daemon_fp);
        if ( flock(daemon_fd, LOCK_EX | LOCK_NB) != 0 ) {
            message(ERROR, "Could not obtain lock, another daemon process running?\n");
            ABORT(255);
        }

        message(DEBUG, "Forking background daemon process\n");
        if ( daemon(0, 0) < 0 ) {
            message(ERROR, "Could not daemonize: %s\n", strerror(errno));
            ABORT(255);
        }
#endif
    } else if ( strcmp(command, "stop") == 0 ) {
        message(DEBUG, "Stopping namespace daemon process\n");
        return(container_daemon_stop(sessiondir));
    }



//****************************************************************************//
// Environment creation process flow
//****************************************************************************//


    message(VERBOSE, "Creating namespace process\n");
    // Fork off namespace process
    namespace_fork_pid = fork();
    if ( namespace_fork_pid == 0 ) {

        message(DEBUG, "Hello from namespace child process\n");
        if ( daemon_pid == -1 ) {
            namespace_unshare();

            config_rewind();
            int slave = config_get_key_bool("mount slave", 0);
            // Privatize the mount namespaces
#ifdef SINGULARITY_MS_SLAVE
            message(DEBUG, "Making mounts %s\n", (slave ? "slave" : "private"));
            if ( mount(NULL, "/", NULL, (slave ? MS_SLAVE : MS_PRIVATE)|MS_REC, NULL) < 0 ) {
                message(ERROR, "Could not make mountspaces %s: %s\n", (slave ? "slave" : "private"), strerror(errno));
                ABORT(255);
            }
#else
            if ( slave > 0 ) {
                message(WARNING, "Requested option 'mount slave' is not available on this host, using private\n");
            }
            message(DEBUG, "Making mounts private\n");
            if ( mount(NULL, "/", NULL, MS_PRIVATE | MS_REC, NULL) < 0 ) {
                message(ERROR, "Could not make mountspaces %s: %s\n", (slave ? "slave" : "private"), strerror(errno));
                ABORT(255);
            }
#endif


            if ( container_is_image > 0 ) {
                if ( getenv("SINGULARITY_WRITABLE") == NULL ) { // Flawfinder: ignore (only checking for existance of envar)
                    message(DEBUG, "Mounting Singularity image file read only\n");
                    if ( mount_image(loop_dev, containerdir, 0) < 0 ) {
                        ABORT(255);
                    }
                } else {
                    unsetenv("SINGULARITY_WRITABLE");
                    message(DEBUG, "Mounting Singularity image file read/write\n");
                    if ( mount_image(loop_dev, containerdir, 1) < 0 ) {
                        ABORT(255);
                    }
                }
            } else if ( container_is_dir > 0 ) {
            // TODO: container directories should also be mountable readwrite?
                message(DEBUG, "Mounting Singularity chroot read only\n");
                mount_bind(containerimage, containerdir, 0);
            }


            // /bin/sh MUST exist as the minimum requirements for a container
            message(DEBUG, "Checking if container has /bin/sh\n");
            if ( is_exec(joinpath(containerdir, "/bin/sh")) < 0 ) {
                message(ERROR, "Container image does not have a valid /bin/sh\n");
                ABORT(1);
            }


            // Bind mounts
            message(DEBUG, "Checking to see if we are running contained\n");
            if ( getenv("SINGULARITY_CONTAIN") == NULL ) { // Flawfinder: ignore (only checking for existance of envar)
                unsetenv("SINGULARITY_CONTAIN");

                message(DEBUG, "Checking configuration file for 'mount home'\n");
                config_rewind();
                if ( config_get_key_bool("mount home", 1) > 0 ) {
                    mount_home(containerdir);
                } else {
                    message(VERBOSE2, "Not mounting home directory per config\n");
                }

                bind_paths(containerdir);

            }

        } else {
            namespace_join(daemon_pid);
        }

        if ( uid != 0 ) { // If we are root, no need to mess with passwd or group
            message(DEBUG, "Checking configuration file for 'config passwd'\n");
            config_rewind();
            if ( config_get_key_bool("config passwd", 1) > 0 ) {
                if ( is_file(joinpath(sessiondir, "/passwd")) < 0 ) {
                    if (is_file(joinpath(containerdir, "/etc/passwd")) == 0 ) {
                        message(VERBOSE2, "Creating template of /etc/passwd for containment\n");
                        if ( ( copy_file(joinpath(containerdir, "/etc/passwd"), joinpath(sessiondir, "/passwd")) ) < 0 ) {
                            message(ERROR, "Failed copying template passwd file to sessiondir\n");
                            ABORT(255);
                        }
                    }
                    message(VERBOSE2, "Staging /etc/passwd with user info\n");
                    update_passwd_file(joinpath(sessiondir, "/passwd"));
                    message(VERBOSE, "Binding staged /etc/passwd into container\n");
                    mount_bind(joinpath(sessiondir, "/passwd"), joinpath(containerdir, "/etc/passwd"), 0);
                }
            } else {
                message(VERBOSE, "Not staging /etc/passwd per config\n");
            }

            message(DEBUG, "Checking configuration file for 'config group'\n");
            config_rewind();
            if ( config_get_key_bool("config group", 1) > 0 ) {
                if ( is_file(joinpath(sessiondir, "/group")) < 0 ) {
                    if (is_file(joinpath(containerdir, "/etc/group")) == 0 ) {
                        message(VERBOSE2, "Creating template of /etc/group for containment\n");
                        if ( ( copy_file(joinpath(containerdir, "/etc/group"), joinpath(sessiondir, "/group")) ) < 0 ) {
                            message(ERROR, "Failed copying template group file to sessiondir\n");
                            ABORT(255);
                        }
                    }
                    message(VERBOSE2, "Staging /etc/group with user info\n");
                    update_group_file(joinpath(sessiondir, "/group"));
                    message(VERBOSE, "Binding staged /etc/group into container\n");
                    mount_bind(joinpath(sessiondir, "/group"), joinpath(containerdir, "/etc/group"), 0);
                }
            } else {
                message(VERBOSE, "Not staging /etc/group per config\n");
            }
        } else {
            message(VERBOSE, "Not staging passwd or group (running as root)\n");
        }

        //  Handle scratch directories
        config_rewind();
        char *tmp_config_string;
        while ( ( tmp_config_string = config_get_key_value("bind scratch") ) != NULL ) {
            char *dest = tmp_config_string;
            if ( dest[0] == ' ' ) {
                dest++;
            }
            chomp(dest);
            message(VERBOSE2, "Found 'bind scratch' = %s\n", dest);
            if ( ( is_file(joinpath(containerdir, dest)) != 0 ) && ( is_dir(joinpath(containerdir, dest)) != 0 ) ) {
                message(WARNING, "Non existant 'bind scratch' in container: '%s'\n", dest);
                continue;
            }

            message(VERBOSE, "Binding '%s' to '%s:%s'\n", scratch_dir, containername, dest);
            mount_bind(scratch_dir, joinpath(containerdir, dest), 1);
        }

        // Handle user-specified scratch directories
        if ( ( tmp_config_string = getenv("SINGULARITY_USER_SCRATCH") ) != NULL ) {
#ifdef SINGULARITY_NO_NEW_PRIVS
            char *scratch = strdup(tmp_config_string);
            if ( scratch == NULL ) {
                message(ERROR, "Failed to allocate memory for configuration string.\n");
            }
            char *cur = scratch, *next = strchr(cur, ':');
            while ( cur != NULL ) {
                if (next) *next = '\0';
                chomp(cur);
                while (isspace(cur[0])) {cur++;}
                char *dest = cur;
                cur = next ? next + 1 : NULL;
                if (cur) {next = strchr(cur, ':');}
                if ( strlen(dest) == 0 ) {continue;}
                message(VERBOSE2, "Found user-specified scratch directory: '%s'\n", dest);
                if ( ( is_file(joinpath(containerdir, dest)) != 0 ) && ( is_dir(joinpath(containerdir, dest)) != 0 ) ) {
                    message(WARNING, "Non existant user-specified scratch directory in container: '%s'\n", dest);
                    continue;
                }

                message(VERBOSE, "Binding '%s' to '%s:%s'\n", scratch_dir, containername, dest);
                mount_bind(scratch_dir, joinpath(containerdir, dest), 1);
            }
            free(scratch);
#else  // SINGULARITY_NO_NEW_PRIVS
            // Without the NO_NEW_PRIVS flag, this would be a security hole: users might
            // wipe out directories that system setuid binaries depend on!
            message(ERROR, "Requested user-specified scratch directories, but they are not supported on this platform.\n");
            ABORT(255);
#endif  // SINGULARITY_NO_NEW_PRIVS
        }

        // Fork off exec process
        message(VERBOSE, "Forking exec process\n");

        exec_fork_pid = fork();
        if ( exec_fork_pid == 0 ) {
            message(DEBUG, "Hello from exec child process\n");

            message(VERBOSE, "Entering container file system space\n");
            if ( chroot(containerdir) < 0 ) { // Flawfinder: ignore (yep, yep, yep... we know!)
                message(ERROR, "failed enter CONTAINERIMAGE: %s\n", containerdir);
                ABORT(255);
            }
            message(DEBUG, "Changing dir to '/' within the new root\n");
            if ( chdir("/") < 0 ) {
                message(ERROR, "Could not chdir after chroot to /: %s\n", strerror(errno));
                ABORT(1);
            }


            if ( daemon_pid < 0 ) {
                // Mount /proc if we are configured
                message(DEBUG, "Checking configuration file for 'mount proc'\n");
                config_rewind();
                if ( config_get_key_bool("mount proc", 1) > 0 ) {
                    if ( is_dir("/proc") == 0 ) {
                        message(VERBOSE, "Mounting /proc\n");
                        if ( mount("proc", "/proc", "proc", 0, NULL) < 0 ) {
                            message(ERROR, "Could not mount /proc: %s\n", strerror(errno));
                            ABORT(255);
                        }
                    } else {
                        message(WARNING, "Not mounting /proc, container has no bind directory\n");
                    }
                } else {
                    message(VERBOSE, "Skipping /proc mount\n");
                }

                // Mount /sys if we are configured
                message(DEBUG, "Checking configuration file for 'mount sys'\n");
                config_rewind();
                if ( config_get_key_bool("mount sys", 1) > 0 ) {
                    if ( is_dir("/sys") == 0 ) {
                        message(VERBOSE, "Mounting /sys\n");
                        if ( mount("sysfs", "/sys", "sysfs", 0, NULL) < 0 ) {
                            message(ERROR, "Could not mount /sys: %s\n", strerror(errno));
                            ABORT(255);
                        }
                    } else {
                        message(WARNING, "Not mounting /sys, container has no bind directory\n");
                    }
                } else {
                    message(VERBOSE, "Skipping /sys mount\n");
                }
            }


            // Drop all privileges for good
            message(VERBOSE3, "Dropping all privileges\n");
            priv_drop_perm();

            // Change to the proper directory
            message(VERBOSE2, "Changing to correct working directory: %s\n", cwd);
            if ( is_dir(cwd) == 0 ) {
               if ( chdir(cwd) < 0 ) {
                    message(ERROR, "Could not chdir to: %s: %s\n", cwd, strerror(errno));
                    ABORT(1);
                }
            } else {
                if ( fchdir(cwd_fd) < 0 ) {
                    message(ERROR, "Could not fchdir to cwd: %s\n", strerror(errno));
                    ABORT(1);
                }
            }

            // Resetting umask
            umask(process_mask); // Flawfinder: ignore (resetting back to original umask)

            // After this, we exist only within the container... Let's make it known!
            message(DEBUG, "Setting environment variable 'SINGULARITY_CONTAINER=1'\n");
            if ( setenv("SINGULARITY_CONTAINER", containername, 1) != 0 ) {
                message(ERROR, "Could not set SINGULARITY_CONTAINER to '%s'\n", containername);
                ABORT(1);
            }

#ifdef SINGULARITY_NO_NEW_PRIVS
            // Prevent this container from gaining any future privileges.
            message(DEBUG, "Setting NO_NEW_PRIVS to prevent future privilege escalations.\n");
            if ( prctl(PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0) != 0 ) {
                message(ERROR, "Could not set NO_NEW_PRIVS safeguard: %s\n", strerror(errno));
                ABORT(1);
            }
#else  // SINGULARITY_NO_NEW_PRIVS
            message(VERBOSE2, "Not enabling NO_NEW_PRIVS flag due to lack of compile-time support.\n");
#endif
            // Do what we came here to do!
            if ( command == NULL ) {
                message(WARNING, "No command specified, launching 'shell'\n");
                command = strdup("shell");
            }
            if ( strcmp(command, "run") == 0 ) {
                message(VERBOSE, "COMMAND=run\n");
                if ( container_run(argc, argv) < 0 ) {
                    ABORT(255);
                }
            }
            if ( strcmp(command, "exec") == 0 ) {
                message(VERBOSE, "COMMAND=exec\n");
                if ( container_exec(argc, argv) < 0 ) {
                    ABORT(255);
                }
            }
            if ( strcmp(command, "shell") == 0 ) {
                message(VERBOSE, "COMMAND=shell\n");
                if ( container_shell(argc, argv) < 0 ) {
                    ABORT(255);
                }
            }
            if ( strcmp(command, "start") == 0 ) {
                message(VERBOSE, "COMMAND=start\n");
                if ( container_daemon_start(sessiondir) < 0 ) {
                    ABORT(255);
                }
                return(0);
            }

            message(ERROR, "Unknown command: %s\n", command);
            ABORT(255);


        // Wait for exec process to finish
        } else if ( exec_fork_pid > 0 ) {
            int tmpstatus;

            if ( strcmp(command, "start") == 0 ) {
                if ( fprintf(daemon_fp, "%d", exec_fork_pid) < 0 ) {
                    message(ERROR, "Could not write to daemon pid file: %s\n", strerror(errno));
                    ABORT(255);
                }
                fflush(daemon_fp);
            }

            strncpy(argv[0], "Singularity: exec", strlen(argv[0])); // Flawfinder: ignore

            message(VERBOSE3, "Dropping privilege...\n");
            priv_drop();

            message(VERBOSE2, "Waiting for Exec process...\n");

            waitpid(exec_fork_pid, &tmpstatus, 0);
            retval = WEXITSTATUS(tmpstatus);
        } else {
            message(ERROR, "Could not fork exec process: %s\n", strerror(errno));
            ABORT(255);
        }

        message(VERBOSE, "Exec parent process returned: %d\n", retval);
        return(retval);

    // Wait for namespace process to finish
    } else if ( namespace_fork_pid > 0 ) {
        int tmpstatus;
        strncpy(argv[0], "Singularity: namespace", strlen(argv[0])); // Flawfinder: ignore

        message(VERBOSE3, "Dropping privilege...\n");
        priv_drop();

        waitpid(namespace_fork_pid, &tmpstatus, 0);
        retval = WEXITSTATUS(tmpstatus);
    } else {
        message(ERROR, "Could not fork management process: %s\n", strerror(errno));
        ABORT(255);
    }

    message(VERBOSE2, "Starting cleanup...\n");

    // Final wrap up before exiting
    if ( close(cwd_fd) < 0 ) {
        message(ERROR, "Could not close cwd_fd: %s\n", strerror(errno));
        retval++;
    }


    message(DEBUG, "Closing the loop device file descriptor: %s\n", loop_fp);
    fclose(loop_fp);
    message(DEBUG, "Closing the container image file descriptor\n");
    fclose(containerimage_fp);

    message(DEBUG, "Checking to see if we are the last process running in this sessiondir\n");
    if ( flock(sessiondirlock_fd, LOCK_EX | LOCK_NB) == 0 ) {
        close(sessiondirlock_fd);

        message(VERBOSE3, "Escalating privs to clean session directory\n");
        priv_escalate();

        message(VERBOSE, "Cleaning sessiondir: %s\n", sessiondir);
        if ( s_rmdir(sessiondir) < 0 ) {
            message(WARNING, "Could not remove all files in %s: %s\n", sessiondir, strerror(errno));
        }

        message(DEBUG, "Calling loop_free(%s)\n", loop_dev);
        loop_free(loop_dev);

        priv_drop_perm();

    } else {
//        printf("Not removing sessiondir, lock still\n");
    }

    message(VERBOSE2, "Cleaning up...\n");

    if (scratch_dir) {
        s_rmdir(scratch_dir);
    }

    close(sessiondirlock_fd);

    free(loop_dev_lock);
    free(sessiondir);
    free(scratch_dir);

    return(retval);
}
