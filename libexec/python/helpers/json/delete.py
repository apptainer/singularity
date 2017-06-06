#!/usr/bin/env python

'''

delete.py: wrapper for "delete" of a json file for Singularity Hub command line tool.

This function takes input arguments (not environment variables) of the following:

   --key: should be the key to delete from the json file
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
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), os.path.pardir, os.path.pardir)))

import optparse
from helpers.json.main import DELETE
from message import bot

def get_parser():

    parser = optparse.OptionParser(description="GET key from json")

    parser.add_option("--key", 
                      dest='key', 
                      help="key to delete from json", 
                      type=str,
                      default=None)

    parser.add_option("--file", 
                      dest='file', 
                      help="Path to json file to delete from", 
                      type=str,
                      default=None)

    return parser



def main():

    parser = get_parser()
    
    try:
        (args,options) = parser.parse_args()
    except:
        sys.exit(0)
    
    if args.key is not None and args.file is not None:

       success = DELETE(key=args.key,
                        jsonfile=args.file)
    else:
        bot.error("--key and --file must be defined for DELETE. Exiting")
        sys.exit(1)

if __name__ == '__main__':
    main()
