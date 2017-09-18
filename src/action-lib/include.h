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

#ifndef __ACTION_LIB_H_
#define __ACTION_LIB_H_

extern void action_ready(void);
extern int action_shell(int argc, char **argv);
extern int action_exec(int argc, char **argv);
extern int action_run(int argc, char **argv);
extern int action_test(int argc, char **argv);
extern int action_appexec(int argc, char **argv);
extern int action_apprun(int argc, char **argv);
extern int action_appshell(int argc, char **argv);
extern int action_apptest(int argc, char **argv);

#endif /* __ACTION_LIB_H */

