import os
import logging

def get_logging_level():
    '''get_logging_level will configure a logging to standard out based on the user's
    selected level, which should be in an environment variable called MESSAGELEVEL.
    if MESSAGELEVEL is not set, the maximum level (5) is assumed (all messages).
    '''

    MESSAGELEVEL = os.environ.get("MESSAGELEVEL",5)
    print("Environment message level found to be %s" %MESSAGELEVEL)
    if MESSAGELEVEL in [5,"5"]:
        level = logging.CRITICAL
    elif MESSAGELEVEL in [4,"4"]:
        level = logging.ERROR
    elif MESSAGELEVEL in [3,"3"]:
        level = logging.WARNING
    elif MESSAGELEVEL in [2,"2"]:
        level = logging.INFO
    elif MESSAGELEVEL in [1,"1"]:
        level = logging.DEBUG
    print("Logging level set to %s" %level)
    return level

level = get_logging_level()
logging.basicConfig(level=level)
logger = logging.getLogger('python')
