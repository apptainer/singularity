/* 
 * Copyright (c) 2015-2016, Gregory M. Kurtzer. All rights reserved.
 * 
 * “Singularity” Copyright (c) 2016, The Regents of the University of California,
 * through Lawrence Berkeley National Laboratory (subject to receipt of any
 * required approvals from the U.S. Dept. of Energy).  All rights reserved.
 * 
 * If you have questions about your rights to use or distribute this software,
 * please contact Berkeley Lab's Innovation & Partnerships Office at
 * IPO@lbl.gov.
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
#include <syslog.h>
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


#ifndef SYSCONFDIR
#define SYSCONFDIR "/etc"
#endif

#ifndef MS_PRIVATE
#define MS_PRIVATE (1<<18)
#endif
#ifndef MS_REC
#define MS_REC 16384
#endif

#ifdef NS_CLONE_PID
#define NS_CLONE_NEWPID NS_CLONE_PID
#endif

pid_t exec_fork_pid = 0;

void sighandler(int sig) {
    signal(sig, sighandler);

    if ( exec_fork_pid > 0 ) {
        fprintf(stderr, "Singularity is sending SIGKILL to child pid: %d\n", exec_fork_pid);

        kill(exec_fork_pid, SIGKILL);
    }
}



int main(int argc, char ** argv) {
    FILE *containerimage_fp;
    FILE *loop_fp;
    FILE *config_fp;
    FILE *daemon_fp = NULL;
    char *containerimage;
    char *containername;
    char *containerpath;
    char *username;
    char *command;
    char *tmpdir;
    char *prompt;
    char *loop_dev_lock;
    char *loop_dev_cache;
    char *loop_dev = 0;
    char *config_path;
    char *tmp_config_string;
    char setns_dir[128+9];
    char cwd[PATH_MAX];
    int cwd_fd;
    int tmpdirlock_fd;
    int containerimage_fd;
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
    signal(SIGKILL, sighandler);
    signal(SIGQUIT, sighandler);

    openlog("Singularity", LOG_CONS | LOG_NDELAY, LOG_LOCAL0);

    // Get all user/group info
    uid = getuid();
    pw = getpwuid(uid);

    if ( get_user_privs(&uinfo) < 0 ) {
        fprintf(stderr, "ABORT...\n");
        return(255);
    }

    // Check to make sure we are installed correctly
    if ( escalate_privs() < 0 ) {
        fprintf(stderr, "ABORT: Check installation, must be performed by root.\n");
        return(255);
    }

    // Lets start off as the calling UID
    if ( drop_privs(&uinfo) < 0 ) {
        fprintf(stderr, "ABORT...\n");
        return(255);
    }

    username = pw->pw_name;
    containerimage = getenv("SINGULARITY_IMAGE");
    command = getenv("SINGULARITY_COMMAND");

    unsetenv("SINGULARITY_COMMAND");
    unsetenv("SINGULARITY_EXEC");

    config_path = (char *) malloc(strlen(SYSCONFDIR) + 30);
    snprintf(config_path, strlen(SYSCONFDIR) + 30, "%s/singularity/singularity.conf", SYSCONFDIR);

    // Figure out where we start
    if ( (cwd_fd = open(".", O_RDONLY)) < 0 ) {
        fprintf(stderr, "ABORT: Could not open cwd fd (%s)!\n", strerror(errno));
        return(1);
    }
    if ( getcwd(cwd, PATH_MAX) == NULL ) {
        fprintf(stderr, "Could not obtain current directory path: %s\n", strerror(errno));
        return(1);
    }

    if ( containerimage == NULL ) {
        fprintf(stderr, "ABORT: SINGULARITY_IMAGE undefined!\n");
        return(1);
    }

    if ( is_file(containerimage) != 0 ) {
        fprintf(stderr, "ABORT: Container image path is invalid: %s\n", containerimage);
        return(1);
    }

    if ( is_file(config_path) != 0 ) {
        fprintf(stderr, "ABORT: Configuration file not found: %s\n", config_path);
        return(255);
    }

    if ( is_owner(config_path, 0) != 0 ) {
        fprintf(stderr, "ABORT: Configuration file is not owned by root: %s\n", config_path);
        return(255);
    }

    // TODO: Offer option to only run containers owned by root (so root can approve
    // containers)
    if ( uid == 0 && is_owner(containerimage, 0) < 0 ) {
        fprintf(stderr, "ABORT: Root should only run containers that root owns!\n");
        return(1);
    }

    containername = basename(strdup(containerimage));

    tmpdir = strjoin("/tmp/.singularity-", file_id(containerimage));
    loop_dev_lock = joinpath(tmpdir, "loop_dev.lock");
    loop_dev_cache = joinpath(tmpdir, "loop_dev");

    containerpath = (char *) malloc(strlen(tmpdir) + 5);
    snprintf(containerpath, strlen(tmpdir) + 5, "%s/mnt", tmpdir);

    syslog(LOG_NOTICE, "User=%s[%d], Command=%s, Container=%s, CWD=%s, Arg1=%s", username, uid, command, containerimage, cwd, argv[1]);


//****************************************************************************//
// Setup
//****************************************************************************//


    prompt = (char *) malloc(strlen(containername) + 16);
    snprintf(prompt, strlen(containername) + 16, "Singularity/%s> ", containername);
    setenv("PS1", prompt, 1);

    if ( ( config_fp = fopen(config_path, "r") ) == NULL ) {
        fprintf(stderr, "ERROR: Could not open config file %s: %s\n", config_path, strerror(errno));
        return(255);
    }

    if ( getenv("SINGULARITY_WRITABLE") == NULL ) {
        if ( ( containerimage_fp = fopen(containerimage, "r") ) == NULL ) {
            fprintf(stderr, "ERROR: Could not open image read only %s: %s\n", containerimage, strerror(errno));
            return(255);
        }
        containerimage_fd = fileno(containerimage_fp);
        if ( flock(containerimage_fd, LOCK_SH | LOCK_NB) < 0 ) {
            fprintf(stderr, "ABORT: Image is locked by another process\n");
            return(5);
        }
    } else {
        if ( ( containerimage_fp = fopen(containerimage, "r+") ) == NULL ) {
            fprintf(stderr, "ERROR: Could not open image read/write %s: %s\n", containerimage, strerror(errno));
            return(255);
        }
        containerimage_fd = fileno(containerimage_fp);
        if ( flock(containerimage_fd, LOCK_EX | LOCK_NB) < 0 ) {
            fprintf(stderr, "ABORT: Image is locked by another process\n");
            return(5);
        }
    }


    if ( is_file(joinpath(tmpdir, "daemon.pid")) == 0 ) {
        FILE *test_daemon_fp;
        int daemon_fd;

        if ( ( test_daemon_fp = fopen(joinpath(tmpdir, "daemon.pid"), "r") ) == NULL ) {
            fprintf(stderr, "ERROR: Could not open daemon pid file %s: %s\n", joinpath(tmpdir, "daemon.pid"), strerror(errno));
            return(255);
        }

        daemon_fd = fileno(test_daemon_fp);
        if ( flock(daemon_fd, LOCK_SH | LOCK_NB) != 0 ) {
            char daemon_pid[128];

            if ( fgets(daemon_pid, 128, test_daemon_fp) != NULL ) {
                snprintf(setns_dir, 128 + 9, "/proc/%s/ns", daemon_pid);
                if ( is_dir(setns_dir) == 0 ) {
                    join_daemon_ns = 1;
                }
            }

        } else {
            fprintf(stderr, "Dead Singularity daemon?\n");
        }
        fclose(test_daemon_fp);
    }


//****************************************************************************//
// We are now running with escalated privileges until we exec
//****************************************************************************//

    if ( seteuid(0) < 0 ) {
        fprintf(stderr, "ABORT: Could not escalate effective user privileges %s\n", strerror(errno));
        return(255);
    }
    if ( setegid(0) < 0 ) {
        fprintf(stderr, "ABORT: Could not escalate effective group privileges: %s\n", strerror(errno));
        return(255);
    }

    if ( s_mkpath(tmpdir, 0755) < 0 ) {
        fprintf(stderr, "ABORT: Could not create temporary directory %s: %s\n", tmpdir, strerror(errno));
        return(255);
    }

    if ( is_owner(tmpdir, 0) < 0 ) {
        fprintf(stderr, "ABORT: Container working directory has wrong ownership: %s\n", tmpdir);
        syslog(LOG_ERR, "Container working directory has wrong ownership: %s", tmpdir);
        return(255);
    }

    tmpdirlock_fd = open(tmpdir, O_RDONLY);
    if ( tmpdirlock_fd < 0 ) {
        fprintf(stderr, "ERROR: Could not obtain file descriptor on %s: %s\n", tmpdir, strerror(errno));
        return(255);
    }

    if ( flock(tmpdirlock_fd, LOCK_SH | LOCK_NB) < 0 ) {
        fprintf(stderr, "ERROR: Could not obtain shared lock on %s: %s\n", tmpdir, strerror(errno));
        return(255);
    }

    if ( ( loop_dev_lock_fd = open(loop_dev_lock, O_CREAT | O_RDWR, 0644) ) < 0 ) {
        fprintf(stderr, "ERROR: Could not open loop_dev_lock %s: %s\n", loop_dev_lock, strerror(errno));
        return(255);
    }

    if ( s_mkpath(containerpath, 0755) < 0 ) {
        fprintf(stderr, "ABORT: Could not create directory %s: %s\n", containerpath, strerror(errno));
        return(255);
    }

    if ( is_owner(containerpath, 0) < 0 ) {
        fprintf(stderr, "ABORT: Container directory is not root owned: %s\n", containerpath);
        syslog(LOG_ERR, "Container directory has wrong ownership: %s", tmpdir);
        return(255);
    }

    if ( flock(loop_dev_lock_fd, LOCK_EX | LOCK_NB) == 0 ) {
        loop_dev = obtain_loop_dev();

        if ( ( loop_fp = fopen(loop_dev, "r+") ) < 0 ) {
            fprintf(stderr, "ERROR: Failed to open loop device %s: %s\n", loop_dev, strerror(errno));
            syslog(LOG_ERR, "Failed to open loop device %s: %s", loop_dev, strerror(errno));
            return(255);
        }

        if ( associate_loop(containerimage_fp, loop_fp, 1) < 0 ) {
            fprintf(stderr, "ERROR: Could not associate %s to loop device %s\n", containerimage, loop_dev);
            syslog(LOG_ERR, "Failed to associate %s to loop device %s", containerimage, loop_dev);
            return(255);
        }

        if ( fileput(loop_dev_cache, loop_dev) < 0 ) {
            fprintf(stderr, "ERROR: Could not write to loop_dev_cache %s: %s\n", loop_dev_cache, strerror(errno));
            return(255);
        }

        flock(loop_dev_lock_fd, LOCK_SH | LOCK_NB);

    } else {
        flock(loop_dev_lock_fd, LOCK_SH);
        if ( ( loop_dev = filecat(loop_dev_cache) ) == NULL ) {
            fprintf(stderr, "ERROR: Could not retrieve loop_dev_cache from %s\n", loop_dev_cache);
            return(255);
        }

        if ( ( loop_fp = fopen(loop_dev, "r") ) < 0 ) {
            fprintf(stderr, "ERROR: Failed to open loop device %s: %s\n", loop_dev, strerror(errno));
            return(255);
        }
    }


    // Manage the daemon bits early
    if ( strcmp(command, "start") == 0 ) {
            int daemon_fd;

            if ( is_file(joinpath(tmpdir, "daemon.pid")) == 0 ) {
                if ( ( daemon_fp = fopen(joinpath(tmpdir, "daemon.pid"), "r+") ) == NULL ) {
                    fprintf(stderr, "ERROR: Could not open daemon pid file for writing %s: %s\n", joinpath(tmpdir, "daemon.pid"), strerror(errno));
                    return(255);
                }
            } else {
                if ( ( daemon_fp = fopen(joinpath(tmpdir, "daemon.pid"), "w") ) == NULL ) {
                    fprintf(stderr, "ERROR: Could not open daemon pid file for writing %s: %s\n", joinpath(tmpdir, "daemon.pid"), strerror(errno));
                    return(255);
                }
            }

            daemon_fd = fileno(daemon_fp);
            if ( flock(daemon_fd, LOCK_EX | LOCK_NB) != 0 ) {
                fprintf(stderr, "ERROR: Could not obtain lock, another daemon process running?\n");
                return(255);
            }

            if ( is_fifo(joinpath(tmpdir, "daemon.comm")) < 0 ) {
                if ( mkfifo(joinpath(tmpdir, "daemon.comm"), 0664) < 0 ) {
                    fprintf(stderr, "ERROR: Could not create communication fifo: %s\n", strerror(errno));
                    return(255);
                }
            }

        if ( daemon(1,1) < 0 ) {
            fprintf(stderr, "ERROR: Could not daemonize: %s\n", strerror(errno));
            return(255);
        }
    } else if ( strcmp(command, "stop") == 0 ) {
        return(container_daemon_stop(tmpdir));
    }



//****************************************************************************//
// Environment creation process flow
//****************************************************************************//

    if ( join_daemon_ns == 0 ) {

        // Fork off namespace process
        namespace_fork_pid = fork();
        if ( namespace_fork_pid == 0 ) {

            // Setup PID namespaces
            if ( getenv("SINGULARITY_NO_NAMESPACE_PID") == NULL ) {
                unsetenv("SINGULARITY_NO_NAMESPACE_PID");
                if ( unshare(CLONE_NEWPID) < 0 ) {
                    fprintf(stderr, "ABORT: Could not virtualize PID namespace: %s\n", strerror(errno));
                    return(255);
                }
            }

            // Setup FS namespaces
            if ( unshare(CLONE_FS) < 0 ) {
                fprintf(stderr, "ABORT: Could not virtualize file system namespace: %s\n", strerror(errno));
                return(255);
            }

            // Setup mount namespaces
            if ( unshare(CLONE_NEWNS) < 0 ) {
                fprintf(stderr, "ABORT: Could not virtualize mount namespace: %s\n", strerror(errno));
                return(255);
            }

            // Setup FS namespaces
//           if ( unshare(CLONE_FILES) < 0 ) {
//               fprintf(stderr, "ABORT: Could not virtualize file descriptor namespace: %s\n", strerror(errno));
//               return(255);
//           }

            // Privatize the mount namespaces
            if ( mount(NULL, "/", NULL, MS_PRIVATE|MS_REC, NULL) < 0 ) {
                fprintf(stderr, "ABORT: Could not make mountspaces private: %s\n", strerror(errno));
                return(255);
            }


            // Mount image
            if ( getenv("SINGULARITY_WRITABLE") == NULL ) {
                unsetenv("SINGULARITY_WRITABLE");
                if ( mount_image(loop_dev, containerpath, 0) < 0 ) {
                    fprintf(stderr, "ABORT: exiting...\n");
                    return(255);
                }
            } else {
                if ( mount_image(loop_dev, containerpath, 1) < 0 ) {
                    fprintf(stderr, "ABORT: exiting...\n");
                    return(255);
                }
            }


            // /bin/sh MUST exist as the minimum requirements for a container
            if ( is_exec(joinpath(containerpath, "/bin/sh")) < 0 ) {
                fprintf(stderr, "ERROR: Container image does not have a valid /bin/sh\n");
                return(1);
            }


            // Bind mounts
            if ( getenv("SINGULARITY_CONTAIN") == NULL ) {
                unsetenv("SINGULARITY_CONTAIN");
    
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

                    if ( ( is_file(source) != 0 ) && ( is_dir(source) != 0 ) ) {
                        fprintf(stderr, "ERROR: Non existant bind source path: '%s'\n", source);
                        continue;
                    }
                    if ( ( is_file(joinpath(containerpath, dest)) != 0 ) && ( is_dir(joinpath(containerpath, dest)) != 0 ) ) {
                        fprintf(stderr, "WARNING: Non existant bind container destination path: '%s'\n", dest);
                        continue;
                    }
                    if ( mount_bind(source, joinpath(containerpath, dest), 1) < 0 ) {
                        fprintf(stderr, "ABORTING!\n");
                        return(255);
                    }
                }

                if ( uid != 0 ) { // If we are root, no need to mess with passwd or group
                    rewind(config_fp);
                    if ( config_get_key_bool(config_fp, "config passwd", 1) > 0 ) {
                        if (is_file(joinpath(containerpath, "/etc/passwd")) == 0 ) {
                            if ( is_file(joinpath(tmpdir, "/passwd")) < 0 ) {
                                if ( build_passwd(joinpath(containerpath, "/etc/passwd"), joinpath(tmpdir, "/passwd")) < 0 ) {
                                    fprintf(stderr, "ABORT: Failed creating template password file\n");
                                    return(255);
                                }
                            }
                            if ( mount_bind(joinpath(tmpdir, "/passwd"), joinpath(containerpath, "/etc/passwd"), 1) < 0 ) {
                                fprintf(stderr, "ABORT: Could not bind /etc/passwd\n");
                                return(255);
                            }
                        }
                    }

                    rewind(config_fp);
                    if ( config_get_key_bool(config_fp, "config passwd", 1) > 0 ) {
                        if (is_file(joinpath(containerpath, "/etc/group")) == 0 ) {
                            if ( is_file(joinpath(tmpdir, "/group")) < 0 ) {
                                if ( build_group(joinpath(containerpath, "/etc/group"), joinpath(tmpdir, "/group")) < 0 ) {
                                    fprintf(stderr, "ABORT: Failed creating template group file\n");
                                    return(255);
                                }
                            }
                            if ( mount_bind(joinpath(tmpdir, "/group"), joinpath(containerpath, "/etc/group"), 1) < 0 ) {
                                fprintf(stderr, "ABORT: Could not bind /etc/group\n");
                                return(255);
                            }
                        }
                    }
                }
            }

            // Fork off exec process
            exec_fork_pid = fork();
            if ( exec_fork_pid == 0 ) {

                if ( chroot(containerpath) < 0 ) {
                    fprintf(stderr, "ABORT: failed enter CONTAINERIMAGE: %s\n", containerpath);
                    return(255);
                }
                if ( chdir("/") < 0 ) {
                    fprintf(stderr, "ABORT: Could not chdir after chroot to /: %s\n", strerror(errno));
                    return(1);
                }


                // Mount /proc if we are configured
                rewind(config_fp);
                if ( config_get_key_bool(config_fp, "mount proc", 1) > 0 ) {
                    if ( is_dir("/proc") == 0 ) {
                        if ( mount("proc", "/proc", "proc", 0, NULL) < 0 ) {
                            fprintf(stderr, "ABORT: Could not mount /proc: %s\n", strerror(errno));
                            return(255);
                        }
                    }
                }

                // Mount /sys if we are configured
                rewind(config_fp);
                if ( config_get_key_bool(config_fp, "mount sys", 1) > 0 ) {
                    if ( is_dir("/sys") == 0 ) {
                        if ( mount("sysfs", "/sys", "sysfs", 0, NULL) < 0 ) {
                            fprintf(stderr, "ABORT: Could not mount /sys: %s\n", strerror(errno));
                            return(255);
                        }
                    }
                }


                // Drop all privileges for good
                if ( drop_privs_perm(&uinfo) < 0 ) {
                    fprintf(stderr, "ABORT...\n");
                    return(255);
                }


                // Change to the proper directory
                if ( is_dir(cwd) == 0 ) {
                   if ( chdir(cwd) < 0 ) {
                        fprintf(stderr, "ABORT: Could not chdir to: %s: %s\n", cwd, strerror(errno));
                        return(1);
                    }
                } else {
                    if ( fchdir(cwd_fd) < 0 ) {
                        fprintf(stderr, "ABORT: Could not fchdir to cwd: %s\n", strerror(errno));
                        return(1);
                    }
                }

                // After this, we exist only within the container... Let's make it known!
                if ( setenv("SINGULARITY_CONTAINER", "true", 0) != 0 ) {
                    fprintf(stderr, "ABORT: Could not set SINGULARITY_CONTAINER to 'true'\n");
                    return(1);
                }


                // Do what we came here to do!
                if ( command == NULL ) {
                    fprintf(stderr, "No command specified, launching 'shell'\n");
                    command = strdup("shell");
                }
                if ( strcmp(command, "run") == 0 ) {
                    if ( container_run(argc, argv) < 0 ) {
                        fprintf(stderr, "ABORTING...\n");
                        return(255);
                    }
                }
                if ( strcmp(command, "exec") == 0 ) {
                    if ( container_exec(argc, argv) < 0 ) {
                        fprintf(stderr, "ABORTING...\n");
                        return(255);
                    }
                }
                if ( strcmp(command, "shell") == 0 ) {
                    if ( container_shell(argc, argv) < 0 ) {
                        fprintf(stderr, "ABORTING...\n");
                        return(255);
                    }
                }
                if ( strcmp(command, "start") == 0 ) {
                    //strncpy(argv[0], "Singularity Init", strlen(argv[0]));

                    if ( container_daemon_start(tmpdir) < 0 ) {
                        fprintf(stderr, "ABORTING...\n");
                        return(255);
                    }
                }

                fprintf(stderr, "ERROR: Unknown command: %s\n", command);
                return(255);


            // Wait for exec process to finish
            } else if ( exec_fork_pid > 0 ) {
                int tmpstatus;

                if ( strcmp(command, "start") == 0 ) {
                    if ( fprintf(daemon_fp, "%d", exec_fork_pid) < 0 ) {
                        fprintf(stderr, "ERROR: Could not write to daemon pid file: %s\n", strerror(errno));
                        return(255);
                    }
                    fflush(daemon_fp);
                }

                strncpy(argv[0], "Singularity: exec", strlen(argv[0]));

                if ( drop_privs(&uinfo) < 0 ) {
                    fprintf(stderr, "ABORT...\n");
                    return(255);
                }

                waitpid(exec_fork_pid, &tmpstatus, 0);
                retval = WEXITSTATUS(tmpstatus);
            } else {
                fprintf(stderr, "ABORT: Could not fork namespace process: %s\n", strerror(errno));
                return(255);
            }
            return(retval);

        // Wait for namespace process to finish
        } else if ( namespace_fork_pid > 0 ) {
            int tmpstatus;

            strncpy(argv[0], "Singularity: namespace", strlen(argv[0]));

            if ( drop_privs(&uinfo) < 0 ) {
                fprintf(stderr, "ABORT...\n");
                return(255);
            }

            waitpid(namespace_fork_pid, &tmpstatus, 0);
            retval = WEXITSTATUS(tmpstatus);
        } else {
            fprintf(stderr, "ABORT: Could not fork management process: %s\n", strerror(errno));
            return(255);
        }


        // Final wrap up before exiting
        if ( close(cwd_fd) < 0 ) {
            fprintf(stderr, "ERROR: Could not close cwd_fd: %s\n", strerror(errno));
            retval++;
        }

        if ( flock(tmpdirlock_fd, LOCK_EX | LOCK_NB) == 0 ) {
            close(tmpdirlock_fd);
            if ( escalate_privs() < 0 ) {
                fprintf(stderr, "ABORT...\n");
                return(255);
            }

            if ( s_rmdir(tmpdir) < 0 ) {
                fprintf(stderr, "WARNING: Could not remove all files in %s: %s\n", tmpdir, strerror(errno));
            }
    
            // Dissociate loops from here Just in case autoflush didn't work.
            (void)disassociate_loop(loop_fp);

            if ( seteuid(uid) < 0 ) {
                fprintf(stderr, "ABORT: Could not drop effective user privileges: %s\n", strerror(errno));
                return(255);
            }

        } else {
//        printf("Not removing tmpdir, lock still\n");
        }



//****************************************************************************//
// Attach to daemon process flow
//****************************************************************************//
    } else {

        pid_t exec_pid;


        // Connect to existing PID namespace
        if ( is_file(joinpath(setns_dir, "pid")) == 0 ) {
            int fd = open(joinpath(setns_dir, "pid"), O_RDONLY);
            if ( setns(fd, CLONE_NEWPID) < 0 ) {
                fprintf(stderr, "ABORT: Could not join existing PID namespace: %s\n", strerror(errno));
                return(255);
            }
            close(fd);

        } else {
            fprintf(stderr, "ABORT: Could not identify PID namespace: %s\n", joinpath(setns_dir, "pid"));
        }

        // Connect to existing mount namespace
        if ( is_file(joinpath(setns_dir, "mnt")) == 0 ) {
            int fd = open(joinpath(setns_dir, "mnt"), O_RDONLY);
            if ( setns(fd, CLONE_NEWNS) < 0 ) {
                fprintf(stderr, "ABORT: Could not join existing mount namespace: %s\n", strerror(errno));
                return(255);
            }
            close(fd);

        } else {
            fprintf(stderr, "ABORT: Could not identify mount namespace: %s\n", joinpath(setns_dir, "mnt"));
        }


//TODO: Add chroot and chdirs to a container method
        if ( chroot(containerpath) < 0 ) {
            fprintf(stderr, "ABORT: failed enter CONTAINERIMAGE: %s\n", containerpath);
            return(255);
        }
        if ( chdir("/") < 0 ) {
            fprintf(stderr, "ABORT: Could not chdir after chroot to /: %s\n", strerror(errno));
            return(1);
        }

        // Change to the proper directory
        if ( is_dir(cwd) == 0 ) {
           if ( chdir(cwd) < 0 ) {
                fprintf(stderr, "ABORT: Could not chdir to: %s: %s\n", cwd, strerror(errno));
                return(1);
            }
        } else {
            if ( fchdir(cwd_fd) < 0 ) {
                fprintf(stderr, "ABORT: Could not fchdir to cwd: %s\n", strerror(errno));
                return(1);
            }
        }

        // Drop all privileges for good
        if ( drop_privs_perm(&uinfo) < 0 ) {
            fprintf(stderr, "ABORT...\n");
            return(255);
        }

        // Fork off exec process
        exec_pid = fork();
        if ( exec_pid == 0 ) {

            // Do what we came here to do!
            if ( command == NULL ) {
                fprintf(stderr, "No command specified, launching 'shell'\n");
                command = strdup("shell");
            }
            if ( strcmp(command, "run") == 0 ) {
                if ( container_run(argc, argv) < 0 ) {
                    fprintf(stderr, "ABORTING...\n");
                    return(255);
                }
            }
            if ( strcmp(command, "exec") == 0 ) {
                if ( container_exec(argc, argv) < 0 ) {
                    fprintf(stderr, "ABORTING...\n");
                    return(255);
                }
            }
            if ( strcmp(command, "shell") == 0 ) {
                if ( container_shell(argc, argv) < 0 ) {
                    fprintf(stderr, "ABORTING...\n");
                    return(255);
                }
            }
    
            fprintf(stderr, "ERROR: Unknown command: %s\n", command);
            return(255);

        } else if ( exec_pid > 0 ) {
            int tmpstatus;
    
            strncpy(argv[0], "Singularity: exec", strlen(argv[0]));
    
            if ( drop_privs(&uinfo) < 0 ) {
                fprintf(stderr, "ABORT...\n");
                return(255);
            }
    
            waitpid(exec_pid, &tmpstatus, 0);
            retval = WEXITSTATUS(tmpstatus);

        } else {
            fprintf(stderr, "ABORT: Could not fork exec process: %s\n", strerror(errno));
            return(255);
        }
    }

    close(containerimage_fd);
    close(tmpdirlock_fd);

    free(loop_dev_lock);
    free(containerpath);
    free(tmpdir);
    closelog();

    return(retval);
}
