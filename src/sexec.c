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
#include <errno.h> 
#include <signal.h>
#include <sched.h>
#include <string.h>
#include <fcntl.h>  
#include <grp.h>
#include <libgen.h>

#include "config.h"
#include "mounts.h"
#include "loop-control.h"
#include "util.h"
#include "file.h"
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

#ifndef MS_PRIVATE
#define MS_PRIVATE (1<<18)
#endif
#ifndef MS_REC
#define MS_REC 16384
#endif


pid_t namespace_fork_pid = 0;
pid_t exec_fork_pid = 0;


void sighandler(int sig) {
    signal(sig, sighandler);

    printf("Caught signal: %d\n", sig);
    fflush(stdout);

    if ( exec_fork_pid > 0 ) {
        fprintf(stderr, "Singularity is sending SIGKILL to child pid: %d\n", exec_fork_pid);

        kill(exec_fork_pid, SIGKILL);
    }
    if ( namespace_fork_pid > 0 ) {
        fprintf(stderr, "Singularity is sending SIGKILL to child pid: %d\n", namespace_fork_pid);

        kill(namespace_fork_pid, SIGKILL);
    }
}


int main(int argc, char ** argv) {
    char *containerimage;
    char *containername;
    char *containerpath;
    char *homepath;
    char *command;
    char *tmpdir;
    char *lockfile;
    char *loop_dev_cache;
    char *loop_dev = 0;
    char *basehomepath;
    char cwd[PATH_MAX];
    int cwd_fd;
    int tmpdirlock_fd;
    int containerimage_fd;
    int loop_fd = -1;
    int lockfile_fd;
    int retval = 0;
    int bind_mount_writable = 0;
    uid_t uid = getuid();
    gid_t gid = getgid();


//****************************************************************************//
// Init
//****************************************************************************//

    signal(SIGINT, sighandler);
    signal(SIGKILL, sighandler);
    signal(SIGQUIT, sighandler);

    // Lets start off as the calling UID
    if ( seteuid(uid) < 0 ) {
        fprintf(stderr, "ABORT: Could not set effective user privledges to %d!\n", uid);
        return(255);
    }
    if ( setegid(gid) < 0 ) {
        fprintf(stderr, "ABORT: Could not set effective group privledges to %d!\n", gid);
        return(255);
    }

    homepath = getenv("HOME");
    containerimage = getenv("SINGULARITY_IMAGE");
    command = getenv("SINGULARITY_COMMAND");

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

    if ( is_dir(homepath) != 0 ) {
        fprintf(stderr, "ABORT: Home directory not found: %s\n", homepath);
        return(1);
    }

    if ( is_owner(homepath, uid) != 0 ) {
        fprintf(stderr, "ABORT: You don't own your own home directory!?: %s\n", homepath);
        return(1);
    }

    // TODO: Offer option to only run containers owned by root (so root can approve
    // containers)
    if ( is_owner(containerimage, uid) < 0 && is_owner(containerimage, 0) < 0 ) {
        fprintf(stderr, "ABORT: Will not execute in a CONTAINERIMAGE you (or root) does not own: %s\n", containerimage);
        return(255);
    }

    containername = basename(strdup(containerimage));
    basehomepath = strjoin("/", strtok(strdup(homepath), "/"));

    containerpath = (char *) malloc(strlen(LOCALSTATEDIR) + 18);
    snprintf(containerpath, strlen(LOCALSTATEDIR) + 18, "%s/singularity/mnt", LOCALSTATEDIR);

    tmpdir = strjoin("/tmp/.singularity-", file_id(containerimage));
    lockfile = joinpath(tmpdir, "lock");
    loop_dev_cache = joinpath(tmpdir, "loop_dev");


//****************************************************************************//
// Setup
//****************************************************************************//

    if ( s_mkpath(tmpdir, 0750) < 0 ) {
        fprintf(stderr, "ABORT: Could not temporary directory %s: %s\n", tmpdir, strerror(errno));
        return(255);
    }

    tmpdirlock_fd = open(tmpdir, O_RDONLY);
    if ( tmpdirlock_fd < 0 ) {
        fprintf(stderr, "ERROR: Could not create lock file %s: %s\n", lockfile, strerror(errno));
        return(255);
    }
    if ( flock(tmpdirlock_fd, LOCK_SH | LOCK_NB) < 0 ) {
        fprintf(stderr, "ERROR: Could not obtain shared lock on %s: %s\n", lockfile, strerror(errno));
        return(255);
    }

    if ( ( lockfile_fd = open(lockfile, O_CREAT | O_RDWR, 0644) ) < 0 ) {
        fprintf(stderr, "ERROR: Could not open lockfile %s: %s\n", lockfile, strerror(errno));
        return(255);
    }

    if ( getenv("SINGULARITY_WRITABLE") == NULL ) {
        if ( ( containerimage_fd = open(containerimage, O_RDONLY) ) < 0 ) {
            fprintf(stderr, "ERROR: Could not open image for reading %s: %s\n", containerimage, strerror(errno));
            return(255);
        }
        if ( flock(containerimage_fd, LOCK_SH | LOCK_NB) < 0 ) {
            fprintf(stderr, "ABORT: Image is locked by another process\n");
            return(5);
        }
    } else {
        if ( ( containerimage_fd = open(containerimage, O_RDWR) ) < 0 ) {
            fprintf(stderr, "ERROR: Could not open image for writing %s: %s\n", containerimage, strerror(errno));
            return(255);
        }
        if ( flock(containerimage_fd, LOCK_EX | LOCK_NB) < 0 ) {
            fprintf(stderr, "ABORT: Image is locked by another process\n");
            return(5);
        }
    }

    // When we contain, we need temporary directories for what should be writable
    if ( getenv("SINGULARITY_CONTAIN") != NULL ) {
        if ( s_mkpath(joinpath(tmpdir, homepath), 0750) < 0 ) {
            fprintf(stderr, "ABORT: Failed creating temporary directory %s: %s\n", joinpath(tmpdir, homepath), strerror(errno));
            return(255);
        }
        if ( s_mkpath(joinpath(tmpdir, "/tmp"), 0750) < 0 ) {
            fprintf(stderr, "ABORT: Failed creating temporary directory %s: %s\n", joinpath(tmpdir, "/tmp"), strerror(errno));
            return(255);
        }
    }


//****************************************************************************//
// We are now running with escalated privleges until we exec
//****************************************************************************//

    if ( seteuid(0) < 0 ) {
        fprintf(stderr, "ABORT: Could not escalate effective user privledges!\n");
        return(255);
    }
    if ( setegid(0) < 0 ) {
        fprintf(stderr, "ABORT: Could not escalate effective group privledges!\n");
        return(255);
    }


    if ( is_dir(containerpath) < 0 ) {
        if ( s_mkpath(containerpath, 0755) < 0 ) {
            fprintf(stderr, "ABORT: Could not create directory %s: %s\n", containerpath, strerror(errno));
            return(255);
        }
    }

    if ( flock(lockfile_fd, LOCK_EX | LOCK_NB) == 0 ) {
        loop_dev = obtain_loop_dev();

        if ( ( loop_fd = open(loop_dev, O_RDWR) ) < 0 ) {
            fprintf(stderr, "ERROR: Failed to open loop device %s: %s\n", loop_dev, strerror(errno));
            return(255);
        }

        if ( associate_loop(containerimage_fd, loop_fd) < 0 ) {
            fprintf(stderr, "ERROR: Could not associate %s to loop device %s\n", containerimage, loop_dev);
            return(255);
        }

        if ( fileput(loop_dev_cache, loop_dev) < 0 ) {
            fprintf(stderr, "ERROR: Could not write to loop_dev_cache %s: %s\n", loop_dev_cache, strerror(errno));
            return(255);
        }

        flock(lockfile_fd, LOCK_SH | LOCK_NB);

    } else {
        flock(lockfile_fd, LOCK_SH);
        if ( ( loop_dev = filecat(loop_dev_cache) ) == NULL ) {
            fprintf(stderr, "ERROR: Could not retrieve loop_dev_cache from %s\n", loop_dev_cache);
            return(255);
        }

        if ( ( loop_fd = open(loop_dev, O_RDWR) ) < 0 ) {
            fprintf(stderr, "ERROR: Failed to open loop device %s: %s\n", loop_dev, strerror(errno));
            return(255);
        }
    }



//****************************************************************************//
// Management fork
//****************************************************************************//

    namespace_fork_pid = fork();
    if ( namespace_fork_pid == 0 ) {

//****************************************************************************//
// Setup namespaces
//****************************************************************************//

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


//****************************************************************************//
// Temporary file generation
//****************************************************************************//

        if ( is_file(joinpath(tmpdir, "/passwd")) < 0 ) {
            if ( build_passwd(joinpath(containerpath, "/etc/passwd"), joinpath(tmpdir, "/passwd")) < 0 ) {
                fprintf(stderr, "ABORT: Failed creating template password file\n");
                return(255);
            }
        }

        if ( is_file(joinpath(tmpdir, "/group")) < 0 ) {
            if ( build_group(joinpath(containerpath, "/etc/group"), joinpath(tmpdir, "/group")) < 0 ) {
                fprintf(stderr, "ABORT: Failed creating template group file\n");
                return(255);
            }
        }

        if ( is_file(joinpath(tmpdir, "/resolv.conf")) < 0 ) {
            if ( copy_file("/etc/resolv.conf", joinpath(tmpdir, "/resolv.conf")) < 0 ) {
                fprintf(stderr, "ABORT: Failed copying temporary resolv.conf\n");
                return(255);
            }
        }

        if ( is_file(joinpath(tmpdir, "/hosts")) < 0 ) {
            if ( copy_file("/etc/hosts", joinpath(tmpdir, "/hosts")) < 0 ) {
                fprintf(stderr, "ABORT: Failed copying temporary hosts\n");
                return(255);
            }
        }

        if ( is_file(joinpath(tmpdir, "/nsswitch.conf")) < 0 ) {
            if ( is_file(joinpath(SYSCONFDIR, "/singularity/default-nsswitch.conf")) == 0 ) {
                if ( copy_file(joinpath(SYSCONFDIR, "/singularity/default-nsswitch.conf"), joinpath(tmpdir, "/nsswitch.conf")) < 0 ) {
                    fprintf(stderr, "ABORT: Failed copying temporary nsswitch.conf\n");
                    return(255);
                }
            } else {
                fprintf(stderr, "WARNING: Template /etc/nsswitch.conf does not exist: %s\n", joinpath(SYSCONFDIR, "/singularity/default-nsswitch.conf"));
            }
        }


//****************************************************************************//
// Bind mounts
//****************************************************************************//

        if ( getenv("SINGULARITY_CONTAIN") == NULL ) {
            unsetenv("SINGULARITY_CONTAIN");

            bind_mount_writable = 1;

            if ( is_dir(joinpath(containerpath, "/tmp")) == 0 ) {
                if ( mount_bind("/tmp", joinpath(containerpath, "/tmp"), 1) < 0 ) {
                    fprintf(stderr, "ABORT: Could not bind mount /tmp\n");
                    return(255);
                }
            }
            if ( is_dir(joinpath(containerpath, "/var/tmp")) == 0 ) {
                if ( mount_bind("/var/tmp", joinpath(containerpath, "/var/tmp"), 1) < 0 ) {
                    fprintf(stderr, "ABORT: Could not bind mount /var/tmp\n");
                    return(255);
                }
            }
            if ( is_dir(joinpath(containerpath, basehomepath)) == 0 ){
                if ( mount_bind(basehomepath, joinpath(containerpath, basehomepath), 1) < 0 ) {
                    fprintf(stderr, "ABORT: Could not bind home path to container %s: %s\n", homepath, strerror(errno));
                    return(255);
                }
            }
        } else {

            if ( is_dir(joinpath(containerpath, "/tmp")) == 0 ) {
                if ( mount_bind(joinpath(tmpdir, "/tmp"), joinpath(containerpath, "/tmp"), 1) < 0 ) {
                    fprintf(stderr, "ABORT: Could not bind tmp path to container %s: %s\n", "/tmp", strerror(errno));
                    return(255);
                }
            }
            if ( is_dir(joinpath(containerpath, "/var/tmp")) == 0 ) {
                if ( mount_bind(joinpath(tmpdir, "/tmp"), joinpath(containerpath, "/var/tmp"), 1) < 0 ) {
                    fprintf(stderr, "ABORT: Could not bind tmp path to container %s: %s\n", "/var/tmp", strerror(errno));
                    return(255);
                }
            }
            if ( is_dir(joinpath(containerpath, basehomepath)) == 0 ){
                if ( mount_bind(joinpath(tmpdir, basehomepath), joinpath(containerpath, basehomepath), 1) < 0 ) {
                    fprintf(stderr, "ABORT: Could not bind tmp path to container %s: %s\n", basehomepath, strerror(errno));
                    return(255);
                }
            }
            strcpy(cwd, homepath);
        }

        if ( is_dir(joinpath(containerpath, "/dev/")) == 0 ) {
            if ( mount_bind("/dev", joinpath(containerpath, "/dev"), bind_mount_writable) < 0 ) {
                fprintf(stderr, "ABORT: Could not bind mount /dev\n");
                return(255);
            }
        }

        if (is_file(joinpath(containerpath, "/etc/resolv.conf")) == 0 ) {
            if ( mount_bind(joinpath(tmpdir, "/resolv.conf"), joinpath(containerpath, "/etc/resolv.conf"), bind_mount_writable) < 0 ) {
                fprintf(stderr, "ABORT: Could not bind /etc/resolv.conf\n");
                return(255);
            }
        }

        if (is_file(joinpath(containerpath, "/etc/hosts")) == 0 ) {
            if ( mount_bind(joinpath(tmpdir, "hosts"), joinpath(containerpath, "/etc/hosts"), bind_mount_writable) < 0 ) {
                fprintf(stderr, "ABORT: Could not bind /etc/hosts\n");
                return(255);
            }
        }

        if (is_file(joinpath(containerpath, "/etc/passwd")) == 0 ) {
            if ( mount_bind(joinpath(tmpdir, "/passwd"), joinpath(containerpath, "/etc/passwd"), bind_mount_writable) < 0 ) {
                fprintf(stderr, "ABORT: Could not bind /etc/passwd\n");
                return(255);
            }
        }

        if (is_file(joinpath(containerpath, "/etc/group")) == 0 ) {
            if ( mount_bind(joinpath(tmpdir, "/group"), joinpath(containerpath, "/etc/group"), bind_mount_writable) < 0 ) {
                fprintf(stderr, "ABORT: Could not bind /etc/group\n");
                return(255);
            }
        }

        if (is_file(joinpath(containerpath, "/etc/nsswitch.conf")) == 0 ) {
            if ( mount_bind(joinpath(tmpdir, "/nsswitch.conf"), joinpath(containerpath, "/etc/nsswitch.conf"), bind_mount_writable) != 0 ) {
                fprintf(stderr, "ABORT: Could not bind /etc/nsswitch.conf\n");
                return(255);
            }
        }


//****************************************************************************//
// Fork child in new namespaces
//****************************************************************************//

        exec_fork_pid = fork();

        if ( exec_fork_pid == 0 ) {
            char *prompt;


//****************************************************************************//
// Enter the file system
//****************************************************************************//

            if ( chroot(containerpath) < 0 ) {
                fprintf(stderr, "ABORT: failed enter CONTAINERIMAGE: %s\n", containerpath);
                return(255);
            }


//****************************************************************************//
// Setup real mounts within the container
//****************************************************************************//

            if ( is_dir("/proc") == 0 ) {
                if ( mount("proc", "/proc", "proc", 0, NULL) < 0 ) {
                    fprintf(stderr, "ABORT: Could not mount /proc: %s\n", strerror(errno));
                    return(255);
                }
            }
            if ( is_dir("/sys") == 0 ) {
                if ( mount("sysfs", "/sys", "sysfs", 0, NULL) < 0 ) {
                    fprintf(stderr, "ABORT: Could not mount /sys: %s\n", strerror(errno));
                    return(255);
                }
            }


//****************************************************************************//
// Drop all privledges for good
//****************************************************************************//

            if ( setregid(gid, gid) < 0 ) {
                fprintf(stderr, "ABORT: Could not dump real and effective group privledges!\n");
                return(255);
            }
            if ( setreuid(uid, uid) < 0 ) {
                fprintf(stderr, "ABORT: Could not dump real and effective user privledges!\n");
                return(255);
            }


//****************************************************************************//
// Setup final envrionment
//****************************************************************************//

            prompt = (char *) malloc(strlen(containername) + 16);
            snprintf(prompt, strlen(containerimage) + 16, "Singularity/%s> ", containername);
            setenv("PS1", prompt, 1);

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


//****************************************************************************//
// Execv to container process
//****************************************************************************//

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

            } else if ( strcmp(command, "exec") == 0 ) {

                if ( execvp(argv[1], &argv[1]) != 0 ) {
                    fprintf(stderr, "ABORT: execvp of '%s' failed: %s\n", argv[1], strerror(errno));
                }

            } else if ( strcmp(command, "shell") == 0 ) {
                if ( is_exec("/bin/bash") == 0 ) {
                    char *args[argc+1];
                    int i;

                    args[0] = strdup("/bin/bash");
                    args[1] = strdup("--norc");
                    for(i=1; i<=argc; i++) {
                        args[i+1] = argv[i];
                    }

                    if ( execv("/bin/bash", args) != 0 ) {
                        fprintf(stderr, "ABORT: exec of /bin/bash failed: %s\n", strerror(errno));
                    }
                } else {
                    argv[0] = strdup("/bin/sh");
                    if ( execv("/bin/sh", argv) != 0 ) {
                        fprintf(stderr, "ABORT: exec of /bin/sh failed: %s\n", strerror(errno));
                    }
                }

            } else {
                fprintf(stderr, "ABORT: Unrecognized Singularity command: %s\n", command);
                return(1);
            }

            return(255);


//****************************************************************************//
// Outer child waits for inner child
//****************************************************************************//

        } else if ( exec_fork_pid > 0 ) {
            int tmpstatus;

            strncpy(argv[0], "Singularity: exec", strlen(argv[0]));

            if ( seteuid(uid) < 0 ) {
                fprintf(stderr, "ABORT: Could not set effective user privledges to %d!\n", uid);
                return(255);
            }

            waitpid(exec_fork_pid, &tmpstatus, 0);
            retval = WEXITSTATUS(tmpstatus);
        } else {
            fprintf(stderr, "ABORT: Could not fork namespace process\n");
            return(255);
        }
        return(retval);

    } else if ( namespace_fork_pid > 0 ) {
        int tmpstatus;

        strncpy(argv[0], "Singularity: namespace", strlen(argv[0]));

        if ( seteuid(uid) < 0 ) {
            fprintf(stderr, "ABORT: Could not set effective user privledges to %d!\n", uid);
            return(255);
        }

        waitpid(namespace_fork_pid, &tmpstatus, 0);
        retval = WEXITSTATUS(tmpstatus);
    } else {
        fprintf(stderr, "ABORT: Could not fork management process\n");
        return(255);
    }


//****************************************************************************//
// Finall wrap up before exiting
//****************************************************************************//

    if ( close(cwd_fd) < 0 ) {
        fprintf(stderr, "ERROR: Could not close cwd_fd!\n");
        retval++;
    }

    if ( flock(tmpdirlock_fd, LOCK_EX | LOCK_NB) == 0 ) {
        close(tmpdirlock_fd);
        if ( seteuid(0) < 0 ) {
            fprintf(stderr, "ABORT: Could not re-escalate effective user privledges!\n");
            return(255);
        }

        if ( s_rmdir(tmpdir) < 0 ) {
            fprintf(stderr, "WARNING: Could not remove all files in %s: %s\n", tmpdir, strerror(errno));
        }
    
        // Dissociate loops from here Just incase autoflush didn't work.
        (void)disassociate_loop(loop_fd);

        if ( seteuid(uid) < 0 ) {
            fprintf(stderr, "ABORT: Could not drop effective user privledges!\n");
            return(255);
        }

    } else {
//        printf("Not removing tmpdir, lock still\n");
    }

    close(containerimage_fd);
    close(tmpdirlock_fd);

    free(lockfile);
    free(containerpath);
    free(tmpdir);

    return(retval);
}
