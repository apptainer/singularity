SUBDIRS = binds cwd dev home hostfs kernelfs scratch tmp userbinds

MAINTAINERCLEANFILES = Makefile.in
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ *.la

AM_CFLAGS = -Wall -fpie
AM_LDFLAGS = -pie
libns_pid_a_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES)
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES)

noinst_LTLIBRARIES = libinternal.la

libinternal_la_LIBADD = binds/libinternal.la cwd/libinternal.la dev/libinternal.la home/libinternal.la hostfs/libinternal.la kernelfs/libinternal.la scratch/libinternal.la tmp/libinternal.la userbinds/libinternal.la
libinternal_la_SOURCES = mounts.c

EXTRA_DIST = mounts.h
