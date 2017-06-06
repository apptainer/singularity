/* 
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 *
 * Copyright (c) 2015-2017, Gregory M. Kurtzer. All rights reserved.
 * 
 * Copyright (c) 2016-2017, The Regents of the University of California,
 * through Lawrence Berkeley National Laboratory (subject to receipt of any
 * required approvals from the U.S. Dept. of Energy).  All rights reserved.
 * 
 * This software is licensed under a customized 3-clause BSD license.  Please
 * consult LICENSE file distributed with the sources of this project regarding
 * your rights to use or distribute this software.
 * 
 * NOTICE.  This Software was developed under funding from the U.S. Department of
 * Energy and the U.S. Government consequently retains certain rights. As such,
 * the U.S. Government has been granted for itself and others acting on its
 * behalf a paid-up, nonexclusive, irrevocable, worldwide license in the Software
 * to reproduce, distribute copies to the public, prepare derivative works, and
 * perform publicly and display publicly, and to permit other to do so. 
 * 
 */


#ifndef __PRIVILEGE_H_
#define __PRIVILEGE_H_

    void singularity_priv_init(void);
    void singularity_priv_userns(void);
    void singularity_priv_escalate(void);
    void singularity_priv_drop(void);
    void singularity_priv_drop_perm(void);
    uid_t singularity_priv_getuid(void);
    gid_t singularity_priv_getgid(void);
    const gid_t *singularity_priv_getgids();
    int singularity_priv_getgidcount(void);
    int singularity_priv_userns_enabled(void);
    int singularity_priv_is_suid(void);
    char *singularity_priv_home(void);
    char *singularity_priv_homedir(void);
    char *singularity_priv_getuser(void);

    // Escalate privileges to the 'singularity' user.
    void singularity_priv_escalate_singularity(void);

    // Returns 1 if invoking user is a member of `gid`, 0 otherwise
    int singularity_priv_has_gid(gid_t gid);

    // Returns the UID of the singularity user.
    uid_t singularity_priv_singularity_uid();

    // Returns the GID of the singularity user.
    gid_t singularity_priv_singularity_gid();

#endif /* __PRIVILEGE_H_ */
