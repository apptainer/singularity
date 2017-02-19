import os
import logging

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
    elif MESSAGELEVEL in [1,-2]:
        level = logging.WARNING

    #define LOG -1
    #define INFO 1
    elif MESSAGELEVEL == -1:
        level = logging.INFO

    #define VERBOSE 2
    #define VERBOSE1 2
    #define VERBOSE2 3
    #define VERBOSE3 4
    elif MESSAGELEVEL in [2,3,4,5]:
        level = logging.DEBUG

    #print("Logging level set to %s" %level)
    return level

level = get_logging_level()
logging.basicConfig(level=level)
logger = logging.getLogger('python')
