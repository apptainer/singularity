

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/mount.h>
#include <unistd.h>


int main(int argc, char *argv[]) {
    unsigned int exit_status = 0;
    int SMALLBUFF = 64;
    int BUFF = 512;
    int uid = getuid();
    int euid = geteuid();
    int sapp_file_len = strlen(argv[1]);
    int arg_string_len = 0;
    int i = 0;
    int j = 0;
    char *sapp_file;
    char *tmpdir;
    char *mktmpdir;
    char *rmtmpdir;
    char *explode_sapp;
    char *run_cmd;
    char *arg_string;
//    char *bind_mountpoint;
    char cwd[BUFF];

    seteuid(uid);
    
    getcwd(cwd, BUFF);

    for (i = 2; i < argc; i++) {
        arg_string_len += strlen(argv[i]) + 1;
    }
    arg_string_len ++;

    sapp_file = (char *) malloc(sapp_file_len + 1);
    tmpdir = (char *) malloc(SMALLBUFF);
    mktmpdir = (char *) malloc(SMALLBUFF);
    rmtmpdir = (char *) malloc(SMALLBUFF);
    explode_sapp = (char *) malloc(BUFF + sapp_file_len);
    run_cmd = (char *) malloc(BUFF + arg_string_len);
    arg_string = (char *) malloc(arg_string_len);
//    bind_mountpoint = (char *) malloc(BUFF);

    strcpy(sapp_file, argv[1]);

    
    for (i = 2; i < argc; i++) {
        memcpy(arg_string + j, argv[i], strlen(argv[i]));
        j += strlen(argv[i]);
        arg_string[j] = ' ';
        j++;
    }

    arg_string[j+1] = '\0';

    snprintf(tmpdir, /*sizeof(tmpdir)*/ SMALLBUFF, "/tmp/.singularity.%d.%d", uid, getpid());
    snprintf(mktmpdir, /*sizeof(mktmpdir)*/ SMALLBUFF, "mkdir -p %s", tmpdir);
    snprintf(rmtmpdir, /*sizeof(mktmpdir)*/ SMALLBUFF, "rm -rf %s", tmpdir);
    snprintf(explode_sapp, BUFF + sapp_file_len, "zcat %s | (cd %s; cpio -id --quiet)", sapp_file, tmpdir);
    snprintf(run_cmd, BUFF, "/run %s", arg_string);
//    snprintf(bind_mountpoint, BUFF, "%s/home", tmpdir);

    //Prepare
    system(mktmpdir);
    system(explode_sapp);
//    mkdir(bind_mountpoint, 0770);

    //Chroot
    seteuid(0);
    /*
     * It doesn't appear that the mount is necessary.. the chdir command
     * escapes the chroot! Is this reliable?
    if ( mount("/home", bind_mountpoint, "", MS_BIND, NULL) != 0 ) {
        printf("Mount failed\n\n");
    }
    */

    pid_t forkpid = fork();
    if ( forkpid == 0 ) {
        //Work
        chroot(tmpdir);
        chdir(cwd);
        seteuid(uid);
        system(run_cmd);
        exit(0);
    } else if ( forkpid > 0 ) {
        //get exit of child... later
        wait(-1);
    } else {
        printf("Could not fork!!!\n");
    }

    //Root Cleanup
    /*
     * Uncomment if we end up doing the mount
    if ( umount(bind_mountpoint) != 0) {
        printf("Umount failed\n\n");
    }
    */

    //User Cleanup
    seteuid(uid);
    //system(rmtmpdir);

    return(exit_status);
}
