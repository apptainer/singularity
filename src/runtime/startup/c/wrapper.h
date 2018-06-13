/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

#ifndef _SINGULARITY_WRAPPER_H
#define _SINGULARITY_WRAPPER_H

#define MAX_JSON_SIZE   128*1024
#define JOKER           42
#define MAX_ID_MAPPING  5

struct uidMapping {
    uid_t hostID;
    uid_t containerID;
    unsigned int size;
};

struct gidMapping {
    gid_t hostID;
    gid_t containerID;
    unsigned int size;
};

struct cConfig {
    unsigned long long capPermitted;
    unsigned long long capEffective;
    unsigned long long capInheritable;
    unsigned long long capBounding;
    unsigned long long capAmbient;
    unsigned int nsFlags;
    pid_t containerPid;
    pid_t netPid;
    pid_t mntPid;
    pid_t userPid;
    pid_t ipcPid;
    pid_t utsPid;
    pid_t cgroupPid;
    pid_t pidPid;
    unsigned char isSuid;
    unsigned char isInstance;
    unsigned char noNewPrivs;
    unsigned char hasNoNewPrivs;
    struct uidMapping uidMapping[MAX_ID_MAPPING];
    struct gidMapping gidMapping[MAX_ID_MAPPING];
    unsigned int jsonConfSize;
};

#endif /* _SINGULARITY_WRAPPER_H */
