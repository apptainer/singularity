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

import json
import os
import re
import sys
sys.path.append('..') # parent directory

from utils import write_file, read_file
from logman import logger
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
    eg: CMD /code/run_uwsgi.sh --> /code/run_uwsgi.sh.
    '''
    return "%s" %(cmd)


def parse_entry(cmd):
    '''parse_entry will parse a Dockerfile ENTRYPOINT command to a singularity appropriate one
    eg: ENTRYPOINT /code/run_uwsgi.sh --> exec /code/run_uwsgi.sh.
    '''
    return "exec %s" %(cmd)


def parse_copy(copy_str):
    '''parse_copy will copy a file from one location to another. This likely will need
    tweaking, as the files might need to be mounted from some location before adding to
    the image.
    '''
    return "cp %s" %(copy_str)


def parse_http(url,destination):
    '''parse_http will get the filename of an http address, and return a statement
    to download it to some location
    '''
    file_name = os.path.basename(url)
    download_path = "%s/%s" %(destination,file_name)
    return "curl %s -o %s" %(url,download_path)


def parse_targz(targz,destination):
    '''parse_targz will return a commnd to extract a targz file to a destination.
    '''
    return "tar -xzvf %s %s" %(targz,destination)


def parse_zip(zipfile,destination):
    '''parse_zipfile will return a commnd to unzip a file to a destination.
    '''
    return "unzip %s %s" %(zipfile,destination)


def parse_add(add):
    '''parse_add will copy multiple files from one location to another. This likely will need
    tweaking, as the files might need to be mounted from some location before adding to
    the image. The add command is done for an entire directory.
    :param add: the command to parse
    '''
    # In the case that there are newlines or comments
    command,rest = add.split('\n',1)
    from_thing,to_thing = command.split(" ")

    # People like to use dots for PWD.
    if from_thing == ".":
        from_thing = os.getcwd()
    if to_thing == ".":
        to_thing = os.getcwd()

    # If it's a url or http address, then we need to use wget/curl to get it
    if re.search("^http",from_thing):
        result = parse_http(url=from_thing,
                           destination=to_thing)

    # If it's a tar.gz, then we are supposed to uncompress
    if re.search(".tar.gz$",from_thing):
        result = parse_targz(targz=from_thing,
                             destination=to_thing)

    # If it's .zip, then we are supposed to unzip it
    if re.search(".zip$",from_thing):
        result = parse_zip(zipfile=from_thing,
                         destination=to_thing)

    # Is from thing a directory or something else?
    if os.path.isdir(from_thing):
        result = "cp -R %s %s" %(from_thing,to_thing)
    else:
        result = "cp %s %s" %(from_thing,to_thing)
    return "%s\n%s" %(result,rest)


def parse_workdir(workdir):
    '''parse_workdir will simply cd to the working directory
    '''
    return "cd %s" %(workdir)


def get_mapping():
    '''get_mapping returns a dictionary mapping from a Dockerfile command to a Singularity
    build spec section. Note - this currently ignores lines that we don't know what to do with
    in the context of Singularity (eg, EXPOSE, LABEL, USER, VOLUME, STOPSIGNAL, escape,
    MAINTAINER)

    :: note
    each KEY of the mapping should be a command start in the Dockerfile (eg, RUN)
    for each corresponding value, there should be a dictionary with the following:
    
        - section: the Singularity build file section to write the new command to
        - fun: any function to pass the output through before writing to the section (optional)
        - json: Boolean, if the section can optionally have json (eg a list)

    I'm not sure the subtle differences between add and copy, other than copy doesn't support
    external files. It should suffice for our purposes (for now) to use the same function 
    (parse_add) until evidence for a major difference is determined.
    '''
    #  Docker : Singularity
    add_command = {"section": "%post","fun": parse_add, "json": True }
    copy_command = {"section": "%post", "fun": parse_add, "json": True }  
    cmd_command = {"section": "%runscript", "fun": parse_cmd, "json": True }  
    env_command = {"section": "%post", "fun": parse_env, "json": False }
    from_command = {"section": "From", "json": False }
    run_command = {"section": "%post", "json": True}       
    workdir_command = {"section": "%post","fun": parse_workdir, "json": False }  
    entry_command = {"section": "%post", "fun": parse_entry, "json": True }

    return {"ADD": add_command,
            "COPY":copy_command,
            "CMD":cmd_command,
            "ENTRYPOINT":entry_command,
            "ENV": env_command,
            "FROM": from_command,
            "RUN":run_command,
            "WORKDIR":workdir_command}
           
    

def dockerfile_to_singularity(dockerfile_path, output_dir=None):
    '''dockerfile_to_singularity will return a Singularity build file based on
    a provided Dockerfile. If output directory is not specified, the string
    will be returned. Otherwise, a file called Singularity will be written to 
    output_dir
    :param dockerfile_path: the path to the Dockerfile
    :param output_dir: the output directory to write the Singularity file to
    '''
    if os.path.basename(dockerfile_path) == "Dockerfile":
        spec = read_file(dockerfile_path)
        # Use a common mapping
        mapping = get_mapping()
   
        # Put into dict of keys (section titles) and list of commands (values)
        sections = organize_sections(lines=spec,
                                     mapping=mapping)

        # We have to, by default, add the Docker bootstrap
        sections["bootstrap"] = ["docker"]

        # Put into one string based on "order" variable in mapping
        build_file = print_sections(sections=sections,
                                    mapping=mapping)
        if output_dir != None:
            write_file("%s/Singularity" %(output_dir),build_file)
            print("Singularity spec written to %s" %(output_dir))
        return build_file

    # If we make it here, something didn't work
    logger.error("Could not find %s, exiting.", dockerfile_path)
    return sys.exit(1)


def organize_sections(lines,mapping=None):
    '''organize_sections will break apart lines from a Dockerfile, and put into 
    appropriate Singularity sections.
    :param lines: the raw lines from the Dockerfile
    :mapping: a dictionary mapping Docker commands to Singularity sections
    '''
    if mapping == None:
        mapping = get_mapping()
    sections = dict()
    startre = "|".join(["^%s" %x for x in mapping.keys()])
    command = None
    name = None
    while len(lines) > 0:
        line = lines.pop(0)
        # Do we have a new line/section?
        if re.search(startre,line):
            # Parse the last section, and start over
            if command != None and name != None:
                sections = parse_section(name=name,
                                         command=command,
                                         mapping=mapping,
                                         sections=sections)
            name,command = line.split(" ",1)
        # We have a continuation of the last command or an empty line
        else:
            command = "%s %s" %(command,line)

    return sections

def parse_section(sections,name,command,mapping=None):
    '''parse_section will take a command that has lookup key "name" as a key in "mapping"
    and add a line to the list of each in sections that will be rendered into a Singularity
    build file.
    :param sections: the current sections, a dictionary of keys (singularity section titles)
    and a list of lines.
    :param name: the name of the section to add
    :param command: the command to parse:
    :param mapping: the mapping object to use
    '''
    if mapping == None:
        mapping = get_mapping()
    if name in mapping:
        build_section = mapping[name]['section']
        # Can the command potentially be json (a list?)
        if mapping[name]['json']:
            try:
                command = " ".join(json.loads(command))
            except:
                pass 
        # Do we need to pass it through a function first?
        if 'fun' in mapping[name]:
            command = mapping[name]['fun'](command)
        # Add to our dictionary of sections!
        if build_section not in sections:
            sections[build_section] = [command]
        else:
            sections[build_section].append(command)
    return sections


def print_sections(sections,mapping=None):
    '''print_sections will take a sections object (dict with section names and
    list of commands) and parse into a common string, to output to file or return
    to user.
    :param sections: output from organize_sections
    :mapping: a dictionary mapping Docker commands to Singularity sections
    '''
    if mapping == None:
        mapping = get_mapping()
    finished_spec = ""
    ordering = ['bootstrap',"From","%runscript","%post"]

    for section in ordering:

        # Was the section found in the file?
        if section in sections:
            if not re.search("^%",section):
                # A single command, intended to go after a colon (yaml)
                content = "".join(sections[section])
                finished_spec = "%s\n%s:%s" %(finished_spec,section,content)
            else:
                # A list of things to join, after the section header
                content = "".join(sections[section])
                finished_spec = "%s\n%s\n%s" %(finished_spec,section,content)

    return finished_spec
