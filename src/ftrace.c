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

#include <errno.h>
#include <signal.h>
#include <stdio.h>
#include <string.h>
#include <sys/types.h>
#include <sys/reg.h>
#include <sys/ptrace.h>
//#include <linux/ptrace.h>
#include <sys/syscall.h>
#include <sys/wait.h>
#include <sys/user.h>
#include <unistd.h>
#include "file.h"

#ifndef ARCH_x86_64
#ifndef ARCH_i386
#error Singularity build arch not supported
#endif
#endif


int main(int argc, char **argv) {
    pid_t child;

    // fork early
    child = fork ();

    if ( child == -1 ) {
        printf ("Error calling fork()");
        return(1);
    } else if ( child == 0 ) {
        // reassign arguments -1
        char *newargv[argc]; // Flawfinder: ignore
        int i;

        for(i=0; i<argc-1; i++) {
            newargv[i] = argv[i+1];
        }
        newargv[argc-1] = NULL;

        // redirect stderr to stdout
        dup2(1, 2);

        ptrace(PTRACE_TRACEME, 0, NULL, NULL);
        ptrace(PTRACE_SETOPTIONS, 0, NULL, PTRACE_O_TRACECLONE|PTRACE_O_TRACEFORK|PTRACE_O_TRACEVFORK);

        execv(newargv[0], newargv); // Flawfinder: ignore (exec* is necessary)
    } else {
        char str[256*8]; // Flawfinder: ignore

        // loop through running binary until binary is done
        while (1){
            int syscall;
            struct user_regs_struct regs;
            int status;
            pid_t pid;

            // wait at every ptrace stopping point
            //wait (&status);
            pid = waitpid(-1, &status, __WALL);

            // exit if the process has exited
            if ( pid == child && WIFEXITED(status) ) {
                break;
            }

            // get the current register struct
            ptrace(PTRACE_GETREGS, pid, 0, &regs);     
#ifdef ARCH_x86_64
            syscall = regs.orig_rax;
#elif ARCH_i386
            syscall = regs.orig_eax;
#endif

            // if we are in an open() system call...
            if (syscall == SYS_open || syscall == SYS_execve) {
                int len = 0;

                // we need to iterate through, and pull sizeof(long)
                while(len <= 256) {
                    union u { 
                        long val;
                        char string[sizeof(long)]; // Flawfinder: ignore
                    } data;

#ifdef ARCH_x86_64
                    data.val = ptrace(PTRACE_PEEKDATA, pid, regs.rdi + len, NULL);
#elif ARCH_i386
                    data.val = ptrace(PTRACE_PEEKDATA, pid, regs.ebx + len, NULL);
#endif
                    if ( data.val == -1 ) {
                        break;
                    }

                    memcpy(str + len, data.string, sizeof(long)); // Flawfinder: ignore

                    len += sizeof(long);

                }

                str[len] = '\0';

                if ( syscall == SYS_open ) {
                    // how did open() exit
#ifdef ARCH_x86_64
                    long ret = ptrace(PTRACE_PEEKUSER, pid, 8 * RAX, NULL);
#elif ARCH_i386
                    long ret = ptrace(PTRACE_PEEKUSER, pid, 4 * EAX, NULL);
#endif
                    if ( ret >= 0 ) {
                        if ( strncmp(str, "/dev", 4) == 0 ) {
                        } else if ( strncmp(str, "/sys", 4) == 0 ) {
                        } else if ( strncmp(str, "/proc", 5) == 0 ) {
                        } else {
                            if ( is_file(str) == 0 || is_link(str) == 0 ) {
                                fprintf(stderr, "%s\n", str);
                            }
                        }
                    }
                } else if ( syscall == SYS_execve ) {
                    if ( is_exec(str) == 0 ) {
                        fprintf(stderr, "%s\n", str);
                    }

                }

            // Catching the fork/clone is very fustratingly not working. If you got
            // an idea on how to fix this, please!
            } else if (syscall == SYS_clone || syscall == SYS_fork || syscall == SYS_vfork) {
#ifdef ARCH_x86_64
                long pid2 = ptrace(PTRACE_PEEKUSER, pid, 8 * RAX);
#elif ARCH_i386
                long pid2 = ptrace(PTRACE_PEEKUSER, pid, 4 * EAX);
#endif
                if ( pid2 > 0 ) {
                    ptrace(PTRACE_ATTACH, pid2, NULL, NULL);
                }
            }

            // run and pause at the next system call
            ptrace (PTRACE_SYSCALL, pid, NULL, NULL);
        }
    }

    return 0;
}
