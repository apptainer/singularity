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
    FILE *containerimage_fp;
    FILE *config_fp;
    FILE *daemon_fp = NULL;
    char *containerimage;
    char *containername;
    char *containerdir;
    char *command;
    char *sessiondir;
    char *sessiondir_prefix;
    char *loop_dev_lock;
    char *loop_dev_cache;
    char *homedir;
    char *homedir_base = 0;
    char *loop_dev = 0;
    char *config_path;
    char *tmp_config_string;
    char setns_dir[128+9]; // Flawfinder: ignore
    char cwd[PATH_MAX]; // Flawfinder: ignore
    int cwd_fd;
    int sessiondirlock_fd;
    int loop_dev_lock_fd;
    int join_daemon_ns = 0;
    int retval = 0;
    uid_t uid;
    pid_t namespace_fork_pid = 0;
    struct passwd *pw;
    struct s_privinfo uinfo;



//****************************************************************************//
// Init
//****************************************************************************//

    signal(SIGINT, sighandler);
    signal(SIGQUIT, sighandler);
    signal(SIGTERM, sighandler);
    signal(SIGKILL, sighandler);

    // Get all user/group info
    uid = getuid();
    pw = getpwuid(uid);

    message(DEBUG, "Gathering and caching user info.\n");
    if ( get_user_privs(&uinfo) < 0 ) {
        message(ERROR, "Could not obtain user privs\n");
        ABORT(255);
    }

    // Check to make sure we are installed correctly
    message(DEBUG, "Checking if we can escalate privs properly.\n");
    if ( escalate_privs() < 0 ) {
        message(ERROR, "Check installation, must be performed by root\n");
        ABORT(255);
    }

    // Lets start off as the calling UID
    message(DEBUG, "Setting privs to calling user\n");
    if ( drop_privs(&uinfo) < 0 ) {
        ABORT(255);
    }

    message(DEBUG, "Obtaining user's homedir\n");
    homedir = pw->pw_dir;

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
    if ( is_file(containerimage) != 0 ) {
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
    if ( ( config_fp = fopen(config_path, "r") ) == NULL ) { // Flawfinder: ignore
        message(ERROR, "Could not open config file %s: %s\n", config_path, strerror(errno));
        ABORT(255);
    }

    // TODO: Offer option to only run containers owned by root (so root can approve
    // containers)
//    if ( uid == 0 && is_owner(containerimage, 0) < 0 ) {
//        message(ERROR, "Root should only run containers that root owns!\n");
//        ABORT(1);
//    }

    message(DEBUG, "Checking Singularity configuration for 'sessiondir prefix'\n");
    rewind(config_fp);
    if ( ( sessiondir_prefix = config_get_key_value(config_fp, "sessiondir prefix") ) != NULL ) {
        sessiondir = strjoin(sessiondir_prefix, file_id(containerimage));
    } else {
        sessiondir = strjoin("/tmp/.singularity-session-", file_id(containerimage));
    }
    message(DEBUG, "Set sessiondir to: %s\n", sessiondir);

    
    containername = basename(strdup(containerimage));
    message(DEBUG, "Set containername to: %s\n", containername);

    message(DEBUG, "Setting loop_dev_* paths\n");
    loop_dev_lock = joinpath(sessiondir, "loop_dev.lock");
    loop_dev_cache = joinpath(sessiondir, "loop_dev");

    rewind(config_fp);
    if ( ( containerdir = config_get_key_value(config_fp, "container dir") ) == NULL ) {
        //containerdir = (char *) malloc(21);
        containerdir = strdup("/var/singularity/mnt");
    }
    message(DEBUG, "Set image mount path to: %s\n", containerdir);

    message(LOG, "Command=%s, Container=%s, CWD=%s, Arg1=%s\n", command, containerimage, cwd, argv[1]);

    message(DEBUG, "Checking if we are opening image as read/write\n");
    if ( getenv("SINGULARITY_WRITABLE") != NULL ) { // Flawfinder: ignore (only checking for existance of getenv)
    	int containerimage_fd;

        if ( getuid() == 0 ) {
            message(DEBUG, "Opening image as read/write: %s\n", containerimage);
            if ( ( containerimage_fp = fopen(containerimage, "r+") ) == NULL ) { // Flawfinder: ignore
                message(ERROR, "Could not open image read/write %s: %s\n", containerimage, strerror(errno));
                ABORT(255);
            }

            containerimage_fd = fileno(containerimage_fp);
            message(DEBUG, "Setting exclusive lock on file descriptor: %d\n", containerimage_fd);
            if ( flock(containerimage_fd, LOCK_EX | LOCK_NB) < 0 ) {
                message(WARNING, "Could not obtain exclusive lock on image\n");
            }
        } else {
            message(ERROR, "Only root can mount images as writable\n");
            ABORT(1);
        }

    } else {

        message(DEBUG, "Opening image as read only: %s\n", containerimage);
        if ( ( containerimage_fp = fopen(containerimage, "r") ) == NULL ) { // Flawfinder: ignore 
            message(ERROR, "Could not open image read only %s: %s\n", containerimage, strerror(errno));
            ABORT(255);
        }

    }

    message(DEBUG, "Checking for namespace daemon pidfile\n");
    if ( is_file(joinpath(sessiondir, "daemon.pid")) == 0 ) {
        FILE *test_daemon_fp;
        int daemon_fd;

        if ( ( test_daemon_fp = fopen(joinpath(sessiondir, "daemon.pid"), "r") ) == NULL ) { // Flawfinder: ignore
            message(ERROR, "Could not open daemon pid file %s: %s\n", joinpath(sessiondir, "daemon.pid"), strerror(errno));
            ABORT(255);
        }

        message(DEBUG, "Checking if namespace daemon is running\n");
        daemon_fd = fileno(test_daemon_fp);
        if ( flock(daemon_fd, LOCK_SH | LOCK_NB) != 0 ) {
            char daemon_pid[128]; // Flawfinder: ignore

            if ( fgets(daemon_pid, 128, test_daemon_fp) != NULL ) {
                snprintf(setns_dir, 128 + 9, "/proc/%s/ns", daemon_pid); // Flawfinder: ignore
                if ( is_dir(setns_dir) == 0 ) {
                    message(VERBOSE, "Found namespace daemon process for this container\n");
                    join_daemon_ns = 1;
                }
            }

        } else {
            message(WARNING, "Singularity namespace daemon pid exists, but daemon not alive?\n");
        }
        fclose(test_daemon_fp);
    }


//****************************************************************************//
// We are now running with escalated privileges until we exec
//****************************************************************************//

    message(DEBUG, "Escalating privledges\n");
    if ( escalate_privs() < 0 ) {
        ABORT(255);
    }

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

    message(DEBUG, "Checking for set loop device\n");
    if ( ( loop_dev_lock_fd = open(loop_dev_lock, O_CREAT | O_RDWR, 0644) ) < 0 ) { // Flawfinder: ignore
        message(ERROR, "Could not open loop_dev_lock %s: %s\n", loop_dev_lock, strerror(errno));
        ABORT(255);
    }

    message(DEBUG, "Requesting exclusive flock() on loop_dev lockfile\n");
    if ( flock(loop_dev_lock_fd, LOCK_EX | LOCK_NB) == 0 ) {
        message(DEBUG, "We have exclusive flock() on loop_dev lockfile\n");

        message(DEBUG, "Binding container to loop interface\n");
        if ( ( loop_fp = loop_bind(containerimage_fp, &loop_dev, 1)) == NULL ) {
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

        message(DEBUG, "Forking namespace daemon process\n");
        if ( daemon(0,0) < 0 ) {
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

    message(DEBUG, "Checking to see if we are joining an existing namespace\n");
    if ( join_daemon_ns == 0 ) {

        message(VERBOSE, "Creating namespace process\n");
        // Fork off namespace process
        namespace_fork_pid = fork();
        if ( namespace_fork_pid == 0 ) {

            message(DEBUG, "Hello from namespace child process\n");
            // Setup PID namespaces
            rewind(config_fp);
#ifdef NS_CLONE_NEWPID
            if ( ( getenv("SINGULARITY_NO_NAMESPACE_PID") == NULL ) && // Flawfinder: ignore (only checking for existance of envar)
                    ( config_get_key_bool(config_fp, "allow pid ns", 1) > 0 ) ) {
                unsetenv("SINGULARITY_NO_NAMESPACE_PID");
                message(DEBUG, "Virtualizing PID namespace\n");
                if ( unshare(CLONE_NEWPID) < 0 ) {
                    message(ERROR, "Could not virtualize PID namespace: %s\n", strerror(errno));
                    ABORT(255);
                }
            } else {
                message(VERBOSE, "Not virtualizing PID namespace\n");
            }
#else
#ifdef NS_CLONE_PID
            if ( ( getenv("SINGULARITY_NO_NAMESPACE_PID") == NULL ) && // Flawfinder: ignore (only checking for existance of envar)
                    ( config_get_key_bool(config_fp, "allow pid ns", 1) > 0 ) ) {
                unsetenv("SINGULARITY_NO_NAMESPACE_PID");
                message(DEBUG, "Virtualizing PID namespace\n");
                if ( unshare(CLONE_NEWPID) < 0 ) {
                    message(ERROR, "Could not virtualize PID namespace: %s\n", strerror(errno));
                    ABORT(255);
                }
            } else {
                message(VERBOSE, "Not virtualizing PID namespace\n");
            }
#endif
#endif

#ifdef NS_CLONE_FS
            // Setup FS namespaces
            message(DEBUG, "Virtualizing FS namespace\n");
            if ( unshare(CLONE_FS) < 0 ) {
                message(ERROR, "Could not virtualize file system namespace: %s\n", strerror(errno));
                ABORT(255);
            }
#endif

            // Setup mount namespaces
            message(DEBUG, "Virtualizing mount namespace\n");
            if ( unshare(CLONE_NEWNS) < 0 ) {
                message(ERROR, "Could not virtualize mount namespace: %s\n", strerror(errno));
                ABORT(255);
            }

            // Privatize the mount namespaces
            message(DEBUG, "Making mounts private\n");
            if ( mount(NULL, "/", NULL, MS_PRIVATE|MS_REC, NULL) < 0 ) {
                message(ERROR, "Could not make mountspaces private: %s\n", strerror(errno));
                ABORT(255);
            }


            // Mount image
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


            // /bin/sh MUST exist as the minimum requirements for a container
            message(DEBUG, "Checking if container has /bin/sh\n");
            if ( is_exec(joinpath(containerdir, "/bin/sh")) < 0 ) {
                message(ERROR, "Container image does not have a valid /bin/sh\n");
                ABORT(1);
            }


            // Bind mounts
            message(DEBUG, "Checking to see if we should do bind mounts\n");
            if ( getenv("SINGULARITY_CONTAIN") == NULL ) { // Flawfinder: ignore (only checking for existance of envar)
                unsetenv("SINGULARITY_CONTAIN");

                message(DEBUG, "Checking configuration file for 'mount home'\n");
                rewind(config_fp);
                if ( config_get_key_bool(config_fp, "mount home", 1) > 0 ) {
                    if ( ( homedir_base = container_basedir(containerdir, homedir) ) != NULL ) {
                        if ( is_dir(homedir_base) == 0 ) {
                            if ( is_dir(joinpath(containerdir, homedir_base)) == 0 ) {
                                message(VERBOSE, "Mounting home directory base path: %s\n", homedir_base);
                                if ( mount_bind(homedir_base, joinpath(containerdir, homedir_base), 1) < 0 ) {
                                    ABORT(255);
                                }
                            } else {
                                message(WARNING, "Container bind point does not exist: '%s' (homedir_base)\n", homedir_base);
                            }
                        } else {
                            message(WARNING, "Home directory base source path does not exist: %s\n", homedir_base);
                        }
                    }
                } else {
                    message(VERBOSE2, "Not mounting home directory...\n");
                }

                message(DEBUG, "Checking configuration file for 'bind path'\n");
                rewind(config_fp);
                while ( ( tmp_config_string = config_get_key_value(config_fp, "bind path") ) != NULL ) {
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

                    if ( ( homedir_base != NULL ) && ( strncmp(dest, homedir_base, strlength(homedir_base, 256)) == 0 )) {
                        // Skipping path as it was already mounted as homedir_base
                        message(VERBOSE2, "Skipping '%s' as it is part of home path and already mounted\n", dest);
                        continue;
                    }

                    if ( ( is_file(source) != 0 ) && ( is_dir(source) != 0 ) ) {
                        message(WARNING, "Non existant 'bind path' source: '%s'\n", source);
                        continue;
                    }
                    if ( ( is_file(joinpath(containerdir, dest)) != 0 ) && ( is_dir(joinpath(containerdir, dest)) != 0 ) ) {
                        message(WARNING, "Non existant 'bind point' in container: '%s'\n", dest);
                        continue;
                    }

                    message(VERBOSE, "Binding '%s' to '%s:%s'\n", source, containername, dest);
                    if ( mount_bind(source, joinpath(containerdir, dest), 1) < 0 ) {
                        ABORT(255);
                    }
                }
            }


            if ( uid != 0 ) { // If we are root, no need to mess with passwd or group
                message(DEBUG, "Checking configuration file for 'config passwd'\n");
                rewind(config_fp);
                if ( config_get_key_bool(config_fp, "config passwd", 1) > 0 ) {
                    if (is_file(joinpath(containerdir, "/etc/passwd")) == 0 ) {
                        if ( is_file(joinpath(sessiondir, "/passwd")) < 0 ) {
                            message(VERBOSE2, "Staging /etc/passwd with user info\n");
                            if ( build_passwd(joinpath(containerdir, "/etc/passwd"), joinpath(sessiondir, "/passwd")) < 0 ) {
                                message(ERROR, "Failed creating template password file\n");
                                ABORT(255);
                            }
                        }
                        message(VERBOSE, "Binding staged /etc/passwd into container\n");
                        if ( mount_bind(joinpath(sessiondir, "/passwd"), joinpath(containerdir, "/etc/passwd"), 1) < 0 ) {
                            message(ERROR, "Could not bind /etc/passwd\n");
                            ABORT(255);
                        }
                    }
                } else {
                    message(VERBOSE, "Skipping /etc/passwd staging\n");
                }

                message(DEBUG, "Checking configuration file for 'config group'\n");
                rewind(config_fp);
                if ( config_get_key_bool(config_fp, "config group", 1) > 0 ) {
                    if (is_file(joinpath(containerdir, "/etc/group")) == 0 ) {
                        if ( is_file(joinpath(sessiondir, "/group")) < 0 ) {
                            message(VERBOSE2, "Staging /etc/group with user info\n");
                            if ( build_group(joinpath(containerdir, "/etc/group"), joinpath(sessiondir, "/group")) < 0 ) {
                                message(ERROR, "Failed creating template group file\n");
                                ABORT(255);
                            }
                        }
                        message(VERBOSE, "Binding staged /etc/group into container\n");
                        if ( mount_bind(joinpath(sessiondir, "/group"), joinpath(containerdir, "/etc/group"), 1) < 0 ) {
                            message(ERROR, "Could not bind /etc/group\n");
                            ABORT(255);
                        }
                    }
                } else {
                    message(VERBOSE, "Skipping /etc/group staging\n");
                }
            } else {
                message(VERBOSE, "Not staging passwd or group (running as root)\n");
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


                // Mount /proc if we are configured
                message(DEBUG, "Checking configuration file for 'mount proc'\n");
                rewind(config_fp);
                if ( config_get_key_bool(config_fp, "mount proc", 1) > 0 ) {
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
                rewind(config_fp);
                if ( config_get_key_bool(config_fp, "mount sys", 1) > 0 ) {
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


                // Drop all privileges for good
                message(VERBOSE2, "Dropping all privileges\n");
                if ( drop_privs_perm(&uinfo) < 0 ) {
                    ABORT(255);
                }


                if ( getenv("SINGULARITY_CONTAIN") == NULL ) { // Flawfinder: ignore (only checking for existance of envar)
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
                } else {
                    message(VERBOSE3, "Not chdir'ing to CWD, called with --contain\n");
                }

                // After this, we exist only within the container... Let's make it known!
                message(DEBUG, "Setting environment variable 'SINGULARITY_CONTAINER=1'\n");
                if ( setenv("SINGULARITY_CONTAINER", containername, 1) != 0 ) {
                    message(ERROR, "Could not set SINGULARITY_CONTAINER to '%s'\n", containername);
                    ABORT(1);
                }


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
                    //strncpy(argv[0], "Singularity Init", strlen(argv[0]));

                    if ( container_daemon_start(sessiondir) < 0 ) {
                        ABORT(255);
                    }
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

                message(DEBUG, "Dropping privs...\n");

                if ( drop_privs(&uinfo) < 0 ) {
                    ABORT(255);
                }

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

            if ( drop_privs(&uinfo) < 0 ) {
                ABORT(255);
            }

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


//****************************************************************************//
// Attach to daemon process flow
//****************************************************************************//
    } else {
#ifdef NO_SETNS
        message(ERROR, "This host does not support joining existing name spaces\n");
        ABORT(1);
#else

        message(VERBOSE, "Attaching to existing namespace daemon environment\n");
        pid_t exec_pid;

        if ( is_file(joinpath(setns_dir, "pid")) == 0 ) {
            message(DEBUG, "Connecting to existing PID namespace\n");
            int fd = open(joinpath(setns_dir, "pid"), O_RDONLY); // Flawfinder: ignore
            if ( setns(fd, CLONE_NEWPID) < 0 ) {
                message(ERROR, "Could not join existing PID namespace: %s\n", strerror(errno));
                ABORT(255);
            }
            close(fd);

        } else {
            message(ERROR, "Could not identify PID namespace: %s\n", joinpath(setns_dir, "pid"));
            ABORT(255);
        }

        // Connect to existing mount namespace
        if ( is_file(joinpath(setns_dir, "mnt")) == 0 ) {
            message(DEBUG, "Connecting to existing mount namespace\n");
            int fd = open(joinpath(setns_dir, "mnt"), O_RDONLY); // Flawfinder: ignore
            if ( setns(fd, CLONE_NEWNS) < 0 ) {
                message(ERROR, "Could not join existing mount namespace: %s\n", strerror(errno));
                ABORT(255);
            }
            close(fd);

        } else {
            message(ERROR, "Could not identify mount namespace: %s\n", joinpath(setns_dir, "mnt"));
            ABORT(255);
        }

#ifdef NS_CLONE_FS
        // Setup FS namespaces
        message(DEBUG, "Virtualizing FS namespace\n");
        if ( unshare(CLONE_FS) < 0 ) {
            message(ERROR, "Could not virtualize file system namespace: %s\n", strerror(errno));
            ABORT(255);
        }
#endif

        // Fork off exec process
        message(VERBOSE, "Forking exec process\n");
        exec_pid = fork();
        if ( exec_pid == 0 ) {

            message(DEBUG, "Hello from exec child process\n");

//TODO: Add chroot and chdirs to a container method
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

            // Drop all privileges for good
            message(VERBOSE2, "Dropping all privileges\n");
            if ( drop_privs_perm(&uinfo) < 0 ) {
                ABORT(255);
            }

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
    
            message(ERROR, "Unknown command: %s\n", command);
            ABORT(255);

        } else if ( exec_pid > 0 ) {
            int tmpstatus;
    
            strncpy(argv[0], "Singularity: exec", strlen(argv[0])); // Flawfinder: ignore
    
            message(DEBUG, "Dropping privs...\n");

            if ( drop_privs(&uinfo) < 0 ) {
                ABORT(255);
            }
    
            message(VERBOSE2, "Waiting for Exec process...\n");

            waitpid(exec_pid, &tmpstatus, 0);
            retval = WEXITSTATUS(tmpstatus);

        } else {
            message(ERROR, "Could not fork exec process: %s\n", strerror(errno));
            ABORT(255);
        }

        message(VERBOSE, "Exec parent process returned: %d\n", retval);
#endif
    }

    message(DEBUG, "Checking to see if we are the last process running in this sessiondir\n");


    message(DEBUG, "Closing the loop device file descriptor: %s\n", loop_fp);
    fclose(loop_fp);
    message(DEBUG, "Closing the container image file descriptor\n");
    fclose(containerimage_fp);

    if ( flock(sessiondirlock_fd, LOCK_EX | LOCK_NB) == 0 ) {
        close(sessiondirlock_fd);

        message(DEBUG, "Escalating privs to clean session directory\n");
        if ( escalate_privs() < 0 ) {
            ABORT(255);
        }

        message(VERBOSE, "Cleaning sessiondir: %s\n", sessiondir);
        if ( s_rmdir(sessiondir) < 0 ) {
            message(WARNING, "Could not remove all files in %s: %s\n", sessiondir, strerror(errno));
        }

        message(DEBUG, "Calling loop_free(%s)\n", loop_dev);
        loop_free(loop_dev);

        if ( drop_privs(&uinfo) < 0 ) {
            ABORT(255);
        }

    } else {
//        printf("Not removing sessiondir, lock still\n");
    }

    message(VERBOSE2, "Cleaning up...\n");

    close(sessiondirlock_fd);

    free(loop_dev_lock);
    free(sessiondir);

    return(retval);
}
