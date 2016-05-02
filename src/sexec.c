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

#include "config.h"
#include "mounts.h"
#include "util.h"
#include "user.h"


#ifndef LIBEXECDIR
#define LIBEXECDIR "undefined"
#endif
#ifndef SYSCONFDIR
#define SYSCONFDIR "/etc"
#endif
#ifndef LOCALSTATEDIR
#define LOCALSTATEDIR "/var/"
#endif

// Yes, I know... Global variables suck but necessary to pass sig to child
pid_t child_pid = 0;


void sighandler(int sig) {
    signal(sig, sighandler);

    printf("Caught signal: %d\n", sig);
    fflush(stdout);

    if ( child_pid > 0 ) {
        printf("Singularity is sending SIGKILL to child pid: %d\n", child_pid);
        fflush(stdout);

        kill(child_pid, SIGKILL);
    }
}


int main(int argc, char ** argv) {
    char *containerimage;
    char *containername;
    char *containerpath;
    char *homepath;
    char *command;
    char *command_exec;
    char *runpath;
    char cwd[PATH_MAX];
    int cwd_fd;
    int retval = 0;
    uid_t uid = getuid();
    gid_t gid = getgid();


    //****************************************************************************//
    // Init
    //****************************************************************************//

    // Lets start off as the calling UID
    if ( seteuid(uid) < 0 ) {
        fprintf(stderr, "ABORT: Could not set effective user privledges to %d!\n", uid);
        return(255);
    }

    homepath = getenv("HOME");
    containerimage = getenv("SINGULARITY_IMAGE");
    command = getenv("SINGULARITY_COMMAND");
    command_exec = getenv("SINGULARITY_EXEC");

    unsetenv("SINGULARITY_IMAGE");
    unsetenv("SINGULARITY_COMMAND");
    unsetenv("SINGULARITY_EXEC");

    // Figure out where we start
    if ( (cwd_fd = open(".", O_RDONLY)) < 0 ) {
        fprintf(stderr, "ABORT: Could not open cwd fd (%s)!\n", strerror(errno));
        return(1);
    }
    if ( getcwd(cwd, PATH_MAX) == NULL ) {
        fprintf(stderr, "Could not obtain current directory path\n");
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

    // TODO: Offer option to only run containers owned by root (so root can approve
    // containers)
    if ( is_owner(containerimage, uid) < 0 && is_owner(containerimage, 0) < 0 ) {
        fprintf(stderr, "ABORT: Will not execute in a CONTAINERIMAGE you (or root) does not own: %s\n", containerimage);
        return(255);
    }

    containername = basename(strdup(containerimage));

    containerpath = (char *) malloc(strlen(LOCALSTATEDIR) + 18);
    snprintf(containerpath, strlen(LOCALSTATEDIR) + 18, "%s/singularity/mnt", LOCALSTATEDIR);

    runpath = (char *) malloc(strlen(LOCALSTATEDIR) + strlen(containername) + intlen(uid) + 20);
    snprintf(runpath, strlen(LOCALSTATEDIR) + strlen(containername) + intlen(uid) + 20, "%s/singularity/run/%d/%s", LOCALSTATEDIR, uid, containername);


    //****************************************************************************//
    // Setup
    //****************************************************************************//

    if ( seteuid(0) < 0 ) {
        fprintf(stderr, "ABORT: Could not escalate effective user privledges!\n");
        return(255);
    }

    if ( is_dir(containerpath) < 0 ) {
        if ( s_mkpath(containerpath, S_IRUSR | S_IWUSR | S_IXUSR | S_IRGRP | S_IWGRP | S_IXGRP | S_IROTH | S_IXOTH) < 0 ) {
            fprintf(stderr, "ABORT: Could not create directory %s: %s\n", containerpath, strerror(errno));
            return(255);
        }
    }

    if ( is_dir(runpath) < 0 ) {
        if ( s_mkpath(runpath, S_IRUSR | S_IWUSR | S_IXUSR | S_IRGRP | S_IWGRP | S_IXGRP | S_IROTH | S_IXOTH) < 0 ) {
            fprintf(stderr, "ABORT: Could not create directory %s: %s\n", runpath, strerror(errno));
            return(255);
        }
    }
    
    //****************************************************************************//
    // Setup namespaces
    //****************************************************************************//

    // Always virtualize our mount namespace
    if ( unshare(CLONE_NEWNS) < 0 ) {
        fprintf(stderr, "ABORT: Could not virtulize mount namespace\n");
        return(255);
    }

    // Privitize the mount namespaces (thank you for the pointer Doug Jacobsen!)
    if ( mount(NULL, "/", NULL, MS_PRIVATE|MS_REC, NULL) < 0 ) {
        // I am not sure if this error needs to be caught, maybe it will fail
        // on older kernels? If so, we can fix then.
        fprintf(stderr, "ABORT: Could not make mountspaces private: %s\n", strerror(errno));
        return(255);
    }


#ifdef NS_CLONE_NEWPID
    if ( getenv("SINGULARITY_NO_NAMESPACE_PID") == NULL ) {
        unsetenv("SINGULARITY_NO_NAMESPACE_PID");
        if ( unshare(CLONE_NEWPID) < 0 ) {
            fprintf(stderr, "ABORT: Could not virtulize PID namespace\n");
            return(255);
        }
    }
#else
#ifdef NS_CLONE_PID
    // This is for older legacy CLONE_PID
    if ( getenv("SINGULARITY_NO_NAMESPACE_PID") == NULL ) {
        unsetenv("SINGULARITY_NO_NAMESPACE_PID");
        if ( unshare(CLONE_PID) < 0 ) {
            fprintf(stderr, "ABORT: Could not virtulize PID namespace\n");
            return(255);
        }
    }
#endif
#endif
#ifdef NS_CLONE_FS
    if ( getenv("SINGULARITY_NO_NAMESPACE_FS") == NULL ) {
        unsetenv("SINGULARITY_NO_NAMESPACE_FS");
        if ( unshare(CLONE_FS) < 0 ) {
            fprintf(stderr, "ABORT: Could not virtulize file system namespace\n");
            return(255);
        }
    }
#endif
#ifdef NS_CLONE_FILES
    if ( getenv("SINGULARITY_NO_NAMESPACE_FILES") == NULL ) {
        unsetenv("SINGULARITY_NO_NAMESPACE_FILES");
        if ( unshare(CLONE_FILES) < 0 ) {
            fprintf(stderr, "ABORT: Could not virtulize file descriptor namespace\n");
            return(255);
        }
    }
#endif

    //****************************************************************************//
    // Mount image
    //****************************************************************************//

    if ( getenv("SINGULARITY_WRITABLE") == NULL ) {
        unsetenv("SINGULARITY_WRITABLE");
        if ( mount_image(containerimage, containerpath, 0) < 0 ) {
            fprintf(stderr, "FAILED: Could not mount image: %s\n", containerimage);
            return(255);
        }
    } else {
        if ( mount_image(containerimage, containerpath, 1) < 0 ) {
            fprintf(stderr, "FAILED: Could not mount image: %s\n", containerimage);
            return(255);
        }
    }

    //****************************************************************************//
    // Fork child in new namespaces
    //****************************************************************************//

    // Drop privledges for fork and parent
    if ( seteuid(uid) < 0 ) {
        fprintf(stderr, "ABORT: Could not drop effective user privledges!\n");
        return(255);
    }

    child_pid = fork();

    if ( child_pid == 0 ) {
        char *nsswitch = joinpath(SYSCONFDIR, "/singularity/default-nsswitch.conf");
        char *prompt;
        char *local_passwd;
        char *container_passwd;
        char *local_group;
        char *container_group;

        nsswitch = (char *) malloc(strlen(SYSCONFDIR) + 36);
        snprintf(nsswitch, strlen(SYSCONFDIR) + 36, "%s/singularity/default-nsswitch.conf", SYSCONFDIR);

        local_passwd = (char *) malloc(strlen(runpath) + 9);
        snprintf(local_passwd, strlen(runpath) + 9, "%s/passwd", runpath);

        container_passwd = (char *) malloc(strlen(containerpath) + 13);
        snprintf(container_passwd, strlen(containerpath) + 13, "%s/etc/passwd", containerpath);

        local_group = (char *) malloc(strlen(runpath) + 8);
        snprintf(local_group, strlen(runpath) + 8, "%s/group", runpath);

        container_group = (char *) malloc(strlen(containerpath) + 12);
        snprintf(container_group, strlen(containerpath) + 12, "%s/etc/group", containerpath);


        prompt = (char *) malloc(strlen(containerimage) + 22);
        if ( uid == 0 ) {
            snprintf(prompt, strlen(containerimage) + 22, "[\\u@Singularity:%s \\W]# ", containername);
        } else {
            snprintf(prompt, strlen(containerimage) + 22, "[\\u@Singularity:%s \\W]$ ", containername);
        }

        setenv("PS1", prompt, 1);

        if ( build_passwd(container_passwd, local_passwd) < 0 ) {
            fprintf(stderr, "ABORT: Failed creating template password file\n");
            return(255);
        }

        if ( build_group(container_group, local_group) < 0 ) {
            fprintf(stderr, "ABORT: Failed creating template group file\n");
            return(255);
        }

        if ( seteuid(0) < 0 ) {
            fprintf(stderr, "ABORT: Could not re-escalate effective user privledges!\n");
            return(255);
        }

        if ( getenv("SINGULARITY_NO_NAMESPACE_ROOTFS") == NULL ) {
            unsetenv("SINGULARITY_NO_NAMESPACE_ROOTFS");

            if ( mount_bind("/dev", joinpath(containerpath, "/dev"), 0) < 0 ) {
                fprintf(stderr, "ABORT: Could not bind mount /dev\n");
                return(255);
            }

            if ( getenv("SINGULARITY_NO_SHARE") == NULL ) {
                unsetenv("SINGULARITY_NO_SHARE");

                if ( getenv("SINGULARITY_NO_SHARE_TMP") == NULL ) {
                    unsetenv("SINGULARITY_NO_SHARE_TMP");
                    if ( mount_bind("/tmp", joinpath(containerpath, "/tmp"), 1) < 0 ) {
                        fprintf(stderr, "ABORT: Could not bind mount /tmp\n");
                        return(255);
                    }
                    if ( mount_bind("/var/tmp", joinpath(containerpath, "/var/tmp"), 1) < 0 ) {
                        fprintf(stderr, "ABORT: Could not bind mount /var/tmp\n");
                        return(255);
                    }
                }

                if ( getenv("SINGULARITY_NO_SHARE_HOME") == NULL ) {
                    unsetenv("SINGULARITY_NO_SHARE_HOME");
                    if ( strncmp(homepath, "/home", 5) == 0 ) {
                        if ( s_mkpath(joinpath(runpath, homepath), S_IRUSR | S_IWUSR | S_IXUSR | S_IRGRP | S_IWGRP | S_IXGRP | S_IROTH | S_IXOTH) < 0 ) {
                            fprintf(stderr, "ABORT: Could not create tmp home dir space at %s: %s\n", joinpath(runpath, homepath), strerror(errno));
                            return(255);
                        }

                        if ( mount_bind(homepath, joinpath(runpath, homepath), 1) < 0 ) {
                            fprintf(stderr, "ABORT: Could not bind mount home to tmphome: %s\n", joinpath(runpath, homepath));
                            return(255);
                        }

                        if ( mount_bind(joinpath(runpath, homepath), joinpath(containerpath, "/home"), 1) < 0 ) {
                            fprintf(stderr, "ABORT: Could not bind mount home dir: %s\n", homepath);
                            return(255);
                        }
                    } else {
                        fprintf(stderr, "ERROR: Could not mount non standard home dir: %s\n", homepath);
                    }
                }

            } else {
                if ( is_dir(homepath) != 0 ) {
                    if ( s_mkpath(homepath, S_IRUSR | S_IWUSR | S_IXUSR | S_IRGRP | S_IWGRP | S_IXGRP | S_IROTH | S_IXOTH) != 0 ) {
                        fprintf(stderr, "ABORT: Could not create directory %s: %s\n", homepath, strerror(errno));
                        return(255);
                    }
                    if ( chown(homepath, uid, gid) != 0 ) {
                        fprintf(stderr, "ABORT: Could not set ownership of home (%s): %s\n", homepath, strerror(errno));
                        return(255);
                    }
                }
                strcpy(cwd, homepath);
            }

            if (is_file(joinpath(containerpath, "/etc/resolv.conf")) == 0 ) {
                if ( mount_bind("/etc/resolv.conf", joinpath(containerpath, "/etc/resolv.conf"), 0) < 0 ) {
                    fprintf(stderr, "ABORT: Could not bind /etc/resolv.conf\n");
                    return(255);
                }
            }
            if (is_file(joinpath(containerpath, "/etc/hosts")) == 0 ) {
                if ( mount_bind("/etc/hosts", joinpath(containerpath, "/etc/hosts"), 0) < 0 ) {
                    fprintf(stderr, "ABORT: Could not bind /etc/hosts\n");
                    return(255);
                }
            }

            if (is_file(joinpath(containerpath, "/etc/passwd")) == 0 ) {
                if ( is_file(container_passwd) == 0 ) {
                    if ( mount_bind(local_passwd, joinpath(containerpath, "/etc/passwd"), 0) < 0 ) {
                        fprintf(stderr, "ABORT: Could not bind /etc/passwd\n");
                        return(255);
                    }
                }
            }
            if (is_file(joinpath(containerpath, "/etc/group")) == 0 ) {
                if ( is_file(container_group) == 0 ) {
                    if ( mount_bind(local_group, joinpath(containerpath, "/etc/group"), 0) < 0 ) {
                        fprintf(stderr, "ABORT: Could not bind /etc/group\n");
                        return(255);
                    }
                }
            }
            if (is_file(joinpath(containerpath, "/etc/nsswitch.conf")) == 0 ) {
                if ( is_file(nsswitch) == 0 ) {
                    if ( mount_bind(nsswitch, joinpath(containerpath, "/etc/nsswitch.conf"), 0) < 0 ) {
                        fprintf(stderr, "ABORT: Could not bind %s\n", nsswitch);
                        return(255);
                    }
                } else {
                    fprintf(stderr, "WARNING: Template /etc/nsswitch.conf does not exist: %s\n", nsswitch);
                }
            }

            if (is_file(joinpath(containerpath, "/etc/mtab")) == 0 ) {
                if ( is_file(joinpath(runpath, "/mtab")) == 0 ) {
                    if ( mount_bind(joinpath(runpath, "/mtab"), joinpath(containerpath, "/etc/mtab"), 0) < 0 ) {
                        fprintf(stderr, "ABORT: Could not bind %s\n", joinpath(runpath, "/mtab"));
                        return(255);
                    }
                } else {
                    fprintf(stderr, "WARNING: Template /etc/mtab does not exist: %s\n", joinpath(runpath, "/mtab"));
                }
            }

            // Do the chroot
            if ( chroot(containerpath) < 0 ) {
                fprintf(stderr, "ABORT: failed enter CONTAINERIMAGE: %s\n", containerpath);
                return(255);
            }

            // Make these, just incase they don't already exist
            if ( is_dir("/proc") != 0 ) {
                if ( s_mkpath("/proc", S_IRUSR | S_IWUSR | S_IXUSR | S_IRGRP | S_IWGRP | S_IXGRP | S_IROTH | S_IXOTH) != 0 ) {
                    fprintf(stderr, "ABORT: Could not create directory /proc: %s\n", strerror(errno));
                    return(255);
                }
            }
            if ( is_dir("/sys") != 0 ) {
                if ( s_mkpath("/sys", S_IRUSR | S_IWUSR | S_IXUSR | S_IRGRP | S_IWGRP | S_IXGRP | S_IROTH | S_IXOTH) != 0 ) {
                    fprintf(stderr, "ABORT: Could not create directory /sys: %s\n", strerror(errno));
                    return(255);
                }
            }

            // Mount up /proc
            if ( mount("proc", "/proc", "proc", 0, NULL) < 0 ) {
                fprintf(stderr, "ABORT: Could not mount /proc: %s\n", strerror(errno));
                return(255);
            }
            // Mount /sys
            if ( mount("sysfs", "/sys", "sysfs", 0, NULL) < 0 ) {
                fprintf(stderr, "ABORT: Could not mount /sys: %s\n", strerror(errno));
                return(255);
            }
        }

        // No more privledge escalation for the child thread
        if ( setregid(gid, gid) < 0 ) {
            fprintf(stderr, "ABORT: Could not dump real and effective group privledges!\n");
            return(255);
        }
        if ( setreuid(uid, uid) < 0 ) {
            fprintf(stderr, "ABORT: Could not dump real and effective user privledges!\n");
            return(255);
        }

        // After this, we exist only within the container... Let's make it known!
        if ( setenv("SINGULARITY_CONTAINER", "true", 0) != 0 ) {
            fprintf(stderr, "ABORT: Could not set SINGULARITY_CONTAINER to 'true'\n");
            return(1);
        }

        if ( is_dir(cwd) == 0 ) {
            if ( chdir(cwd) < 0 ) {
                fprintf(stderr, "ABORT: Could not chdir to: %s\n", cwd);
                return(1);
            }
        } else {
            if ( fchdir(cwd_fd) < 0 ) {
                fprintf(stderr, "ABORT: Could not fchdir to cwd\n");
                return(1);
            }
        }

        if ( command == NULL ) {
            fprintf(stderr, "No command specified, launching 'shell'\n");
            argv[0] = strdup("/bin/sh");
            if ( execv("/bin/sh", argv) != 0 ) {
                fprintf(stderr, "ABORT: exec of /bin/sh failed: %s\n", strerror(errno));
            }
        } else if ( strcmp(command, "run") == 0 ) {
            if ( is_exec("/singularity") == 0 ) {
                argv[0] = strdup("/singularity");
                if ( execv("/singularity", argv) != 0 ) {
                    fprintf(stderr, "ABORT: exec of /bin/sh failed: %s\n", strerror(errno));
                }
            } else {
                fprintf(stderr, "No Singularity runscript found, launching 'shell'\n");
                argv[0] = strdup("/bin/sh");
                if ( execv("/bin/sh", argv) != 0 ) {
                    fprintf(stderr, "ABORT: exec of /bin/sh failed: %s\n", strerror(errno));
                }
            }
        } else if ( strcmp(command, "shell") == 0 ) {
            argv[0] = strdup("/bin/sh");
            if ( execv("/bin/sh", argv) != 0 ) {
                fprintf(stderr, "ABORT: exec of /bin/sh failed: %s\n", strerror(errno));
            }
        } else if ( strcmp(command, "exec") == 0 ) {
            if ( command_exec != NULL ) {
                argv[0] = strdup(command_exec);
                if ( execv(command_exec, argv) != 0 ) {
                    fprintf(stderr, "ABORT: exec of '%s' failed: %s\n", command_exec, strerror(errno));
                }
            } else {
                fprintf(stderr, "ABORT: no command given to execute\n");
                return(1);
            }
        } else {
            fprintf(stderr, "ABORT: Unrecognized Singularity command: %s\n", command);
            return(1);
        }

        return(255);

    } else if ( child_pid > 0 ) {
        int tmpstatus;
        signal(SIGINT, sighandler);
        signal(SIGKILL, sighandler);
        signal(SIGQUIT, sighandler);

        waitpid(child_pid, &tmpstatus, 0);
        retval = WEXITSTATUS(tmpstatus);
    } else {
        fprintf(stderr, "ABORT: Could not fork child process\n");
        return(255);
    }

    if ( close(cwd_fd) < 0 ) {
        fprintf(stderr, "ERROR: Could not close cwd_fd!\n");
        retval++;
    }

    free(containerpath);

    return(retval);
}
