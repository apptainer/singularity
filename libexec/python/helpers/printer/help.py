#!/usr/bin/env python

'''

help.py: python help printer for Singularity help

This function will look for a runscript.help file in the
container base, and print to the console if provided.
If an app name is provided, it will look in the app folder
instead

If not, nothing is printed.

ENVIRONMENTAL VARIABLES that are used for this executable:

SINGULARITY_MOUNTPOINT

Copyright (c) 2017, Vanessa Sochat. All rights reserved.

'''

import sys
import os
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__),
                                os.path.pardir,
                                os.path.pardir)))  # noqa

import optparse
from helpers.printer.main import HELP
from defaults import getenv
from message import bot

def get_parser():

    parser = optparse.OptionParser(description="HELP printer")

    parser.add_option("--file", 
                      dest='file', 
                      help="Path to json file to retrieve from", 
                      type=str,
                      default=None)

    return parser



def main():

    parser = get_parser()
    
    try:
        (args,options) = parser.parse_args()
    except:
        sys.exit(0)
    
    structured = getenv("SINGULARITY_PRINT_STRUCTURED", None)

    if args.file is None:
        bot.error("Please provide a help --file to print.")
        sys.exit(1)

    HELP(filepath=args.file,
         pretty_print=structured)


if __name__ == '__main__':
    main()
