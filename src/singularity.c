

#include <stdio.h>
#include <stdlib.h>
#include <string.h>



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
    char *bind_mountpoint;

    seteuid(uid);

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
    bind_mountpoint = (char *) malloc(BUFF);

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
    snprintf(bind_mountpoint, BUFF, "%s/rootfs", tmpdir);

    //Prepare
    system(mktmpdir);
    system(explode_sapp);
    mkdir(bind_mountpoint);

    //Chroot
    seteuid(0);
    //mount("/", bind_mountpoint, "", "bind");
    mount("/tmp", bind_mountpoint, "", "bind");
    chroot(tmpdir);

    //Work
    seteuid(uid);
    chdir("/");
    //symlink("/rootfs/tmp", "/tmp");
    system(run_cmd);

    //Root Cleanup
    seteuid(0);
    umount(bind_mountpoint);

    //User Cleanup
    seteuid(uid);
    //system(rmtmpdir);

    return(exit_status);
}
