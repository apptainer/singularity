/*
  Copyright (c) 2018-2019, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE.md file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

#ifndef _SINGULARITY_STARTER_H
#define _SINGULARITY_STARTER_H

#include <limits.h>
#include <sys/user.h>

#define fatalf(b...)     singularity_message(ERROR, b); \
                         exit(1)
#define debugf(b...)     singularity_message(DEBUG, b)
#define verbosef(b...)   singularity_message(VERBOSE, b)
#define warningf(b...)   singularity_message(WARNING, b)
#define errorf(b...)     singularity_message(ERROR, b)

#define MAX_MAP_SIZE        4096
#define MAX_PATH_SIZE       PATH_MAX
#define MAX_GID             32
#define MAX_STARTER_FDS     1024
#define MAX_CMD_SIZE        MAX_PATH_SIZE+MAX_MAP_SIZE+64

#ifndef PR_SET_NO_NEW_PRIVS
#define PR_SET_NO_NEW_PRIVS 38
#endif

#ifndef PR_GET_NO_NEW_PRIVS
#define PR_GET_NO_NEW_PRIVS 39
#endif

#define NO_NAMESPACE        -1
#define CREATE_NAMESPACE    0
#define ENTER_NAMESPACE     1

enum goexec {
    STAGE1      = 1,
    STAGE2      = 2,
    MASTER      = 3,
    RPC_SERVER  = 4
};

#ifndef NS_CLONE_NEWPID
#define CLONE_NEWPID        0x20000000
#endif

#ifndef NS_CLONE_NEWNET
#define CLONE_NEWNET        0x40000000
#endif

#ifndef NS_CLONE_NEWIPC
#define CLONE_NEWIPC        0x08000000
#endif

#ifndef NS_CLONE_NEWUTS
#define CLONE_NEWUTS        0x04000000
#endif

#ifndef NS_CLONE_NEWUSER
#define CLONE_NEWUSER       0x10000000
#endif

#ifndef NS_CLONE_NEWCGROUP
#define CLONE_NEWCGROUP     0x02000000
#endif

typedef enum {
    false,
    true
} bool;

/* container capabilities */
struct capabilities {
    unsigned long long permitted;
    unsigned long long effective;
    unsigned long long inheritable;
    unsigned long long bounding;
    unsigned long long ambient;
};

/* container namespaces */
struct namespace {
    /* namespace flags (CLONE_NEWPID, CLONE_NEWUSER ...) */
    unsigned int flags;
    /* container mount namespace propagation */
    unsigned long mountPropagation;
    /* namespace join only */
    bool joinOnly;
    /* should bring up loopback interface with network namespace */
    bool bringLoopbackInterface;

    /* namespaces inodes paths used to join namespaces */
    char network[MAX_PATH_SIZE];
    char mount[MAX_PATH_SIZE];
    char user[MAX_PATH_SIZE];
    char ipc[MAX_PATH_SIZE];
    char uts[MAX_PATH_SIZE];
    char cgroup[MAX_PATH_SIZE];
    char pid[MAX_PATH_SIZE];
};

/* container privileges */
struct privileges {
    /* value for PR_SET_NO_NEW_PRIVS */
    bool noNewPrivs;

    /* user namespace mappings and setgroups control */
    char uidMap[MAX_MAP_SIZE];
    char gidMap[MAX_MAP_SIZE];
    bool allowSetgroups;

    /* path to external newuidmap/newgidmap binaries */
    char newuidmapPath[MAX_PATH_SIZE];
    char newgidmapPath[MAX_PATH_SIZE];

    /* uid/gids set for container process execution */
    uid_t targetUID;
    gid_t targetGID[MAX_GID];
    int numGID;

    /* container process capabilities */
    struct capabilities capabilities;
};

/* container configuration */
struct container {
    /* container process ID */
    pid_t pid;
    /* is container will run as instance */
    bool isInstance;

    /* container privileges */
    struct privileges privileges;
    /* container namespaces */
    struct namespace namespace;
};

/* starter behaviour */
struct starter {
    /* control starter working directory from a file descriptor */
    int workingDirectoryFd;

    /* hold file descriptors that need to be remains open after stage 1 */
    int fds[MAX_STARTER_FDS];
    int numfds;

    /* is starter run as setuid */
    bool isSuid;
    /* master process will share a mount namespace for container mount propagation */
    bool masterPropagateMount;
    /* hybrid workflow where master process and container doesn't share user namespace */
    bool hybridWorkflow;

    /* bounding capability set will include caps needed by nvidia-container-cli */
    bool nvCCLICaps;
};

/* engine configuration */
struct engine {
    size_t size;
    size_t map_size;
    char *config;
};

/* starter configuration */
struct starterConfig {
    struct container container;
    struct starter starter;
    struct engine engine;
};

#endif /* _SINGULARITY_STARTER_H */
