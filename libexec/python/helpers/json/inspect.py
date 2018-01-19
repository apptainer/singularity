#!/usr/bin/env python

'''

inspect.py: python helper for Singularity inspect

ENVIRONMENTAL VARIABLES that are used for this executable:

SINGULARITY_MOUNTPOINT
SINGULARITY_INSPECT_LABELS
SINGULARITY_INSPECT_DEFFILE
SINGULARITY_INSPECT_RUNSCRIPT
SINGULARITY_INSPECT_TEST
SINGULARITY_INSPECT_ENVIRONMENT
SINGULARITY_APPNAME

Copyright (c) 2017, Vanessa Sochat. All rights reserved.

'''

import sys
import os
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__),
                                os.path.pardir,
                                os.path.pardir)))  # noqa

from helpers.json.main import INSPECT
from message import bot
from defaults import getenv


def main():
    '''this function will run the main inspect function
    '''
    labels = getenv("SINGULARITY_INSPECT_LABELS", None)
    deffile = getenv("SINGULARITY_INSPECT_DEFFILE", None)
    runscript = getenv("SINGULARITY_INSPECT_RUNSCRIPT", None)
    test = getenv("SINGULARITY_INSPECT_TEST", None)
    environment = getenv("SINGULARITY_INSPECT_ENVIRONMENT", None)
    helpfile = getenv("SINGULARITY_INSPECT_HELP", None)
    structured = getenv("SINGULARITY_PRINT_STRUCTURED", None)
    app = getenv("SINGULARITY_APPNAME", None)

    pretty_print = True
    if structured is not None:
        pretty_print = False

    INSPECT(inspect_app=app,
            inspect_labels=labels,
            inspect_def=deffile,
            inspect_runscript=runscript,
            inspect_test=test,
            inspect_help=helpfile,
            inspect_env=environment,
            pretty_print=pretty_print)


if __name__ == '__main__':
    main()
