#!/usr/bin/python

import os
import re
import sys

define_re = re.compile("#define ([A-Z_]+) (.*)")

defaultfile = open(sys.argv[1], "r")
infile = open(sys.argv[2], "r")
outfile = open(sys.argv[3] + ".tmp", "w")

data = infile.read()

defaults = {}
for line in defaultfile:
    m = define_re.match(line)
    if m:
        key, value = m.groups()
        defaults[key] = value

for key, value in defaults.items():
    new_value = value.replace('"', '')
    if new_value == "1":
        new_value = "yes"
    elif new_value == "0":
        new_value = "no"
    data = data.replace("@" + key + "@", new_value)

outfile.write(data)
outfile.close()
os.rename(sys.argv[3] + ".tmp", sys.argv[3])

