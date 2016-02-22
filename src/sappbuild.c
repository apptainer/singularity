/*
 *
 * Copyright (c) 2015-2016, Gregory M. Kurtzer
 * All rights reserved.
 *
 *
 * Copyright (c) 2015-2016, The Regents of the University of California,
 * through Lawrence Berkeley National Laboratory (subject to receipt of
 * any required approvals from the U.S. Dept. of Energy).
 * All rights reserved.
 *
 *
*/


// USAGE: ./sappbuilder [name of container] [container archive]
// OUTPUT: pipe of the executable output data
//
// Combine 3 binary bits in order and output to STDOUT:
//
//  * LIBEXECDIR/sapplauncher
//  * argv[2]: Container archive
//  * createheader(headerstruct) (library function in sappheader.c)
//

#define _GNU_SOURCE
#include <stdlib.h>
#include <stdio.h>
#include "sappheader.h"


#ifndef LIBEXECDIR
#define LIBEXECDIR "undefined"
#endif


int main(int argc, char **argv) {



    return(0);
}
