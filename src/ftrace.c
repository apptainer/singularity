/*
 *
 * Copyright (c) 2015, Gregory M. Kurtzer
 * All rights reserved.
 *
 *
 * Copyright (c) 2015, The Regents of the University of California,
 * through Lawrence Berkeley National Laboratory (subject to receipt of
 * any required approvals from the U.S. Dept. of Energy).
 * All rights reserved.
 *
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
            syscall = regs.orig_rax;

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
    
                        data.val = ptrace(PTRACE_PEEKDATA, child, regs.rdi + len, NULL);
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
                    long ret = ptrace(PTRACE_PEEKUSER, child, 8 * RAX, NULL);
                    if ( ret >= 0 ) {
                        fprintf(stderr, "%s\n", str);
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
