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

#include <errno.h>
#include <signal.h>
#include <stdio.h>
#include <string.h>
#include <sys/types.h>
#include <sys/reg.h>
#include <sys/ptrace.h>
#include <sys/syscall.h>
#include <sys/wait.h>
#include <sys/user.h>
#include <unistd.h>
#include "util.h"

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
        char *newargv[argc];
        int i;

        for(i=0; i<argc-1; i++) {
            newargv[i] = argv[i+1];
        }
        newargv[argc-1] = NULL;

        // redirect stderr to stdout
        dup2(1, 2);

        ptrace(PTRACE_TRACEME, 0, NULL, NULL);

        execv(newargv[0], newargv);
    } else {
        char str[256*8];
        int insyscall = 0;

        // loop through running binary until binary is done
        while (1){
            int syscall;
            struct user_regs_struct regs;
            int status;

            // wait at every ptrace stopping point
            wait (&status);

            // exit if the child has exited
            if ( WIFEXITED(status) ) {
                break;
            }

            // get the current register struct
            ptrace(PTRACE_GETREGS,child, 0, &regs);     
#ifdef ARCH_x86_64
            syscall = regs.orig_rax;
#elif ARCH_i386
            syscall = regs.orig_eax;
#endif

            // if we are in an open() system call...
            if (syscall == SYS_open) {

                // check to see if we are already in the midst of a system call
                if (insyscall == 0){
                    int len = 0;

                    // we need to iterate through, and pull sizeof(long)
                    while(len <= 256) {
                        union u { 
                            long val;
                            char string[sizeof(long)];
                        } data;

#ifdef ARCH_x86_64
                        data.val = ptrace(PTRACE_PEEKDATA, child, regs.rdi + len, NULL);
#elif ARCH_i386
                        data.val = ptrace(PTRACE_PEEKDATA, child, regs.ebx + len, NULL);
#endif
                        if ( data.val == -1 ) {
                            break;
                        }

                        memcpy(str + len, data.string, sizeof(long));

                        len += sizeof(long);

                    }

                    str[len] = '\0';

                    // the following ptrace SYS_open will be the close
                    insyscall = 1;
                } else {
                    // how did the system call exit
#ifdef ARCH_x86_64
                    long ret = ptrace(PTRACE_PEEKUSER, child, 8 * RAX, NULL);
#elif ARCH_i386
                    long ret = ptrace(PTRACE_PEEKUSER, child, 4 * ORIG_EAX, NULL);
#endif
                    if ( ret >= 0 ) {
                        if ( strncmp(str, "/dev", 4) == 0 ) {
                        } else if ( strncmp(str, "/etc", 4) == 0 ) {
                        } else if ( strncmp(str, "/sys", 4) == 0 ) {
                        } else if ( strncmp(str, "/proc", 5) == 0 ) {
                        } else {
                            if ( s_is_dir(str) < 0 ) {
                                fprintf(stderr, "%s\n", str);
                            }
                        }
                    }
                    insyscall = 0;
                }
            }
        
            // run and pause at the next system call
            ptrace (PTRACE_SYSCALL, child, NULL, NULL);
        }
    }

    return 0;
}
