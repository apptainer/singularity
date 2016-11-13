#!/usr/bin/env python

'''

converted.py: Parse a Dockerfile into a Singularity spec file

Copyright (c) 2016, Vanessa Sochat. All rights reserved. 

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

import re
import sys
sys.path.append('..') # parent directory

from utils import write_file, read_file
import json

# Parsing functions ---------------------------------------------------------------

def parse_env(env):
    '''parse_env will parse a Dockerfile ENV command to a singularity appropriate one
    eg: ENV PYTHONBUFFER 1 --> export PYTHONBUFFER=1
    ::note  This has to handle multiple exports per line. In the case of having an =,
    It could be that we have more than one pair of variables. If no equals, then
    we probably don't. See:
    see: https://docs.docker.com/engine/reference/builder/#/env
    '''
    # If the user has "=" then we can have more than one export per line
    exports = []
    name = None
    value = None
    if re.search("=",env):
        pieces = [p for p in re.split("( |\\\".*?\\\"|'.*?')", env) if p.strip()]
        while len(pieces) > 0:
            contender = pieces.pop(0)
            # If there is an equal, we've found a name
            if re.search("=",contender):
                if name != None:
                    exports.append(join_env(name,value))
                name = contender         
                value = None
            else:
                if value == None:
                    value = contender
                else:
                    value = "%s %s" %(value,contender)
        exports.append(join_env(name,value))
        return "\n".join(exports)

    # otherwise, the rule is one per line
    else: 
        name,value = re.split(' ',env,1)
        return "export %s=%s" %(name,value)
    

def join_env(name,value):
    # If it's the end of the string, we don't want a space
    if re.search("=$",name):
        return "export %s%s" %(name,value)
    return "export %s %s" %(name,value)


def parse_cmd(cmd):
    '''parse_cmd will parse a Dockerfile CMD command to a singularity appropriate one
    eg: CMD /code/run_uwsgi.sh --> exec /code/run_uwsgi.sh
    '''
    return "exec %s" %(cmd)


def parse_copy(copy_str):
    '''parse_copy will copy a file from one location to another. This likely will need
    tweaking, as the files might need to be mounted from some location before adding to
    the image.
    '''
    return "cp %s" %(copy_str)


def parse_add(add)
    '''parse_add will copy multiple files from one location to another. This likely will need
    tweaking, as the files might need to be mounted from some location before adding to
    the image. The add command is done for an entire directory.
    :param add: the command to parse
    '''
    from_thing,to_thing = add.split(" ")

    # If it's a url or http address, then we need to use wget/curl to get it
    if re.search("^http",from_thing):
        return "curl %s -o %s" %(from_thing,to_thing)

    # People like to use dots for PWD.
    if from_thing == ".":
        from_thing = os.getcwd()
    if to_thing == ".":
        to_thing = os.getcwd()

    # Is from thing a directory or something else?
    if os.path.isdir(from_thing):
        return "cp -R %s %s" %(from_thing,to_thing)

    print("Cannot determine add command for %s, skipping" %(add))     
    return ""


def parse_workdir(workdir)
    '''parse_workdir will simply cd to the working directory
    '''
    return "cd %s" %(workdir)


def get_mapping():
    '''get_mapping returns a dictionary mapping from a Dockerfile command to a Singularity
    build spec section. Note - this currently ignores lines that we don't know what to do with
    in the context of Singularity (eg, EXPOSE, LABEL, USER, VOLUME, STOPSIGNAL)
    '''
    #  Docker : Singularity
    add_command = {"section": "%post", "fun": parse_copy }  
    copy_command = {"section": "%post", "fun": parse_copy }  
    cmd_command = {"section": "%runscript", "fun": parse_cmd }  
    env_command = {"section": "%post", "fun": parse_env } 
    from_command = {"section": "From"}
    run_command = {"section": "%post"}       
    workdir_command = {"section": "%post", "fun": parse_workdir }  

    return {"ADD": add_command,
            "ENV": env_command,
            "FROM": from_command,
            "WORKDIR":workdir_command}
           
    # STOPPING FOR TODAY - remainder of parsing functions need to be written, tested,
    # and the mapping finished, and then the mapping used in organize_sections!


def dockerfile_to_singularity(dockerfile_path, output_dir):
    '''dockerfile_to_singularity will return a Singularity build file based on
    a provided Dockerfile
    :param dockerfile_path: the path to the Dockerfile
    :param output_dir: the output directory to write the Singularity file to
    '''
    if os.path.basename(dockerfile_path) == "Dockerfile":
        spec = read_file(dockerfile_path)


    # If we make it here, something didn't work
    return sys.exit(1)


def organize_sections(lines,mapping=None):
    '''organize_sections will break apart lines from a Dockerfile, and put into 
    appropriate Singularity sections.
    :param lines: the raw lines from the Dockerfile
    :mapping: a dictionary mapping Docker commands to Singularity sections
    '''

    #TODO: read in lines until we reach the next section.
    # If section isn't in list, or is, parse to one or the other (FOR: RUN,CMD)
    # send entire section to be parsed and put (as a unit) into an object


def sniff_command(line):
    '''sniff_command will return the command type and command for one or more lines
    :param line: the line to read

