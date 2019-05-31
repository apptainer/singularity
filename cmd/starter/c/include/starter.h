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

#define MAX_JSON_SIZE       128*1024
#define MAX_MAP_SIZE        4096
#define MAX_NS_PATH_SIZE    PATH_MAX
#define MAX_GID             32
#define MAX_STARTER_FDS     1024

#ifndef PR_SET_NO_NEW_PRIVS
#define PR_SET_NO_NEW_PRIVS 38
#endif

#ifndef PR_GET_NO_NEW_PRIVS
#define PR_GET_NO_NEW_PRIVS 39
#endif

#define CLONE_STACK_SIZE    1024*1024
#define BUFSIZE             512

#define NO_NAMESPACE        -1
#define CREATE_NAMESPACE    0
#define ENTER_NAMESPACE     1

#define STAGE1      1
#define STAGE2      2
#define MASTER      3
#define RPC_SERVER  4

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

struct fdlist {
    int *fds;
    unsigned int num;
};

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
    unsigned int flags;
    unsigned long mountPropagation;
    unsigned char joinOnly;
    unsigned char bringLoopbackInterface;

    char network[MAX_NS_PATH_SIZE];
    char mount[MAX_NS_PATH_SIZE];
    char user[MAX_NS_PATH_SIZE];
    char ipc[MAX_NS_PATH_SIZE];
    char uts[MAX_NS_PATH_SIZE];
    char cgroup[MAX_NS_PATH_SIZE];
    char pid[MAX_NS_PATH_SIZE];
};

/* container privileges */
struct privileges {
    unsigned char noNewPrivs;

    char uidMap[MAX_MAP_SIZE];
    char gidMap[MAX_MAP_SIZE];

    uid_t targetUID;
    gid_t targetGID[MAX_GID];
    int numGID;

    struct capabilities capabilities;
};

/* container configuration */
struct container {
    pid_t pid;
    unsigned char isInstance;

    struct privileges privileges;
    struct namespace namespace;
};

/* starter behaviour */
struct starter {
    unsigned char isSuid;
    unsigned char masterPropagateMount;
    int workingDirectoryFd;

    /* hold file descriptors that need to be remains open after stage 1 */
    int fds[MAX_STARTER_FDS];
    int numfds;
};

/* engine configuration */
struct engine {
    char config[MAX_JSON_SIZE];
    size_t size;
};

/* starter configuration */
struct starterConfig {
    struct container container;
    struct starter starter;
    struct engine engine;
};

/* helper to check if namespace flag is set */
static inline unsigned char is_namespace_create(struct namespace *nsconfig, unsigned int nsflag) {
    return (nsconfig->flags & nsflag) != 0;
}

/* helper to check if the corresponding namespace need to be joined */
static inline unsigned char is_namespace_enter(const char *nspath) {
    return nspath[0] != 0;
}

#endif /* _SINGULARITY_STARTER_H */
