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

#include "config.h"
#include "util.h"
#include "mounts.h"


#ifndef LIBEXECDIR
#define LIBEXECDIR "undefined"
#endif
#ifndef SYSCONFDIR
#define SYSCONFDIR "/etc"
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


int main(int argc, char **argv) {
    char *containerimage;
    char *homepath;
    char cwd[PATH_MAX];
    int cwd_fd;
//    int opt_contain = 0;
    int retval = 0;
    uid_t uid = getuid();
    gid_t gid = getgid();
    mode_t initial_umask = umask(0);

char containerpath[5] = "/mnt\0";


    //****************************************************************************//
    // Sanity
    //****************************************************************************//

//    // We don't run as root!
//    if ( uid == 0 || gid == 0 ) {
//        fprintf(stderr, "ERROR: Do not run singularities as root!\n");
//        return(255);
//    }

    // Lets start off and confirm non-root
    if ( seteuid(uid) < 0 ) {
        fprintf(stderr, "ERROR: Could not set effective user privledges to %d!\n", uid);
        return(255);
    }

//    // Check for SINGULARITY_CONTAIN environment variable
//    if ( getenv("SINGULARITY_CONTAIN") != NULL ) {
//        opt_contain = 1;
//    }

    // Figure out where we start
    if ( (cwd_fd = open(".", O_RDONLY)) < 0 ) {
        fprintf(stderr, "ERROR: Could not open cwd fd (%s)!\n", strerror(errno));
        return(1);
    }
    if ( getcwd(cwd, PATH_MAX) == NULL ) {
        fprintf(stderr, "Could not obtain current directory path\n");
        return(1);
    }


    //****************************************************************************//
    // Sanity
    //****************************************************************************//

    homepath = getenv("HOME");
    // Get containerimage from the environment (we check on this shortly)
    containerimage = getenv("SINGULARITY_IMAGE");

    // Check CONTAINERIMAGE
    if ( containerimage == NULL ) {
        fprintf(stderr, "ERROR: SINGULARITY_IMAGE undefined!\n");
        return(1);
    }
    if ( s_is_dir(containerpath) < 0 ) {
        fprintf(stderr, "ERROR: Container path is not a directory: %s!\n", containerpath);
        return(1);
    }
    // TODO: Offer option to only run containers owned by root (so root can approve
    // containers)
//    if ( s_is_owner(containerimage, uid) < 0 ) {
//        fprintf(stderr, "ERROR: Will not execute in a CONTAINERIMAGE you don't own: %s\n", containerimage);
//        return(255);
//    }

    
    // Check the singularity within the CONTAINERPATH
//    singularitypath = (char *) malloc(strlen(containerimage) + 13);
//    snprintf(singularitypath, strlen(containerimage) + 13, "%s/singularity", containerimage);
//    if ( s_is_file(singularitypath) < 0 ) {
//        fprintf(stderr, "ERROR: The singularity is not found in CONTAINERPATH!\n");
//        return(1);
//    }
//    if ( s_is_owner(singularitypath, uid) < 0 ) {
//        fprintf(stderr, "ERROR: Will not execute a singularity you don't own: %s!\n", singularitypath);
//        return(255);
//    }
//    if ( s_is_exec(singularitypath) < 0 ) {
//        fprintf(stderr, "ERROR: The singularity can not be executed!\n");
//        return(1);
//    }


//    // Get the scratch path from the environment and setup
//    scratchpath = getenv("SINGULARITY_SCRATCH");
//    if ( scratchpath != NULL ) {
//        if ( ( strncmp(homepath, scratchpath, strlen(homepath)) == 0 ) || ( strncmp(homepath, scratchpath, strlen(scratchpath)) == 0 ) ) {
//            fprintf(stderr, "ERROR: Overlapping paths (scratch and home)!\n");
//            return(255);
//        }
//        if ( strncmp(scratchpath, "/lib", 4) == 0 ) {
//            fprintf(stderr, "ERROR: Can not link scratch directory over /lib\n");
//            return(255);
//        }
//        if ( strncmp(scratchpath, "/bin", 4) == 0 ) {
//            fprintf(stderr, "ERROR: Can not link scratch directory over /bin\n");
//            return(255);
//        }
//        if ( strncmp(scratchpath, "/sbin", 4) == 0 ) {
//            fprintf(stderr, "ERROR: Can not link scratch directory over /sbin\n");
//            return(255);
//        }
//        if ( strncmp(scratchpath, "/etc", 4) == 0 ) {
//            fprintf(stderr, "ERROR: Can not link scratch directory over /etc\n");
//            return(255);
//        }
//
//        containerscratchpath = (char *) malloc(strlen(containerimage) + strlen(scratchpath) + 1);
//        snprintf(containerscratchpath, strlen(containerimage) + strlen(scratchpath) + 1, "%s%s", containerimage, scratchpath);
//        if ( s_is_dir(scratchpath) == 0 ) {
//            if ( s_is_dir(containerscratchpath) < 0 ) {
//                if ( s_mkpath(containerscratchpath, S_IRUSR | S_IWUSR | S_IXUSR | S_IRGRP | S_IWGRP | S_IROTH | S_IXOTH) > 0 ) {
//                    fprintf(stderr, "ERROR: Could not create directory %s\n", scratchpath);
//                    return(255);
//                }
//            }
//        } else {
//            fprintf(stderr, "WARNING: Could not locate your scratch directory (%s), not linking to container.\n", scratchpath);
//            scratchpath = NULL;
//        }
//    }
//

    // Reset umask back to where we have started
    umask(initial_umask);


    //****************************************************************************//
    // Do root bits
    //****************************************************************************//

    // Entering danger zone
    if ( seteuid(0) < 0 ) {
        fprintf(stderr, "ERROR: Could not escalate effective user privledges!\n");
        return(255);
    }

    // Separate out the appropriate namespaces

//#ifdef NS_CLONE_NEWNS
    // Always virtualize our mount namespace
    if ( unshare(CLONE_NEWNS) < 0 ) {
        fprintf(stderr, "ERROR: Could not virtulize mount namespace\n");
        return(255);
    }

    // Privitize the mount namespaces (thank you for the pointer Doug Jacobsen!)
    if ( mount(NULL, "/", NULL, MS_PRIVATE|MS_REC, NULL) < 0 ) {
        // I am not sure if this error needs to be caught, maybe it will fail
        // on older kernels? If so, we can fix then.
        fprintf(stderr, "ERROR: Could not make mountspaces private: %s\n", strerror(errno));
        return(255);
    }
//#endif

    if ( getenv("SINGULARITY_WRITABLE") == NULL ) {
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


//#ifdef NS_CLONE_NEWPID
    if ( getenv("SINGULARITY_NO_NAMESPACE_PID") == NULL ) {
        if ( unshare(CLONE_NEWPID) < 0 ) {
            fprintf(stderr, "ERROR: Could not virtulize PID namespace\n");
            return(255);
        }
    }
//#else
//#ifdef NS_CLONE_PID
//    // This is for older legacy CLONE_PID
//    if ( getenv("SINGULARITY_NO_NAMESPACE_PID") == NULL ) {
//        if ( unshare(CLONE_PID) < 0 ) {
//            fprintf(stderr, "ERROR: Could not virtulize PID namespace\n");
//            return(255);
//        }
//    }
//#endif
//#endif
//#ifdef NS_CLONE_FS
    if ( getenv("SINGULARITY_NO_NAMESPACE_FS") == NULL ) {
        if ( unshare(CLONE_FS) < 0 ) {
            fprintf(stderr, "ERROR: Could not virtulize file system namespace\n");
            return(255);
        }
    }
//#endif
//#ifdef NS_CLONE_FILES
    if ( getenv("SINGULARITY_NO_NAMESPACE_FILES") == NULL ) {
        if ( unshare(CLONE_FILES) < 0 ) {
            fprintf(stderr, "ERROR: Could not virtulize file descriptor namespace\n");
            return(255);
        }
    }
//#endif

    // Drop privledges for fork and parent
    if ( seteuid(uid) < 0 ) {
        fprintf(stderr, "ERROR: Could not drop effective user privledges!\n");
        return(255);
    }

    child_pid = fork();

    if ( child_pid == 0 ) {
        char * mtab;
        char * prompt;
        char * container_name = basename(strdup(containerimage));

        mtab = (char *) malloc(strlen(SYSCONFDIR) + 27);
        snprintf(mtab, strlen(SYSCONFDIR) + 27, "%s/singularity/default-mtab", SYSCONFDIR);

        prompt = (char *) malloc(strlen(container_name) + 4);
        snprintf(prompt, strlen(container_name) + 4, "%s> ", container_name);

        setenv("PS1", prompt, 1);

        if ( getenv("SINGULARITY_NOCHROOT") == NULL ) {

            // Root needed for chroot and /proc mount
            if ( seteuid(0) < 0 ) {
                fprintf(stderr, "ERROR: Could not re-escalate effective user privledges!\n");
                return(255);
            }


            if ( mount_bind(containerpath, "/dev", "/dev", 0) < 0 ) {
                fprintf(stderr, "ERROR: Could not bind mount /dev\n");
                return(255);
            }
            if ( mount_bind(containerpath, "/tmp", "/tmp", 1) < 0 ) {
                fprintf(stderr, "ERROR: Could not bind mount /dev\n");
                return(255);
            }
            if ( mount_bind(containerpath, homepath, homepath, 1) < 0 ) {
                fprintf(stderr, "ERROR: Could not bind mount home dir: %s\n", homepath);
                return(255);
            }
            if ( mount_bind(containerpath, "/etc/resolv.conf", "/etc/resolv.conf", 0) < 0 ) {
                fprintf(stderr, "ERROR: Could not bind /etc/resolv.conf\n");
                return(255);
            }
            if ( mount_bind(containerpath, "/etc/passwd", "/etc/passwd", 0) < 0 ) {
                fprintf(stderr, "ERROR: Could not bind /etc/passwd\n");
                return(255);
            }
            if ( mount_bind(containerpath, "/etc/group", "/etc/group", 0) < 0 ) {
                fprintf(stderr, "ERROR: Could not bind /etc/group\n");
                return(255);
            }
            if ( mount_bind(containerpath, "/etc/hosts", "/etc/hosts", 0) < 0 ) {
                fprintf(stderr, "ERROR: Could not bind /etc/hosts\n");
                return(255);
            }
            if ( s_is_file(mtab) == 0 ) {
                if ( mount_bind(containerpath, mtab, "/etc/mtab", 0) < 0 ) {
                    fprintf(stderr, "ERROR: Could not bind %s\n", mtab);
                    return(255);
                }
            } else {
                fprintf(stderr, "WARNING: Could not open %s\n", mtab);
            }


            // Do the chroot
            if ( chroot(containerpath) < 0 ) {
                fprintf(stderr, "ERROR: failed enter CONTAINERIMAGE: %s\n", containerpath);
                return(255);
            }

//#ifdef NS_CLONE_NEWNS
            // Mount up /proc
            if ( mount("proc", "/proc", "proc", 0, NULL) < 0 ) {
                fprintf(stderr, "ERROR: Could not mount /proc: %s\n", strerror(errno));
                return(255);
            }
            // Mount /sys
            if ( mount("sysfs", "/sys", "sysfs", 0, NULL) < 0 ) {
                fprintf(stderr, "ERROR: Could not mount /sys: %s\n", strerror(errno));
                return(255);
            }

//TODO: Create, and update mtab.

//#endif
        }

        // Dump all privs permanently for this process
        if ( setregid(gid, gid) < 0 ) {
            fprintf(stderr, "ERROR: Could not dump real and effective group privledges!\n");
            return(255);
        }
        if ( setreuid(uid, uid) < 0 ) {
            fprintf(stderr, "ERROR: Could not dump real and effective user privledges!\n");
            return(255);
        }

//        // Confirm we no longer have any escalated privledges whatsoever
//        if ( setuid(0) == 0 ) {
//            fprintf(stderr, "ERROR: Root not allowed here!\n");
//            return(1);
//        }
//
//        // change directory back to starting point if needed
//        if ( opt_contain > 0 ) {
//            if ( chdir("/") < 0 ) {
//                fprintf(stderr, "ERROR: Could not changedir to /\n");
//                return(1);
//            }
//        } else {
//            if (strncmp(homepath, cwd, strlen(homepath)) == 0 ) {
//                if ( chdir(cwd) < 0 ) {
//                    fprintf(stderr, "ERROR: Could not chdir!\n");
//                    return(1);
//                }
//            } else {
//                if ( fchdir(cwd_fd) < 0 ) {
//                    fprintf(stderr, "ERROR: Could not fchdir!\n");
//                    return(1);
//                }
//            }
//        }





        // After this, we exist only within the container... Let's make it known!
        if ( setenv("SINGULARITY_CONTAINER", "true", 0) != 0 ) {
            fprintf(stderr, "ERROR: Could not set SINGULARITY_CONTAINER to 'true'\n");
            return(1);
        }

        if (strncmp(homepath, cwd, strlen(homepath)) == 0 ) {
            if ( chdir(cwd) < 0 ) {
                fprintf(stderr, "ERROR: Could not chdir!\n");
               return(1);
            }
        } else {
            if ( fchdir(cwd_fd) < 0 ) {
                fprintf(stderr, "ERROR: Could not fchdir!\n");
                return(1);
            }
        }

        if ( argv[1] != NULL && strcmp(argv[1], "shell") == 0) {
            execv("/bin/sh", &argv[1]);
        } else if ( s_is_exec("/singularity") == 0 ) {
            execv("/singularity", argv);
        } else {
            printf("No command specified, launching /bin/sh\n");
            execv("/bin/sh", argv);
        }

    } else if ( child_pid > 0 ) {
        int tmpstatus;
        signal(SIGINT, sighandler);
        signal(SIGKILL, sighandler);
        signal(SIGQUIT, sighandler);

        waitpid(child_pid, &tmpstatus, 0);
        retval = WEXITSTATUS(tmpstatus);
    } else {
        fprintf(stderr, "ERROR: Could not fork child process\n");
        retval++;
    }

    if ( close(cwd_fd) < 0 ) {
        fprintf(stderr, "ERROR: Could not close cwd_fd!\n");
        retval++;
    }

//#ifndef NS_CLONE_NEWNS
//    // If we did not create a new mount namespace, unmount as needed
//
//    // Root needed again to umount
//    if ( seteuid(0) < 0 ) {
//        fprintf(stderr, "ERROR: Could not re-escalate effective user privledges!\n");
//        return(255);
//    }
//
//    chdir("/");
//
//    if ( umount(containerdevpath) != 0 ) {
//        fprintf(stderr, "WARNING: Could not unmount %s\n", containerdevpath);
//        retval++;
//    }
//    if ( umount(containersyspath) != 0 ) {
//        fprintf(stderr, "WARNING: Could not unmount %s\n", containersyspath);
//        retval++;
//    }
//    if ( umount(containerprocpath) != 0 ) {
//        fprintf(stderr, "WARNING: Could not unmount %s\n", containerprocpath);
//        retval++;
//    }
//
//    if ( opt_contain == 0 ) {
//        if ( scratchpath != NULL ) {
//            if ( umount(containerscratchpath) != 0 ) {
//                fprintf(stderr, "WARNING: Could not unmount %s\n", containerscratchpath);
//                retval++;
//            }
//        }
//        if ( umount(containertmppath) != 0 ) {
//            fprintf(stderr, "WARNING: Could not unmount %s\n", containertmppath);
//            retval++;
//        }
//        if ( umount(containerhomepath) != 0 ) {
//            fprintf(stderr, "WARNING: Could not unmount %s: %s\n", containerhomepath, strerror(errno));
//            retval++;
//        }
//    }
//
//    // Dump all privs permanently at this point
//    if ( setregid(gid, gid) < 0 ) {
//        fprintf(stderr, "ERROR: Could not dump real and effective group privledges!\n");
//        return(255);
//    }
//    if ( setreuid(uid, uid) < 0 ) {
//        fprintf(stderr, "ERROR: Could not dump real and effective user privledges!\n");
//        return(255);
//    }
//#endif

    return(retval);
}
