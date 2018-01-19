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
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__),
                os.path.pardir)))  # noqa

from sutils import (
    print_json,
    read_file,
    read_json,
    write_json
)

from defaults import (
    ENVIRONMENT,
    HELPFILE,
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
    bot.debug("Reading %s to prepare dump to STDOUT" % jsonfile)
    if not os.path.exists(jsonfile):
        bot.error("Cannot find %s, exiting." % jsonfile)
        sys.exit(1)

    contents = read_json(jsonfile)
    dump = ""
    for key, value in contents.items():
        dump = '%s%s:"%s"\n' % (dump, key, value)
    dump = dump.strip('\n')
    print(dump)
    return dump


def GET(key, jsonfile):
    '''GET will return a key from the jsonfile
    if it exists. If it doesn't, returns None.
    '''
    key = format_keyname(key)
    bot.debug("GET %s from %s" % (key, jsonfile))
    if not os.path.exists(jsonfile):
        bot.error("Cannot find %s, exiting." % jsonfile)
        sys.exit(1)

    contents = read_json(jsonfile)
    if key in contents:
        value = contents[key]
        print(value)
        bot.debug('%s is %s' % (key, value))
    else:
        bot.error("%s is not defined in file. Exiting" % key)
        sys.exit(1)
    return value


def ADD(key, value, jsonfile, force=False, quiet=False):
    '''ADD will write or update a key in a json file
    '''

    # Check that key is not empty
    if key.strip() in ['#', '', None]:
        bot.verbose('Empty key %s, skipping' % key)
        sys.exit(0)

    key = format_keyname(key)
    bot.debug("Adding label: '%s' = '%s'" % (key, value))
    bot.debug("ADD %s from %s" % (key, jsonfile))
    if os.path.exists(jsonfile):
        contents = read_json(jsonfile)
        if key in contents:
            msg = 'Warning, %s is already set. ' % key
            msg += 'Overwrite is set to %s' % force
            if not quiet:
                bot.debug(msg)
            if force is True:
                contents[key] = value
            else:
                msg = '%s found in %s ' % (key, jsonfile)
                msg += 'and overwrite set to %s.' % force
                bot.error(msg)
                sys.exit(1)
        else:
            contents[key] = value
    else:
        contents = {key: value}
    bot.debug('%s is %s' % (key, value))
    write_json(contents, jsonfile)
    return value


def DELETE(key, jsonfile):
    '''DELETE will remove a key from a json file
    '''
    key = format_keyname(key)
    bot.debug("DELETE %s from %s" % (key, jsonfile))
    if not os.path.exists(jsonfile):
        bot.error("Cannot find %s, exiting." % jsonfile)
        sys.exit(1)

    contents = read_json(jsonfile)
    if key in contents:
        del contents[key]
        if len(contents) > 0:
            write_json(contents, jsonfile)
        else:
            bot.debug('%s is empty, deleting.' % jsonfile)
            os.remove(jsonfile)
        return True
    else:
        bot.debug('Warning, %s not found in %s' % (key, jsonfile))
        return False


def INSPECT(inspect_labels=None,
            inspect_def=None,
            inspect_app=None,
            inspect_runscript=None,
            inspect_test=None,
            inspect_help=None,
            inspect_env=None,
            pretty_print=True):
    '''INSPECT will print a "manifest" for an image, with one or more
    variables asked for by the user. The default prints a human readable
    format (text and json) and if pretty_print  is turned to False,
    the entire thing will be returned as json via the JSON API standard.
    The base is checked for the metadata folder, and if it does not exist,
    the links are searched for and parsed (to support older versions).
    For all of the following, each is returned if it exists

    :param inspect_runscript: if not None, will include runscript
    :param inspect_labels: if not None, will include labels
    :param inspect_def: if not None, will include definition file
    :param inspect_test: if not None, will include test
    :param inspect_env: if not None, will include environment
    :param inspect_help: if not None, include helpfile
    :param pretty_print: if False, return all JSON API spec
    '''
    from defaults import (LABELFILE, HELPFILE, RUNSCRIPT,
                          TESTFILE, ENVIRONMENT)

    if inspect_app is not None:
        LABELBASE = "scif/apps/%s/scif/labels.json" % inspect_app
        LABELFILE = LABELFILE.replace('.singularity.d/labels.json', LABELBASE)
        HELPBASE = "scif/apps/%s/scif/runscript.help" % inspect_app
        HELPFILE = HELPFILE.replace('.singularity.d/runscript.help', HELPBASE)
        RUNBASE = "scif/apps/%s/scif/runscript" % inspect_app
        RUNSCRIPT = "%s/%s" % (RUNSCRIPT.strip('singularity'), RUNBASE)

        TESTBASE = "scif/apps/%s/scif/test" % inspect_app
        TESTFILE = TESTFILE.replace(".singularity.d/test", TESTBASE)
        ENVBASE = "scif/apps/%s/scif/" % inspect_app
        ENVIRONMENT = ENVIRONMENT.replace(".singularity.d/", ENVBASE)

    data = dict()
    errors = dict()

    # Labels
    if inspect_labels:
        bot.verbose2("Inspection of labels selected.")
        if os.path.exists(LABELFILE):
            data["labels"] = read_json(LABELFILE)
        else:
            data["labels"] = None
            error_detail = "This container does not have labels"
            errors["labels"] = generate_error(404,
                                              detail=error_detail,
                                              title="Labels Undefined")

    # Helpfile
    if inspect_help:
        bot.verbose2("Inspection of helpfile selected.")
        if os.path.exists(HELPFILE):
            data["help"] = read_file(HELPFILE, readlines=False)
        else:
            data["help"] = None
            error_detail = "This container does not have a helpfile"
            errors["help"] = generate_error(404,
                                            detail=error_detail,
                                            title="Help Undefined")

    # Definition File
    if inspect_def:
        bot.verbose2("Inspection of deffile selected.")
        if os.path.exists(DEFFILE):
            data["deffile"] = read_file(DEFFILE, readlines=False)
        else:
            error_detail = "This container doesn't have a bootstrap recipe"
            data["deffile"] = None
            errors["deffile"] = generate_error(404,
                                               title="Recipe Undefined",
                                               detail=error_detail)

    # Runscript
    if inspect_runscript:
        bot.verbose2("Inspection of runscript selected.")
        if os.path.exists(RUNSCRIPT):
            data["runscript"] = read_file(RUNSCRIPT, readlines=False)
        else:
            error_detail = "This container does not have any runscript defined"
            data["runscript"] = None
            errors["runscript"] = generate_error(404,
                                                 title="Runscript Undefined",
                                                 detail=error_detail)

    # Test
    if inspect_test:
        bot.verbose2("Inspection of test selected.")
        if os.path.exists(TESTFILE):
            data["test"] = read_file(TESTFILE, readlines=False)
        else:
            error_detail = "This container does not have any tests defined"
            data["test"] = None
            errors["test"] = generate_error(404,
                                            title="Tests Undefined",
                                            detail=error_detail)

    # Environment
    if inspect_env:
        bot.verbose2("Inspection of environment selected.")
        if os.path.exists(ENVIRONMENT):
            data["environment"] = read_file(ENVIRONMENT, readlines=False)
        else:
            error_detail = "This container doesn't have environment defined"
            data["environment"] = None
            errors["environment"] = generate_error(404,
                                                   title="Tests Undefined",
                                                   detail=error_detail)

    if pretty_print:
        bot.verbose2("Structured printing specified.")
        for dtype, content in data.items():
            if content is not None:
                if isinstance(content, dict):
                    print_json(content, print_console=True)
                else:
                    bot.info(content)
            else:
                print_json(errors[dtype], print_console=True)

    else:
        bot.verbose2("Unstructed printing specified")
        # Only define a type if there is data to return
        # else return errors
        result = dict()
        if len(data) > 0:
            result["data"] = {"attributes": data,
                              "type": "container"}
        else:
            result["errors"] = []
            for dtype, content in errors.items():
                result["errors"].append(content)
        print_json(result, print_console=True)


def generate_error(status=None, title=None, detail=None):
    '''generate_error will return a JSON API error object,
    intended to be added to the list of "errors" at the top level.
    Errors and data cannot both be returned, and so these data
     structures are only returned given no data.
    :param status: the status code to return (http status codes)
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
    '''format keyname will ensure that all keys
    are uppcase, with no special characters
    '''
    if key.startswith('org.label-schema'):
        return re.sub('[^A-Za-z0-9-.]+', '_', key).lower()
    return re.sub('[^A-Za-z0-9]+', '_', key).upper()
