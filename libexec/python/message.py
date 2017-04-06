'''

message.py: simple logger for Singularity python helper

Copyright (c) 2016-2017, Vanessa Sochat. All rights reserved. 

"Singularity" Copyright (c) 2016, The Regents of the University of California,
through Lawrence Berkeley National Laboratory (subject to receipt of any
required approvals from the U.S. Dept. of Energy).  All rights reserved.
 
This software is licensed under a customized 3-clause BSD license.  Please
consult LICENSE file distributed with the sources of this project regarding
your rights to use or distribute this software.
 
NOTICE.  This Software was developed under funding from the U.S. Department of
Energy and the U.S. Government consequently retains certain rights. As such,
the U.S. Government has been granted for itself and others acting on its
behalf a paid-up, nonexclusive, irrevocable, worldwide license in the Software
to reproduce, distribute copies to the public, prepare derivative works, and
perform publicly and display publicly, and to permit other to do so. 

'''

import os
import logging

class SingularityMessage:

    def __init__(self,MESSAGELEVEL=None):
        self.level = get_logging_level()
        logging.basicConfig(level=self.level)
        self.logger = logging.getLogger('python')
        formatter = logging.Formatter('%(message)s')
        self.logger.setFormatter(formatter)


    def is_quiet(self):
        '''get_level will return the current (SINGULARITY) level
        '''
        if self.logger.getEffectiveLevel() < 50:
            return False
        return True


    def get_mapping(self):
        '''get_mapping returns a lookup dictionary for how Singularity (C)
        logging levels (ints) translate to the python logger. The key translates
        to the environment variable SINGULARITY_MESSAGELEVEL (as an int)
        This function is primarily for understanding the mapping.
        '''
        levels = { 'DEFAULT':  { 'python_effective_level': logging.FATAL,
                                 'python_level': 'logging.FATAL',
                                 'singularity_levels': [0] },

                   'ABRT' :    { 'python_effective_level': logging.CRITICAL,
                                 'python_level': 'logging.CRITICAL',
                                 'singularity_levels': [-4],
                                 'flags': ['--quiet','-q'] },

                   'ERROR' :   { 'python_effective_level': logging.ERROR,
                                 'python_level': 'logging.ERROR',
                                 'singularity_levels': [-3] },

                   'WARNING' : { 'python_effective_level': logging.WARNING,
                                 'python_level': 'logging.WARNING',
                                 'singularity_levels': [-2] },
                  
                   'LOG' :     { 'python_effective_level': logging.INFO,
                                 'python_level': 'logging.INFO',
                                 'singularity_levels': [-1] },

                  'VERBOSE' :  { 'python_effective_level': logging.DEBUG,
                                 'python_level': 'logging.DEBUG',
                                 'singularity_levels': [2] }}

        # LOG is functionally the same as INFO level in python, both are logging.info
        levels['INFO'] = levels['LOG']
        levels['LOG']['singularity_levels'] = [1]

        # VERBOSE singularity levels all map to debug
        levels['VERBOSE1'] = levels['VERBOSE']
        levels['VERBOSE2'] = levels['VERBOSE']
        levels['VERBOSE2']['singularity_levels'] = [3]
        levels['VERBOSE3'] = levels['VERBOSE']
        levels['VERBOSE3']['singularity_levels'] = [4]
        return levels


    

def get_logging_level():
    '''get_logging_level will configure a logging to standard out based on the user's
    selected level, which should be in an environment variable called
    SINGULARITY_MESSAGELEVEL. if SINGULARITY_MESSAGELEVEL is not set, the maximum level
    (5) is assumed (all messages). levels from
    https://github.com/singularityware/singularity/blob/master/src/lib/message.h

    #define ABRT -4
    #define ERROR -3
    #define WARNING -2
    #define LOG -1
    #define INFO 1
    #define VERBOSE 2
    #define VERBOSE1 2
    #define VERBOSE2 3
    #define VERBOSE3 4
    #define DEBUG 5
    '''

    MESSAGELEVEL = int(os.environ.get("SINGULARITY_MESSAGELEVEL",5))

    #print("Environment message level found to be %s" %MESSAGELEVEL)

    if MESSAGELEVEL == 0:
        level = logging.FATAL

    #define ABRT -4
    elif MESSAGELEVEL == -4:
        level = logging.CRITICAL

    #define ERROR -3
    elif MESSAGELEVEL == -3:
        level = logging.ERROR

    #define WARNING -2
    elif MESSAGELEVEL == -2:
        level = logging.WARNING

    #define LOG -1
    #define INFO 1
    elif MESSAGELEVEL in [1,-1]:
        level = logging.INFO

    #define VERBOSE 2
    #define VERBOSE1 2
    #define VERBOSE2 3
    #define VERBOSE3 4
    elif MESSAGELEVEL in [2,3,4,5]:
        level = logging.DEBUG

    #print("Logging level set to %s" %level)
    return level


bot = SingularityMessage()
