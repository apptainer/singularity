#!/usr/bin/awk -f
# Copyright (c) 2016-2018, Yannick Cote <yanick@divyan.org>. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be found
# in the LICENSE file.

# check if we are still reading keyword values or reached a new keyword
function getkeyword(words)
{
	iskey = 0

	if (words[1] != "") {
		for (k in keywords) {
			if (words[1] == keywords[k])
				iskey = 1
		}
		if (iskey == 1) {
			current = words[1]
		} else {
			print "error:", words[1], "is not a keyword"
			exit(1)
		}
	}
}

# for a keyword (name, src, cflags, etc.) read its values
function getvalues(mod, tags, words, current, nfields)
{
	for (j = 2; j <= nfields; j++) {
		if (tags[mod, current] != "")
			tags[mod, current] = tags[mod, current] " "
		tags[mod, current] = tags[mod, current] words[j]
	}
}

# generate object list from [a,c]src,win_[a,c]src,unix_[a,c]src for each module
function gensrcs(mod, tags)
{
	prefix = ""

	# first "[a,c]src"
	split(tags[mod, "csrc"], csrcs, " ")
	tags[mod, "csrc"] = ""
	for (s in csrcs) {
		if (tags[mod, "csrc"] != "")
			tags[mod, "csrc"] = tags[mod, "csrc"] " "
		tags[mod, "csrc"] = tags[mod, "csrc"] prefix csrcs[s]
	}
	split(tags[mod, "asrc"], asrcs, " ")
	tags[mod, "asrc"] = ""
	for (s in asrcs) {
		if (tags[mod, "asrc"] != "")
			tags[mod, "asrc"] = tags[mod, "asrc"] " "
		tags[mod, "asrc"] = tags[mod, "asrc"] prefix asrcs[s]
	}

	# then unix_[a,c]src
	split(tags[mod, "unix_csrc"], csrcs, " ")
	tags[mod, "unix_csrc"] = ""
	for (s in csrcs) {
		if (tags[mod, "unix_csrc"] != "")
			tags[mod, "unix_csrc"] = tags[mod, "unix_csrc"] " "
		tags[mod, "unix_csrc"] = tags[mod, "unix_csrc"] prefix csrcs[s]
	}
	split(tags[mod, "unix_asrc"], asrcs, " ")
	tags[mod, "unix_asrc"] = ""
	for (s in asrcs) {
		if (tags[mod, "unix_asrc"] != "")
			tags[mod, "unix_asrc"] = tags[mod, "unix_asrc"] " "
		tags[mod, "unix_asrc"] = tags[mod, "unix_asrc"] prefix asrcs[s]
	}

	# finaly win_[a,c]src
	split(tags[mod, "win_asrc"], asrcs, " ")
	tags[mod, "win_asrc"] = ""
	for (s in asrcs) {
		if (tags[mod, "win_asrc"] != "")
			tags[mod, "win_asrc"] = tags[mod, "win_asrc"] " "
		tags[mod, "win_asrc"] = tags[mod, "win_asrc"] prefix asrcs[s]
	}
}

# generate object list from [a,c]src,win_[a,c]src,unix_[a,c]src for each module
function genobjs(mod, tags)
{
	# first "[a,c]obj"
	split(tags[mod, "csrc"], objs, " ")
	for (o in objs) {
		gsub(/\.c$/, ".o", objs[o])
		if (tags[mod, "cobj"] != "")
			tags[mod, "cobj"] = tags[mod, "cobj"] " "
		tags[mod, "cobj"] = tags[mod, "cobj"] "$(BUILDDIR)/" objs[o]
	}
	split(tags[mod, "asrc"], objs, " ")
	for (o in objs) {
		gsub(/\.S$/, ".o", objs[o])
		if (tags[mod, "aobj"] != "")
			tags[mod, "aobj"] = tags[mod, "aobj"] " "
		tags[mod, "aobj"] = tags[mod, "aobj"] "$(BUILDDIR)/" objs[o]
	}

	# then "unix_[a,c]obj"
	split(tags[mod, "unix_csrc"], objs, " ")
	for (o in objs) {
		gsub(/\.c$/, ".o", objs[o])
		if (tags[mod, "unix_cobj"] != "")
			tags[mod, "unix_cobj"] = tags[mod, "unix_cobj"] " "
		tags[mod, "unix_cobj"] = tags[mod, "unix_cobj"] "$(BUILDDIR)/" objs[o]
	}
	split(tags[mod, "unix_asrc"], objs, " ")
	for (o in objs) {
		gsub(/\.S$/, ".o", objs[o])
		if (tags[mod, "unix_aobj"] != "")
			tags[mod, "unix_aobj"] = tags[mod, "unix_aobj"] " "
		tags[mod, "unix_aobj"] = tags[mod, "unix_aobj"] "$(BUILDDIR)/" objs[o]
	}

	# finaly "win_[a,c]obj"
	split(tags[mod, "win_csrc"], objs, " ")
	for (o in objs) {
		gsub(/\.c$/, ".o", objs[o])
		if (tags[mod, "win_cobj"] != "")
			tags[mod, "win_cobj"] = tags[mod, "win_cobj"] " "
		tags[mod, "win_cobj"] = tags[mod, "win_cobj"] "$(BUILDDIR)/" objs[o]
	}
	split(tags[mod, "win_asrc"], objs, " ")
	for (o in objs) {
		gsub(/\.S$/, ".o", objs[o])
		if (tags[mod, "win_aobj"] != "")
			tags[mod, "win_aobj"] = tags[mod, "win_aobj"] " "
		tags[mod, "win_aobj"] = tags[mod, "win_aobj"] "$(BUILDDIR)/" objs[o]
	}
}

function gentarget(mod, tags)
{
	if (tags[mod, "prog"] != "") {
		# generate target for a program
		tags[mod, "target"] = tags[mod, "prog"]
		if (envar["host"] == "windows")
			tags[mod, "target"] = tags[mod, "target"] ".exe"
	} else if (tags[mod, "lib"] != "") {
		# generate target for a library
		tags[mod, "target"] = "lib" tags[mod, "lib"]
	} else {
		# generate target for a simple list of objects
		tags[mod, "target"] = tags[mod, "name"] "_OBJ"
	}
}

# for a module.conf file, read and lex all keyword/values pair
function scanmod(mod, path)
{
	module = path "/" mod ".parsed"
	while (getline < module > 0) {
		n = split($0, words, " *:= *| *\\ *|[ \t]*")
		if (n > 0) {
			getkeyword(words)
			getvalues(mod, tags, words, current, n)
		}
	}
}

function usage()
{
	print "usage: genmod modfile=<module file> genconfdir=<temp confdir> host=<host type> tmpldir=<template location>"
	exit(1)
}

# print all keyword vars and their values for all project modules
function printtags(tags)
{
	reset_file("/tmp/tags")
	for (m in modules) {
		for (k in keywords) {
			if (tags[modules[m], keywords[k]] == "")
				continue
			printf("%s:%s [%s]\n", modules[m], keywords[k],
			       tags[modules[m], keywords[k]]) >> "/tmp/tags"
		}
		print "" >> "/tmp/tags"
	}
	printf("cleanfiles: [%s]\n", cleanfiles) >> "/tmp/tags"
}

function reset_file(file)
{
	printf("") > file
}

function put_objlist(name, cobj, aobj, f)
{
	printf("# object files list\n") >> f
	printf("%s_OBJ := \\\n", name) >> f
	
	split(cobj, objs, " ")
	for (o in objs) {
		printf("\t%s \\\n", objs[o]) >> f
		cleanfiles = cleanfiles " " objs[o] " " objs[o] ".d"
	}
	split(aobj, objs, " ")
	for (o in objs) {
		printf("\t%s \\\n", objs[o]) >> f
		cleanfiles = cleanfiles " " objs[o] " " objs[o] ".d"
	}
	print "" >> f
}

function put_suffix_rules(template, cobj, aobj, cflags, f)
{
	printf("# suffix rules (metarules missing from most variants)\n") >> f

	split(cobj, objs, " ")
	for (o in objs) {
		# prepare the source file name `s' out of `o'
		s = objs[o]
		gsub(/\.o$/, ".c", s)
		gsub(/^\$\(BUILDDIR\)\//, "", s)

		while (getline < template > 0) {
			gsub(/__OBJ__/, objs[o], $0)
			gsub(/__SRC__/, s, $0)
			gsub(/__CFLAGS__/, cflags, $0)
			# write the result down in the current fragment
			if ($0 != "")
				printf("%s\n", $0) >> f
		}
		close(template)
	}
	split(aobj, objs, " ")
	for (o in objs) {
		# prepare the source file name `s' out of `o'
		s = objs[o]
		gsub(/\.o$/, ".S", s)
		gsub(/^\$\(BUILDDIR\)\//, "", s)

		while (getline < template > 0) {
			gsub(/__OBJ__/, objs[o], $0)
			gsub(/__SRC__/, s, $0)
			gsub(/__CFLAGS__/, cflags, $0)
			# write the result down in the current fragment
			if ($0 != "")
				printf("%s\n", $0) >> f
		}
		close(template)
	}

	print "" >> f
}

function gendeps_link(modules, moduledirs, idep, imod, tags)
{
	# dependency is a program nothing to link with
	if (tags[modules[idep], "prog"] != "")
		return

	# dependency is a lib, generate library link rules
	if (tags[modules[idep], "lib"] != "") {
		if (tags[modules[imod], "deps_link"] != "")
			tags[modules[imod], "deps_link"] = tags[modules[imod], "deps_link"] " "
		tags[modules[imod], "deps_link"] = tags[modules[imod], "deps_link"] "-L$(BUILDDIR)/" moduledirs[idep] " -l" tags[modules[idep], "lib"]
	} else {
		# dependency is just an object list
		if (tags[modules[imod], "deps_link"] != "")
			tags[modules[imod], "deps_link"] = " " tags[modules[imod], "deps_link"]
		tags[modules[imod], "deps_link"] = "$(" tags[modules[idep], "target"] ")" tags[modules[imod], "deps_link"]
	}
}

function gendeps(modules, moduledirs, idx, tags)
{
	split(tags[modules[idx], "depends"], deps, " ")
	for (d in deps) {
		found = 0
		for (m in modules) {
			if (tags[modules[m], "name"] == deps[d]) {
				gendeps_link(modules, moduledirs, m, idx, tags)
				if (tags[modules[idx], "deps_target"] != "")
					tags[modules[idx], "deps_target"] = tags[modules[idx], "deps_target"] " "
				tags[modules[idx], "deps_target"] = tags[modules[idx], "deps_target"] "$(" tags[modules[m], "target"] ")"
				found = 1
			}
		}
		# if dependency is NOT a module name but just a verbatim expression to paste in place
		if(found == 0)
			tags[modules[idx], "deps_target"] = tags[modules[idx], "deps_target"] deps[d]
	}
}

function put_prog(template, target, path, name, deps_t, deps_l, ldflags, extralibs, f)
{
	prefix = ""

	printf("# link the program `%s'\n", target) >> f
	while (getline < template > 0) {
		gsub(/__TARGET__/, target, $0)
		gsub(/__PATH__/, path, $0)
		gsub(/__NAME__/, name, $0)
		gsub(/__DEPEND_T__/, deps_t, $0)
		gsub(/__DEPEND_L__/, deps_l, $0)
		gsub(/__LDFLAGS__/, ldflags, $0)
		gsub(/__EXTRALIBS__/, extralibs, $0)
		printf("%s\n", $0) >> f
	}
	close(template)
	print "" >> f

	cleanfiles = cleanfiles " " "$(" target ")"
}

function put_lib(template, target, path, name, f)
{
	printf("# create lib `%s'\n", target) >> f
	while (getline < template > 0) {
		gsub(/__TARGET__/, target, $0)
		gsub(/__PATH__/, path, $0)
		gsub(/__NAME__/, name, $0)
		printf("%s\n", $0) >> f
	}
	close(template)
	print "" >> f

	cleanfiles = cleanfiles " " "$(" target ")"
}

# generate 1 .mk file for specified module -- to be inlined in top Makefile
function put_mkfile(mod, moddir, tags, f)
{
	# gather objects from C files
	cob = tags[mod, "cobj"]
	if (envar["host"] == "unix")
		cob = cob " " tags[mod, "unix_cobj"]
	if (envar["host"] == "windows")
		cob = cob " " tags[mod, "win_cobj"]
	# gather objects from assembly .S files
	aob = tags[mod, "aobj"]
	if (envar["host"] == "unix")
		aob = aob " " tags[mod, "unix_aobj"]
	if (envar["host"] == "windows")
		aob = aob " " tags[mod, "win_aobj"]

	# write list of objects to build
	put_objlist(tags[mod, "name"], cob, aob, f)

	# if the module is a program, write link rules
	if (tags[mod, "prog"] != "")
		put_prog(envar["tmpldir"] "/" "prog.tmpl", tags[mod, "target"], moddir, tags[mod, "name"], tags[mod, "deps_target"], tags[mod, "deps_link"], tags[mod, "ldflags"], tags[mod, "extralibs"], f)

	# if the module is a library, write lib creation rules
	if (tags[mod, "lib"] != "")
		put_lib(envar["tmpldir"] "/" "lib.tmpl", tags[mod, "target"], moddir, tags[mod, "name"], f)

	# write each object suffix build rules
	put_suffix_rules(envar["tmpldir"] "/" "suffix.tmpl", cob, aob, tags[mod, "cflags"], f)
}

function genallrule(mods, moddirs, tags, f)
{
	reset_file(envar["genconfdir"] "/modules.mk")

	printf("all:") > f
	for (m in mods) {
		if (tags[mods[m], "prog"] == "" && tags[mods[m], "lib"] == "") {
			printf(" $(%s)", tags[mods[m], "target"]) >> f
			put_mkfile(mods[m], moddirs[m], tags, envar["genconfdir"] "/modules.mk")
		}
	}
	for (m in mods) {
		if (tags[mods[m], "prog"] == "" && tags[mods[m], "lib"] != "") {
			printf(" $(%s)", tags[mods[m], "target"]) >> f
			put_mkfile(mods[m], moddirs[m], tags, envar["genconfdir"] "/modules.mk")
		}
	}
	for (m in mods) {
		if (tags[mods[m], "prog"] != "" && tags[mods[m], "lib"] == "") {
			printf(" $(%s)", tags[mods[m], "target"]) >> f
			put_mkfile(mods[m], moddirs[m], tags, envar["genconfdir"] "/modules.mk")
		}
	}
	printf("\n\n") >> f
	printf("CLEANFILES += %s\n", cleanfiles) >> f
	printf("\n") >> f
}

function checkvars(envar)
{
	if (envar["modfile"] == "")
		usage()
	if (envar["host"] == "")
		usage()
	if (envar["genconfdir"] == "")
		usage()
	if (envar["tmpldir"] == "")
		usage()
}

# main entry
BEGIN {
	# variable defs
	modules[0] = ""
	moduledirs[0] = ""
	nmods = 0
	tags[0] = ""
	words[0] = ""
	current = ""
	envar[0] = ""
	cleanfiles = ""
	klist = "name prog lib csrc win_csrc unix_csrc depends cflags ldflags"
	klist = klist " " "cobj unix_cobj win_cobj target deps_target deps_link"
	klist = klist " " "asrc win_asrc unix_asrc aobj unix_aobj win_aobj"
	klist = klist " " "extralibs"

	# init keywords
	split(klist, keywords, " ")

	# collect program environment vars from command line ARGV array
	for (i = 0; i < ARGC; i++) {
		n = split(ARGV[i], args, "=")
		if (n == 2)
			envar[args[1]] = args[2]
	}

	# check that we were called with all needed environment vars
	checkvars(envar)

	while (getline < envar["modfile"] > 0) {
		modules[nmods] = $1 "/" $2
		moduledirs[nmods] = $1
		nmods++
	}

	for (i = 0; i < nmods; i++) {
		scanmod(modules[i], envar["genconfdir"])
		gensrcs(modules[i], tags)
		genobjs(modules[i], tags)
		gentarget(modules[i], tags)
	}

	for (i = 0; i < nmods; i++) {
		gendeps(modules, moduledirs, i, tags)
	}

	genallrule(modules, moduledirs, tags, envar["genconfdir"] "/" "all.mk")

#	printtags(tags)
}
