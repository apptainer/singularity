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
    print_json,
    read_file,
    read_json,
    write_json
)

from defaults import (
    ENVIRONMENT,
    LABELFILE,
    RUNSCRIPT,
    TESTFILE,
    DEFFILE
)

from message import bot
import json
import re


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


def INSPECT(inspect_labels=None,inspect_def=None,inspect_runscript=None,inspect_test=None,
            inspect_env=None,pretty_print=True):
    '''INSPECT will print a "manifest" for an image, with one or more variables asked for by
    the user. The default prints a human readable format (text and json) and if pretty_print 
    is turned to False, the entire thing will be returned as json via the JSON API standard.
    The base is checked for the metadata folder, and if it does not exist, the links are 
    searched for and parsed (to support older versions).

    :param inspect_runscript: if not None, will include runscript, if it exists
    :param inspect_labels: if not None, will include labels, if they exist.
    :param inspect_def: if not None, will include definition file, if it exists
    :param inspect_test: if not None, will include test, if it exists.
    :param inspect_env: if not None, will include environment, if exists.
    :param pretty_print: if False, return all JSON API spec
    '''

    data = dict()
    errors = dict()

    # Labels
    if inspect_labels:
        bot.verbose2("Inspection of labels selected.")
        if os.path.exists(LABELFILE):
            data["labels"] = read_json(LABELFILE)
        else:
            data["labels"] = None
            errors["labels"] = generate_error(404, detail="This container does not have labels",
                                              title="Labels Undefined")

    # Definition File
    if inspect_def:
        bot.verbose2("Inspection of deffile selected.")
        if os.path.exists(DEFFILE):
            data["deffile"] = read_file(DEFFILE,readlines=False)
        else:
            data["deffile"] = None
            errors["deffile"] = generate_error(404,title="Definition File Undefined",
                                               detail="This container does not include the bootstrap definition")

    # Runscript
    if inspect_runscript:
        bot.verbose2("Inspection of runscript selected.")
        if os.path.exists(RUNSCRIPT):
            data["runscript"] = read_file(RUNSCRIPT,readlines=False)
        else:
            data["runscript"] = None
            errors["runscript"] = generate_error(404,title="Runscript Undefined",
                                                 detail="This container does not have any runscript defined")

    # Test
    if inspect_test:
        bot.verbose2("Inspection of test selected.")
        if os.path.exists(TESTFILE):
            data["test"] = read_file(TESTFILE,readlines=False)
        else:
            data["test"] = None
            errors["test"] = generate_error(404,title="Tests Undefined",
                                            detail="This container does not have any tests defined")


    # Environment
    if inspect_env:
        bot.verbose2("Inspection of environment selected.")
        if os.path.exists(ENVIRONMENT):
            data["environment"] = read_file(ENVIRONMENT,readlines=False)
        else:
            data["environment"] = None
            errors["environment"] = generate_error(404,title="Tests Undefined",
                                                detail="This container does not have any custom environment defined")

    if pretty_print:
        bot.verbose2("Structured printing specified.")
        for dtype,content in data.items():      
            if content is not None:
                if isinstance(content,dict):
                    print_json(content,print_console=True)
                else:
                    bot.info(content)
            else:
                print_json(errors[dtype],print_console=True)

    else:
        bot.verbose2("Unstructed printing specified")
        # Only define a type if there is data to return, else return errors
        result = dict()
        if len(data) > 0:
            result["data"] = {"attributes": data,
                              "type": "container" }
        else:
            result["errors"] = []
            for dtype,content in errors.items():
                result["errors"].append(content)             
        print_json(result,print_console=True)


def generate_error(status=None,title=None,detail=None):
    '''generate_error will return a JSON API error object, intended to be added to
    the list of "errors" at the top level. Errors and data cannot both be returned,
    and so these data structures are only returned given no data.
    :param status: the status code to return (coinciding with http status codes)
    :param title: a short title to describe the error
    :param detail: detail about the error
    '''
    params = locals()
    error = dict()
    for key, value in params.items():
        if value is not None:
            error[key] = value   
    return error


def format_keyname(key):
    '''format keyname will ensure that all keys are uppcase, with no special
    characters
    '''
    return re.sub('[^A-Za-z0-9]+', '_', key).upper()
