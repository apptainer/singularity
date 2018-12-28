#!/usr/bin/awk -f
# Copyright (c) 2015-2018, Yannick Cote <yhcote@gmail.com>. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be found
# in the LICENSE file.

function usage()
{
	print "usage: genmod mconflist=<module file> makeitgendir=<temp confdir>"
	print "       host=<host type> tmpldir=<template location>"
	exit(1)
}

# remove extra spaces from a string
function trim(str)
{
	gsub(/  +/, " ", str)
	gsub(/ +$/, "", str)
	gsub(/^ +/, "", str)

	return str
}

# print all keyword vars and their values for all project modules
function printmvars()
{
	reset_file("/tmp/mvars")
	for (m in mconfs) {
		for (k in keywords) {
			if (mvars[mconfs[m], keywords[k]] == "")
				continue
			printf("%s:%s [%s]\n", mconfs[m], keywords[k],
			       mvars[mconfs[m], keywords[k]]) >> "/tmp/mvars"
		}
		print "" >> "/tmp/mvars"
	}
}

# truncate file
function reset_file(file)
{
	printf("") > file
}

# check if we are still reading keyword values or reached a new keyword
function getkeyword()
{
	iskey = 0

	if (words[1] != "") {
		for (k in keywords) {
			if (words[1] == keywords[k])
				iskey = 1
		}
		if (iskey == 1) {
			currkeywd = words[1]
		} else {
			print "error:", words[1], "is not a keyword"
			exit(1)
		}
	}
}

# for a keyword (name, src, cflags, etc.) read its values
function getvalues(mconf, nfields)
{
	for (j = 2; j <= nfields; j++) {
		mvars[mconf, currkeywd] = mvars[mconf, currkeywd] " " words[j]
	}
	mvars[mconf, currkeywd] = trim(mvars[mconf, currkeywd])
}

# this routine reads and parses all mconf variables from one "mconf" file
function scanmconf(mconf, makeitgendir)
{
	m = makeitgendir "/" mconf ".parsed"
	while (getline < m > 0) {
		n = split($0, words, " *:= *| *\\ *|[ \t]*")
		if (n > 0) {
			getkeyword()
			getvalues(mconf, n)
		}
	}
}

# generate object list from [a,c]src,win_[a,c]src,unix_[a,c]src for each module
function genobjs(mconf)
{
	# first "[a,c]obj"
	split(mvars[mconf, "csrc"], objs, " ")
	for (o in objs) {
		gsub(/\.c$/, ".o", objs[o])
		mvars[mconf, "cobj"] = mvars[mconf, "cobj"] "$(BUILDDIR)/" objs[o] " "
		mvars[mconf, "cleanfiles"] = mvars[mconf, "cleanfiles"] " " "$(BUILDDIR)/" objs[o]
		mvars[mconf, "cleanfiles"] = mvars[mconf, "cleanfiles"] " " "$(BUILDDIR)/" objs[o] ".d"
	}

	split(mvars[mconf, "asrc"], objs, " ")
	for (o in objs) {
		gsub(/\.S$/, ".o", objs[o])
		mvars[mconf, "aobj"] = mvars[mconf, "aobj"] "$(BUILDDIR)/" objs[o] " "
		mvars[mconf, "cleanfiles"] = mvars[mconf, "cleanfiles"] " " "$(BUILDDIR)/" objs[o]
		mvars[mconf, "cleanfiles"] = mvars[mconf, "cleanfiles"] " " "$(BUILDDIR)/" objs[o] ".d"
	}

	# then "unix_[a,c]obj"
	split(mvars[mconf, "unix_csrc"], objs, " ")
	for (o in objs) {
		gsub(/\.c$/, ".o", objs[o])
		mvars[mconf, "unix_cobj"] = mvars[mconf, "unix_cobj"] "$(BUILDDIR)/" objs[o] " "
		mvars[mconf, "cleanfiles"] = mvars[mconf, "cleanfiles"] " " "$(BUILDDIR)/" objs[o]
		mvars[mconf, "cleanfiles"] = mvars[mconf, "cleanfiles"] " " "$(BUILDDIR)/" objs[o] ".d"
	}

	split(mvars[mconf, "unix_asrc"], objs, " ")
	for (o in objs) {
		gsub(/\.S$/, ".o", objs[o])
		mvars[mconf, "unix_aobj"] = mvars[mconf, "unix_aobj"] "$(BUILDDIR)/" objs[o] " "
		mvars[mconf, "cleanfiles"] = mvars[mconf, "cleanfiles"] " " "$(BUILDDIR)/" objs[o]
		mvars[mconf, "cleanfiles"] = mvars[mconf, "cleanfiles"] " " "$(BUILDDIR)/" objs[o] ".d"
	}

	# then "win_[a,c]obj"
	split(mvars[mconf, "win_csrc"], objs, " ")
	for (o in objs) {
		gsub(/\.c$/, ".o", objs[o])
		mvars[mconf, "win_cobj"] = mvars[mconf, "win_cobj"] "$(BUILDDIR)/" objs[o] " "
		mvars[mconf, "cleanfiles"] = mvars[mconf, "cleanfiles"] " " "$(BUILDDIR)/" objs[o]
		mvars[mconf, "cleanfiles"] = mvars[mconf, "cleanfiles"] " " "$(BUILDDIR)/" objs[o] ".d"
	}

	split(mvars[mconf, "win_asrc"], objs, " ")
	for (o in objs) {
		gsub(/\.S$/, ".o", objs[o])
		mvars[mconf, "win_aobj"] = mvars[mconf, "win_aobj"] "$(BUILDDIR)/" objs[o] " "
		mvars[mconf, "cleanfiles"] = mvars[mconf, "cleanfiles"] " " "$(BUILDDIR)/" objs[o]
		mvars[mconf, "cleanfiles"] = mvars[mconf, "cleanfiles"] " " "$(BUILDDIR)/" objs[o] ".d"
	}

	# finally "data"
	split(mvars[mconf, "data"], objs, " ")
	for (o in objs) {
		objs[o] = objs[o] ".bin"
		mvars[mconf, "dobj"] = mvars[mconf, "dobj"] objs[o] " "
		mvars[mconf, "cleanfiles"] = mvars[mconf, "cleanfiles"] " " "$(BUILDDIR)/" objs[o]
	}
}

# create the "target" mconf variable based on kind (prog, lib, obj list)
function gentarget(mconf)
{
	if (mvars[mconf, "prog"] != "") {
		# generate target for a program
		mvars[mconf, "target"] = mvars[mconf, "prog"]
		if (envar["host"] == "windows")
			mvars[mconf, "target"] = mvars[mconf, "target"] ".exe"
	} else if (mvars[mconf, "lib"] != "") {
		# generate target for a library
		mvars[mconf, "target"] = "lib" mvars[mconf, "lib"]
	} else if (mvars[mconf, "data"] != "") {
		# generate target for embedded data
		mvars[mconf, "target"] = mvars[mconf, "name"] ".bin.o"
	} else {
		# generate target for a simple list of objects
		mvars[mconf, "target"] = mvars[mconf, "name"] "_OBJ"
	}
}

# create a list of "deps_link" list based on other mconf target this mconf needs to link with
function gendeps_link(mconf, idx)
{
	# dependency is a program nothing to link with
	if (mvars[mconfs[mconf], "prog"] != "")
		return

	# dependency is a lib, generate library link rules
	if (mvars[mconfs[mconf], "lib"] != "") {
		mvars[mconfs[idx], "deps_link"] = mvars[mconfs[idx], "deps_link"] " " \
			"-L$(BUILDDIR)/" mconfsdirs[mconf] " -l" mvars[mconfs[mconf], "lib"]
	} else if (mvars[mconfs[mconf], "data"] != "") {
		mvars[mconfs[idx], "extralibs"] = "$(" mvars[mconfs[mconf], "target"] ")" " " \
			mvars[mconfs[idx], "extralibs"]
	} else {
		# dependency is just an object list
		mvars[mconfs[idx], "deps_link"] = "$(" mvars[mconfs[mconf], "target"] ")" " " \
			mvars[mconfs[idx], "deps_link"]
	}
}

# create a "deps_target" list based on other mconf targets this mconf depends on
function gendeps(idx)
{
	split(mvars[mconfs[idx], "depends"], deps, " ")
	for (d in deps) {
		found = 0
		for (m in mconfs) {
			if (mvars[mconfs[m], "name"] == deps[d]) {
				gendeps_link(m, idx)
				mvars[mconfs[idx], "deps_target"] = mvars[mconfs[idx], "deps_target"]  " " \
				     "$(" mvars[mconfs[m], "target"] ")"
				found = 1
			}
		}
		# if dependency is NOT a module name but just a verbatim expression to paste in place
		if (found == 0) {
			mvars[mconfs[idx], "deps_target"] = mvars[mconfs[idx], "deps_target"] " " deps[d]
		}
	}
	mvars[mconfs[idx], "deps_link"] = trim(mvars[mconfs[idx], "deps_link"])
	mvars[mconfs[idx], "deps_target"] = trim(mvars[mconfs[idx], "deps_target"])
}

# output all object lists rules for a specific .mconf file
function put_objlist(mconf, cobj, aobj, output)
{
	printf("# object files list\n") >> output
	printf("%s_OBJ := \\\n", mvars[mconf, "name"]) >> output
	
	split(cobj, objs, " ")
	for (o in objs) {
		printf("\t%s \\\n", objs[o]) >> output
	}
	split(aobj, objs, " ")
	for (o in objs) {
		printf("\t%s \\\n", objs[o]) >> output
	}
	split(mvars[mconf, "dobj"], objs, " ")
	for (o in objs) {
		printf("\t%s \\\n", objs[o]) >> output
	}
	print "" >> output
}

# output all suffix (build) rules for a specific .mconf file
function put_suffix_rules(cobj, aobj, mconf, output)
{
	printf("# suffix rules (metarules missing from most variants)\n") >> output

	split(cobj, objs, " ")
	for (o in objs) {
		# prepare the source file name `s' out of `o'
		s = objs[o]
		gsub(/\.o$/, ".c", s)
		gsub(/^\$\(BUILDDIR\)\//, "", s)

		# fix up the target template when building generated source files
		if (match(s, /^\$\(BUILDDIR\)\//) == 1) {
			tmpl = envar["tmpldir"] "/" "suffix_bldir.tmpl"
			n = split(s, gen, "/")
			while (getline < tmpl > 0) {
				gsub(/__OBJ__/, objs[o], $0)
				gsub(/__SRC__/, s, $0)
				gsub(/__GENSRC__/, "[GEN] " gen[n], $0)
				gsub(/__CFLAGS__/, mvars[mconf, "cflags"], $0)
				# write the result down in the current fragment
				if ($0 != "") {
					$0 = trim($0)
					printf("%s\n", $0) >> output
				}
			}
			close(tmpl)
		} else {
			tmpl = envar["tmpldir"] "/" "suffix.tmpl"
			while (getline < tmpl > 0) {
				gsub(/__OBJ__/, objs[o], $0)
				gsub(/__SRC__/, s, $0)
				gsub(/__CFLAGS__/, mvars[mconf, "cflags"], $0)
				# write the result down in the current fragment
				if ($0 != "") {
					$0 = trim($0)
					printf("%s\n", $0) >> output
				}
			}
			close(tmpl)
		}
	}
	split(aobj, objs, " ")
	for (o in objs) {
		# prepare the source file name `s' out of `o'
		s = objs[o]
		gsub(/\.o$/, ".S", s)
		gsub(/^\$\(BUILDDIR\)\//, "", s)

		# fix up the target template when building generated source files
		if (match(s, /^\$\(BUILDDIR\)\//) == 1) {
			tmpl = envar["tmpldir"] "/" "suffix_bldir.tmpl"
			n = split(s, gen, "/")
			while (getline < tmpl > 0) {
				gsub(/__OBJ__/, objs[o], $0)
				gsub(/__SRC__/, s, $0)
				gsub(/__GENSRC__/, "[GEN] " gen[n], $0)
				gsub(/__CFLAGS__/, mvars[mconf, "cflags"], $0)
				# write the result down in the current fragment
				if ($0 != "") {
					$0 = trim($0)
					printf("%s\n", $0) >> output
				}
			}
			close(tmpl)
		} else {
			tmpl = envar["tmpldir"] "/" "suffix.tmpl"
			while (getline < tmpl > 0) {
				gsub(/__OBJ__/, objs[o], $0)
				gsub(/__SRC__/, s, $0)
				gsub(/__CFLAGS__/, mvars[mconf, "cflags"], $0)
				# write the result down in the current fragment
				if ($0 != "") {
					$0 = trim($0)
					printf("%s\n", $0) >> output
				}
			}
			close(tmpl)
		}
	}
	split(mvars[mconf, "dobj"], objs, " ")
	for (o in objs) {
		# prepare the source file name `s' out of `o'
		s = objs[o]
		gsub(/\.bin$/, "", s)
		gsub(/^\$\(BUILDDIR\)\//, "", s)

		tmpl = envar["tmpldir"] "/" "suffix_data.tmpl"
		n = split(s, gen, "/")
		while (getline < tmpl > 0) {
			gsub(/__OBJ__/, objs[o], $0)
			gsub(/__SRC__/, s, $0)
			# write the result down in the current fragment
			if ($0 != "") {
				$0 = trim($0)
				printf("%s\n", $0) >> output
			}
		}
		close(tmpl)
	}

	print "" >> output
}

# output a "prog" target
function put_prog(mconf, mconfdir, output)
{
	tmpl = envar["tmpldir"] "/" "prog.tmpl"
	prefix = ""

	printf("# link the program `%s'\n", mvars[mconf, "target"]) >> output
	while (getline < tmpl > 0) {
		gsub(/__TARGET__/, mvars[mconf, "target"], $0)
		gsub(/__PATH__/, mconfdir, $0)
		gsub(/__NAME__/, mvars[mconf, "name"], $0)
		gsub(/__DEPEND_T__/, mvars[mconf, "deps_target"], $0)
		gsub(/__DEPEND_L__/, mvars[mconf, "deps_link"], $0)
		gsub(/__LDFLAGS__/, mvars[mconf, "ldflags"], $0)
		gsub(/__EXTRALIBS__/, mvars[mconf, "extralibs"], $0)
		$0 = trim($0)
		printf("%s\n", $0) >> output
	}
	close(tmpl)
	print "" >> output

	mvars[mconf, "cleanfiles"] = mvars[mconf, "cleanfiles"] " " "$(" mvars[mconf, "target"] ")"
}

# output a "lib" target
function put_lib(mconf, mconfdir, output)
{
	tmpl = envar["tmpldir"] "/" "lib.tmpl"

	printf("# create lib `%s'\n", mvars[mconf, "target"]) >> output
	while (getline < tmpl > 0) {
		gsub(/__TARGET__/, mvars[mconf, "target"], $0)
		gsub(/__PATH__/, mconfdir, $0)
		gsub(/__NAME__/, mvars[mconf, "name"], $0)
		$0 = trim($0)
		printf("%s\n", $0) >> output
	}
	close(tmpl)
	print "" >> output

	mvars[mconf, "cleanfiles"] = mvars[mconf, "cleanfiles"] " " "$(" mvars[mconf, "target"] ")"
}

# output a "data" target
function put_data(mconf, mconfdir, output)
{
	tmpl = envar["tmpldir"] "/" "data.tmpl"

	printf("# create embedded data object `%s'\n", mvars[mconf, "target"]) >> output
	while (getline < tmpl > 0) {
		gsub(/__TARGET__/, mvars[mconf, "target"], $0)
		gsub(/__PATH__/, mconfdir, $0)
		gsub(/__NAME__/, mvars[mconf, "name"], $0)
		$0 = trim($0)
		printf("%s\n", $0) >> output
	}
	close(tmpl)
	print "" >> output

	mvars[mconf, "cleanfiles"] = mvars[mconf, "cleanfiles"] " " "$(" mvars[mconf, "target"] ")"
}

# generate 1 .mk file for specified .mconf -- to be inlined in top Makefile
function put_mkfile(mconf, mconfdir, output)
{
	# gather objects from C files
	cob = mvars[mconf, "cobj"]
	if (envar["host"] == "unix")
		cob = cob " " mvars[mconf, "unix_cobj"]
	if (envar["host"] == "windows")
		cob = cob " " mvars[mconf, "win_cobj"]

	# gather objects from assembly .S files
	aob = mvars[mconf, "aobj"]
	if (envar["host"] == "unix")
		aob = aob " " mvars[mconf, "unix_aobj"]
	if (envar["host"] == "windows")
		aob = aob " " mvars[mconf, "win_aobj"]

	# write list of objects to build
	put_objlist(mconf, cob, aob, output)

	# if the mconf module is a program, write link rules
	if (mvars[mconf, "prog"] != "")
		put_prog(mconf, mconfdir, output)

	# if the mconf module is a library, write lib creation rules
	if (mvars[mconf, "lib"] != "")
		put_lib(mconf, mconfdir, output)

	# if the mconf module is embedded data objects, write data object creation rules
	if (mvars[mconf, "data"] != "")
		put_data(mconf, mconfdir, output)

	# write each object suffix build rules
	put_suffix_rules(cob, aob, mconf, output)
}

# generate the all: rule starting with loose targets, then libs, then programs in that order
# then generate the CLEANFILES rule
function genallrule(output)
{
	combined_mconfsfile = envar["makeitgendir"] "/combined-mconfsready.mk"
	reset_file(combined_mconfsfile)

	printf("all:") > output
	# write targets that are NOT libraries of programs first
	for (m in mconfs) {
		if (mvars[mconfs[m], "prog"] == "" && mvars[mconfs[m], "lib"] == "") {
			printf(" $(%s)", mvars[mconfs[m], "target"]) >> output
			put_mkfile(mconfs[m], mconfsdirs[m], combined_mconfsfile)
		}
	}
	# then libraries
	for (m in mconfs) {
		if (mvars[mconfs[m], "lib"] != "") {
			printf(" $(%s)", mvars[mconfs[m], "target"]) >> output
			put_mkfile(mconfs[m], mconfsdirs[m], combined_mconfsfile)
		}
	}
	# finally programs
	for (m in mconfs) {
		if (mvars[mconfs[m], "prog"] != "") {
			printf(" $(%s)", mvars[mconfs[m], "target"]) >> output
			put_mkfile(mconfs[m], mconfsdirs[m], combined_mconfsfile)
		}
	}

	# collect cleanfiles from all mconfs and set a CLEANFILES make var
	for (m in mconfs) {
		cl = cl " " mvars[mconfs[m], "cleanfiles"] " "
	}
	cl = trim(cl)
	printf("\n\nCLEANFILES += %s\n", cl) >> output
}

# check the parameters passed to the program
function checkvars()
{
	if (envar["mconflist"] == "")
		usage()
	if (envar["host"] == "")
		usage()
	if (envar["makeitgendir"] == "")
		usage()
	if (envar["tmpldir"] == "")
		usage()
}

# main entry
BEGIN {
	# variable defs
	mconfs[0] = ""
	mconfsdirs[0] = ""
	nmconfs = 0
	mvars[0] = ""
	words[0] = ""
	currkeywd = ""
	envar[0] = ""
	keywords[0] = ""
	klist = "name prog lib asrc data csrc win_asrc win_csrc unix_asrc unix_csrc"
	klist = klist " " "depends cflags ldflags extralibs cleanfiles"

	# init keywords
	split(klist, keywords, " ")

	# collect program environment vars from command line ARGV array
	for (i = 0; i < ARGC; i++) {
		n = split(ARGV[i], args, "=")
		if (n == 2)
			envar[args[1]] = args[2]
	}

	# check that we were called with all needed environment vars
	checkvars()

	# extract mconf dirname and basename from mconfig generated module.lst
	while (getline < envar["mconflist"] > 0) {
		mconfs[nmconfs] = $1 "/" $2
		mconfsdirs[nmconfs] = $1
		nmconfs++
	}

	# for all .mconf files found, 1) parse, 2) gen src/obj lists, 3) gen make targets
	for (i = 0; i < nmconfs; i++) {
		scanmconf(mconfs[i], envar["makeitgendir"])
		genobjs(mconfs[i])
		gentarget(mconfs[i])
	}

	# for each make targets generated above, generate dependency rules
	for (i = 0; i < nmconfs; i++) {
		gendeps(i)
	}

	# finally, generate the "all:" rule listing all target in dependency order
	genallrule(envar["makeitgendir"] "/" "all.mk")

#	printmvars()
}
