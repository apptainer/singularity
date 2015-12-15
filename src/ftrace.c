#include <stdio.h>
#include <sys/types.h>
#include <sys/reg.h>
#include <sys/ptrace.h>
#include <sys/syscall.h>
#include <sys/wait.h>
#include <unistd.h>
#include <sys/ptrace.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <sys/syscall.h>
#include <unistd.h>
#include <sys/reg.h>
#include <errno.h>
#include <string.h>
#include <stdio.h>
#include <sys/types.h>
#include <signal.h>
#include <sys/user.h>



int main() {
    pid_t child;
    long orig_eax, eax;
    int status;
    int insyscall =0;
    child = fork ();

    if (child == -1) 
        printf ("Error in fork");

    if (child == 0) {
        // In the child process
        ptrace(PTRACE_TRACEME, 0, NULL, NULL);
        execl("/bin/cat", "cat", "/etc/fstab", NULL);
    
    } else {
        char str[256*8];

        //In parent process
        while (1){
            int syscall;
            struct user_regs_struct u_in, u_out;            

            wait (&status);
            if (WIFEXITED(status))
                break;

            ptrace(PTRACE_GETREGS,child, 0, &u_in);     
            syscall = u_in.orig_rax;

//            printf ("System call is : %d\n", syscall);
            if (syscall == SYS_open) {
                orig_eax = ptrace (PTRACE_PEEKUSER, child, 4 * ORIG_RAX, NULL);

                if (insyscall == 0){
                    long bx, cx;
                    int len = 0;

                    while(len <= 256) {
                        union u { 
                            long val;
                            char string[sizeof(long)];
                        } data;
    
                        data.val = ptrace(PTRACE_PEEKDATA, child, u_in.rdi + len, NULL);
                        if ( data.val == -1 ) {
                            printf("len: %d\n", len);
                            break;
                        }

                        memcpy(str + len, data.string, sizeof(long));

                        len += sizeof(long);

                    }

                    str[len] = '\0';


                    insyscall = 1;
                } else {
                    eax = ptrace(PTRACE_PEEKUSER, child, 8 * RAX, NULL);
                    if ( eax >= 0 ) {
                        fprintf(3, "%s\n", str);
                    }
                    insyscall = 0;
                }
            }
        
        ptrace (PTRACE_SYSCALL, child, NULL, NULL);
        }
    }
    return 0;
}
