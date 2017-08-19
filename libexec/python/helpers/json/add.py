#!/usr/bin/env python

'''

add.py: wrapper for "add" of a key to a json file for
        Singularity Hub command line tool.

This function takes input arguments of the following:

   --key: should be the key to lookup from the json file
   --value: the value to add to the key
   --file: should be the json file to read

Copyright (c) 2017, Vanessa Sochat. All rights reserved.

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

import sys
import os
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__),
                os.path.pardir,
                os.path.pardir)))  # noqa

import optparse
from helpers.json.main import ADD
from message import bot


def get_parser():

    parser = optparse.OptionParser(description="GET key from json")

    parser.add_option("--key",
                      dest='key',
                      help="key to add to json",
                      type=str,
                      default=None)

    parser.add_option("--value",
                      dest='value',
                      help="value to add to the json",
                      type=str,
                      default=None)

    parser.add_option("--file",
                      dest='file',
                      help="Path to json file to add to",
                      type=str,
                      default=None)

    parser.add_option('-f', dest="force",
                      help="force add (overwrite if exists)",
                      default=False, action='store_true')

    parser.add_option('--quiet', dest="quiet",
                      help="do not display debug",
                      default=False, action='store_true')

    return parser


def main():

    parser = get_parser()

    try:
        (args, options) = parser.parse_args()
    except Exception:
        sys.exit(0)

    if args.key is not None and args.file is not None:
        if args.value is not None:

            value = ADD(key=args.key,
                        value=args.value,
                        jsonfile=args.file,
                        force=args.force,
                        quiet=args.quiet)

    else:
        bot.error("--key and --file and --value must be defined for ADD.")
        sys.exit(1)


if __name__ == '__main__':
    main()
