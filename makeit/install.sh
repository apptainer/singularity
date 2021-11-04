#!/bin/sh -
# Copyright (c) 2015-2018, Yannick Cote <yhcote@gmail.com>. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be found
# in the LICENSE file.
set -e

name=
rootdir=

usage () {
	echo "Usage: ${0##*/} PROJECT_NAME PROJECT_ROOTDIR"
	echo
	echo "  Where :"
	echo "    PROJECT_NAME: name of source code project"
	echo "    PROJECT_ROOTDIR: the path to the root directory of the source code"
	echo
	exit 2
}

help () {
	echo
	echo "  Getting Start With Makeit"
	echo " ==========================="
	echo
	echo "  ${0##*/} replaces the mconfig \`package_name' var with PROJECT_NAME the first"
	echo "  option to the command and sets up a makeit tree inside the root directory of"
	echo "  the source code project designated by PROJECT_ROOTDIR the second option passed"
	echo "  to the command."
	echo
	echo "  At this point, to use makeit to bootstrap a \`make' (non-recursive Makefile)"
	echo "	build system:"
	echo
	echo "    * Edit to remove or augment:"
	echo "      - mlocal/frags/{release_opts.mk, Makefile.stub, common_opts.mk, etc.}"
	echo "      - mlocal/checks/{project-pre.chk, project-post.chk}"
	echo "    * Create {prog,lib}.mconf files for each elements to build"
	echo "    * Run ./mconfig from the project rootdir"
	echo "    * Build the project with make"
	echo
	echo "  Example:"
	echo "  1) ./install.sh myproject /home/yhcote/projects/myproject"
	echo "  2) cd /home/yhcote/projects/myproject"
	echo "  3) Adjust CFLAGS, CPPFLAGS, LDFLAGS, etc., from files in mlocal/frags/*"
	echo "  4) write project specific configure checks in project-pre.chk (to be run"
	echo "     before basechecks.chk) or in project-post.chk (to be run after"
	echo "     basechecks.chk)."
	echo "  5) create an mconf file (myprog.mconf) for \`myprog' program to build:"
	echo "  -----------------------------------------------------------------------"
	echo "  name := myprog"
	echo "  prog := myprog"
	echo "  csrc := src/file1.c src/file2.c src/file3.c"
	echo "  -----------------------------------------------------------------------"
	echo "  6) ./mconfig"
	echo "  7) cd builddir && make"
	echo
}

if [ $# != 2 ]; then
	usage
fi

name=$1
if ! rootdir=`(cd $2 2>/dev/null && pwd -P)`; then
	echo "error: $2 non-existent or permission denied"
	exit 2
fi

echo "  => INSTALL: makeit for project $name -> $rootdir ..."

install -d $rootdir/makeit
install -d $rootdir/makeit/tmpl
install -d $rootdir/mlocal/checks
install -d $rootdir/mlocal/frags
install -d $rootdir/mlocal/scripts
install -m 0644 mlocal/frags/* $rootdir/mlocal/frags
install -m 0644 mlocal/checks/* $rootdir/mlocal/checks
install -m 0644 mlocal/scripts/* $rootdir/mlocal/scripts
install -m 0644 tmpl/* $rootdir/makeit/tmpl
install -m 0644 CONTRIBUTORS INSTALL.md LICENSE README.md $rootdir/makeit
install -m 0755 genmod.awk install.sh $rootdir/makeit
install -m 0755 mconfig $rootdir

echo "  => NAME: setting variable \`package_name' to $name in $rootdir/mconfig..."
sed -i "s/PROJECT_NAME/$name/g" $rootdir/mconfig

echo "  => DONE: mconfig, mlocal/* and makeit/* added to project structure!"

help
