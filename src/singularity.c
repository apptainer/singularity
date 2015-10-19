
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/mount.h>
#include <unistd.h>

#define TEMP_PATH "/tmp/.singularity."
#define SMALLBUFF 64
#define BUFF 512
      
show_usage() {
    printf("Usage : singularity filename.sapp application-arguments\n");
    printf("        -h|-help for this usage info\n\n");
}

need_help(char *arg1) {
    if( !strcmp(arg1,"-h") || !strcmp(arg1,"--h") || !strcmp(arg1,"-help") || !strcmp(arg1,"--help")) {    
       return(1);
    } else {
       return(0);
    }
}
    
mk_tmpdir(char *tmpdir) {

    char *mktmpdir;

    mktmpdir = (char *) malloc(SMALLBUFF);
    snprintf(mktmpdir, /*sizeof(mktmpdir)*/ SMALLBUFF, "mkdir -p %s", tmpdir);
  
    system(mktmpdir);
    free(mktmpdir);
}

rm_tmpdir(char *tmpdir) {

    char *rmtmpdir;

    rmtmpdir = (char *) malloc(SMALLBUFF);
    snprintf(rmtmpdir, /*sizeof(mktmpdir)*/ SMALLBUFF, "rm -rf %s", tmpdir);
   
    system(rmtmpdir);
    free(rmtmpdir);
}

int main(int argc, char *argv[]) {

    //Make sure the UID is set back to the user
    int uid = getuid();
    int euid = geteuid();
    seteuid(uid);

    //Check for argument count and help option
    int exit_status = 255;
    if(argc < 2 || need_help(argv[1])) {
       show_usage();
       return(exit_status);
    }

    int i=0, j=0;
    char cwd[BUFF];
    int sapp_file_len;
    char *sapp_file;
    int arg_string_len = 0;
    char *arg_string;
    char *tmpdir;
    char *explode_sapp;
    char *run_cmd;
//    char *bind_mountpoint;
    
    getcwd(cwd, BUFF);

    //Setup temporary space to work with
    //Create tmpdir
    tmpdir = (char *) malloc(SMALLBUFF);
    snprintf(tmpdir, /*sizeof(tmpdir)*/ SMALLBUFF, "%s.%d.%d", TEMP_PATH, uid, getpid());
    mk_tmpdir(tmpdir);

    //Get sapp file
    sapp_file_len = strlen(argv[1]);
    sapp_file = (char *) malloc(sapp_file_len + 1);
    strcpy(sapp_file, argv[1]);

    //Get app arguments
    for (i = 2; i < argc; i++) {
        arg_string_len += strlen(argv[i]) + 1;
    }
    arg_string_len ++;
    arg_string = (char *) malloc(arg_string_len);

    for (i = 2; i < argc; i++) {
        memcpy(arg_string + j, argv[i], strlen(argv[i]));
        j += strlen(argv[i]);
        arg_string[j] = ' ';
        j++;
    }
    arg_string[j+1] = '\0';
  
    //Explode the application's cpio archive
    explode_sapp = (char *) malloc(BUFF + sapp_file_len);
    snprintf(explode_sapp, BUFF + sapp_file_len, "zcat %s | (cd %s; cpio -id --quiet)", sapp_file, tmpdir);

    run_cmd = (char *) malloc(BUFF + arg_string_len);
    snprintf(run_cmd, BUFF, "/run %s", arg_string);

    system(explode_sapp);

    //Setup for the bind mounts
    //bind_mountpoint = (char *) malloc(BUFF);
    //snprintf(bind_mountpoint, BUFF, "%s/home", tmpdir);
    //mkdir(bind_mountpoint, 0770);

    //Start the Chroot
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
        int retval;

        chroot(tmpdir);
        chdir(cwd);
        seteuid(uid);
        retval = system(run_cmd);
        exit(WEXITSTATUS(retval));
    } else if ( forkpid > 0 ) {
        //get exit of child... later
        //exit_status = wait(forkpid);
        int retval;
        waitpid(forkpid, &retval, 0);
        exit_status = WEXITSTATUS(retval);
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
    //rm_tmpdir(tmpdir);

    return(exit_status);
}
