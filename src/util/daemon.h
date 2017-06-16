/* 
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 *
 * Copyright (c) 2015-2017, Gregory M. Kurtzer. All rights reserved.
 * 
 * This software is licensed under a 3-clause BSD license.  Please
 * consult LICENSE.md file distributed with the sources of this project 
 * regarding your rights to use or distribute this software.
 * 
 */

#ifndef __DAEMON_H_
#define __DAEMON_H_

    void daemon_join(void);
    void daemon_path(char *host_uid);
    void daemon_rootfs(void);

#endif
