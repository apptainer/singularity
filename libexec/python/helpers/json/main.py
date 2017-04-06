'''

main.py: main utility functions for json module in Singularity python API

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

import sys
import os
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), os.path.pardir)))

from sutils import (
    read_json,
    write_json
)

from message import bot
import json
import re
import os
import tempfile


def DUMP(jsonfile):
    '''DUMP will return the entire layfile as text, key value pairs
    :param jsonfile_path: the path to the jsonfile
    '''
    bot.debug("Reading %s to prepare dump to STDOUT" %jsonfile)
    if not os.path.exists(jsonfile):
        bot.error("Cannot find %s, exiting." %jsonfile)
        sys.exit(1)
    
    contents = read_json(jsonfile)
    dump = ""
    for key,value in contents.items():
        dump = '%s%s:"%s"\n' %(dump,key,value)
    dump = dump.strip('\n')
    print(dump)
    return dump



def GET(key,jsonfile):
    '''GET will return a key from the jsonfile, if it exists. If it doesn't, returns None.
    '''
    key = format_keyname(key)
    bot.debug("GET %s from %s" %(key,jsonfile))
    if not os.path.exists(jsonfile):
        bot.error("Cannot find %s, exiting." %jsonfile)
        sys.exit(1)
    
    contents = read_json(jsonfile)
    if key in contents:
        value = contents[key]
        print(value)
        bot.debug('%s is %s' %(key,value))
    else:
        bot.error("%s is not defined in file. Exiting" %key)
        sys.exit(1)
    return value


def ADD(key,value,jsonfile,force=False):
    '''ADD will write or update a key in a json file
    '''
    key = format_keyname(key)
    bot.debug("Adding label: '%s' = '%s'" %(key, value))
    bot.debug("ADD %s from %s" %(key,jsonfile))
    if os.path.exists(jsonfile):    
        contents = read_json(jsonfile)
        if key in contents:
            bot.debug('Warning, %s is already set. Overwrite is set to %s' %(key,force))
            if force == True:
                contents[key] = value
            else:
                bot.error('%s found in %s and overwrite set to %s.' %(key,jsonfile,force))
                sys.exit(1)
        else:
            contents[key] = value
    else:
        contents = {key:value}
    bot.debug('%s is %s' %(key,value))
    write_json(contents,jsonfile)
    return value


def DELETE(key,jsonfile):
    '''DELETE will remove a key from a json file
    '''
    key = format_keyname(key)
    bot.debug("DELETE %s from %s" %(key,jsonfile))
    if not os.path.exists(jsonfile):
        bot.error("Cannot find %s, exiting." %jsonfile)
        sys.exit(1)
    
    contents = read_json(jsonfile)
    if key in contents:
        del contents[key]
        if len(contents) > 0:
            write_json(contents,jsonfile)
        else:
            bot.debug('%s is empty, deleting.' %jsonfile)
            os.remove(jsonfile)
        return True
    else:    
        bot.debug('Warning, %s not found in %s' %(key,jsonfile))
        return False


def format_keyname(key):
    '''format keyname will ensure that all keys are uppcase, with no special
    characters
    '''
    return re.sub('[^A-Za-z0-9]+', '_', key).upper()
