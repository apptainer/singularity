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

from message import bot
import json
import re


def HELP(filepath, pretty_print=False):
    '''HELP will print a help file to the console for a user,
    if the file exists. If pretty print is True, the file will be
    parsed into json.
    :param filepath: the file path to show
    :param pretty_print: if False, return all JSON API spec
    '''

    # Definition File
    if os.path.exists(filepath):
        bot.verbose2("Printing help")
        text = read_file(filepath, readlines=False)

        if pretty_print:
            bot.verbose2("Structured printing specified.")
            text = {"org.label-schema.usage.singularity.runscript.help": text}
            print_json(text, print_console=True)
        else:
            bot.info(text)
    else:
        bot.verbose2("Container does not have runscript.help")
