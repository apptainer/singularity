'''

templates.py: template data structures for Singularity python
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

from message import bot


def get_template(template_name):
    '''get_template will return a default template for some function in
    Singularity Python. This is to reduce redundancy if data structures
    are used multiple times, etc. If there is no template, None is returned.
    '''
    template_name = template_name.lower()
    templates = dict()

    templates['tarinfo'] = {"gid": 0,
                            "uid": 0,
                            "uname": "root",
                            "gname": "root",
                            "mode": 493}

    if template_name in templates:
        bot.debug("Found template for %s" % (template_name))
        return templates[template_name]
    else:
        bot.warning("Cannot find template %s" % (template_name))
    return None
