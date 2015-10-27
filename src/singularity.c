
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/mount.h>
#include <unistd.h>
#include "singularity.h"
#include "constants.h"
#include "config.h"

int proglevel;

int message(int msglevel,char *message) {
    if ( msglevel <= proglevel ) {
        printf(message);
    }
}

int show_usage() {
    message(0,"Usage : singularity filename.sapp application-arguments\n");
    message(0,"        -h|-help    for this usage info\n");
    message(0,"        -d|-debug   Show debugging output\n\n");

    return(0);
}

int need_help(char *arg1) {
    if( !strcmp(arg1,"-h") || !strcmp(arg1,"--h") || !strcmp(arg1,"-help") || !strcmp(arg1,"--help")) {    
       return(1);
    }

    return(0);
}

int mk_folder(char *tmpdir) {
    char *mktmpdir;

    mktmpdir = (char *) malloc(SMALLBUFF);
    snprintf(mktmpdir, /*sizeof(mktmpdir)*/ SMALLBUFF, "mkdir -p %s", tmpdir);
  
    system(mktmpdir);
    free(mktmpdir);

    return(0);
}

int rm_folder(char *tmpdir) {
    char *rmtmpdir;

    rmtmpdir = (char *) malloc(SMALLBUFF);
    snprintf(rmtmpdir, /*sizeof(mktmpdir)*/ SMALLBUFF, "rm -rf %s", tmpdir);
   
    system(rmtmpdir);
    free(rmtmpdir);

    return(0);
}

int explode_archive(char *sapp_file, char *tmpdir) {
    char *explode_sapp;
    int sapp_file_len;

    sapp_file_len = strlen(sapp_file);
    explode_sapp = (char *) malloc(BUFF + sapp_file_len);
    snprintf(explode_sapp, BUFF + sapp_file_len, "%s/sapp_explode '%s' '%s'", LIBEXECPATH, sapp_file, tmpdir);

    system(explode_sapp);
    message(2,"Finished exploding archive\n");
    free(explode_sapp);

    return(0);
}

int main(int argc, char *argv[]) {

    proglevel = DEFAULTLEVEL;
    //Make sure the UID is set back to the user
    int uid = getuid();
    int euid = geteuid();

    //Check for argument count and help option
    int exit_status = 255;
    if(argc < 2 || need_help(argv[1])) {
       show_usage();
       return(exit_status);
    }
    
    //Check for -d if so set level to 3. For now increased to 2.
    //proglevel = 2;

    int i=0, j=0;
    char cwd[BUFF];
    int sapp_file_len;
    char *sapp_file;
    int arg_string_len = 0;
    char *arg_string;
    char *tmpdir;
    char *run_cmd;
//    char *bind_mountpoint;

    seteuid(uid);
    message(2,"Initalization...\n");
    
    getcwd(cwd, BUFF);

    message(2,"Creating temporary directory space\n");
    //Setup temporary space to work with
    //Create tmpdir
    tmpdir = (char *) malloc(SMALLBUFF);
    snprintf(tmpdir, /*sizeof(tmpdir)*/ SMALLBUFF, "%s.%d.%d", TEMP_PATH, uid, getpid());
    mk_folder(tmpdir);

    message(2,"Exploding the sapp file to temporary directory\n");
    //Get sapp file and explode the cpio archive into TEMP dir
    sapp_file_len = strlen(argv[1]);
    sapp_file = (char *) malloc(sapp_file_len + 1); //Plus 1 for \0
    strcpy(sapp_file, argv[1]);
    explode_archive(sapp_file,tmpdir);

    //Get app arguments and create run command
    for (i = 2; i < argc; i++) {
        arg_string_len += strlen(argv[i]);
    }
    // Add spaces
    arg_string_len += argc;
    arg_string = (char *) malloc(arg_string_len);

    for (i = 2; i < argc; i++) {
        memcpy(arg_string + j, argv[i], strlen(argv[i]));
        j += strlen(argv[i]);
        arg_string[j] = ' ';
        j++;
    }
    arg_string[j] = '\0';
    run_cmd = (char *) malloc(BUFF + arg_string_len);
    snprintf(run_cmd, BUFF + arg_string_len, "/run \"%s\"", arg_string);

    //Setup for the bind mounts
    //bind_mountpoint = (char *) malloc(BUFF);
    //snprintf(bind_mountpoint, BUFF, "%s/home", tmpdir);
    //mkdir(bind_mountpoint, 0770);
    
    message(2,"Escalating privs\n");

    seteuid(0);
    //Get down to root
    /*
     * It doesn't appear that the mount is necessary.. the chdir command
     * escapes the chroot! Is this reliable?
    if ( mount("/home", bind_mountpoint, "", MS_BIND, NULL) != 0 ) {
        message(1,"Mount failed\n\n");
    }
    */

    message(2,"Forking\n");
    pid_t forkpid = fork();
    if ( forkpid == 0 ) { //Child process starts here
        int retval;
        message(2,"Hello from child spawn\n");
        //Start the chroot on TEMP dir
        message(2,"Preparing Chroot\n");
        if ( chroot(tmpdir) != 0 ) {
            message(1,strcat("Error: failed chroot to:",tmpdir));
            message(1,"\n");
            exit(255);
        }
        seteuid(uid);
        message(2,"Changing to working dir\n");
        chdir(cwd);
        message(3,"running command: ");
        message(3,run_cmd);
        message(3,"\n");
        retval = system(run_cmd);
        exit(WEXITSTATUS(retval)); //Child stops running here
    } else if ( forkpid > 0 ) { 
        //Parent process
        //get exit of child... later
        //exit_status = wait(forkpid);
        int retval;
        waitpid(forkpid, &retval, 0);
        exit_status = WEXITSTATUS(retval);
        message(2,"Child has returned home!\n");
    } else {
        message(1,"Could not fork!!!\n");
    }

    //Root Cleanup
    /*
     * Uncomment if we end up doing the mount
    if ( umount(bind_mountpoint) != 0) {
        message(1,"Umount failed\n\n");
    }
    */

    //User Cleanup
    seteuid(uid);
    //rm_folder(tmpdir);

    return(exit_status);
}
