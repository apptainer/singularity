SUBDIRS = binds cwd dev home hostfs kernelfs scratch tmp userbinds

MAINTAINERCLEANFILES = Makefile.in
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ *.la

AM_CFLAGS = -Wall -fpie
AM_LDFLAGS = -pie
libns_pid_a_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)

noinst_LTLIBRARIES = libinternal.la

libinternal_la_LIBADD = kernelfs/libinternal.la binds/libinternal.la home/libinternal.la tmp/libinternal.la hostfs/libinternal.la userbinds/libinternal.la dev/libinternal.la cwd/libinternal.la scratch/libinternal.la
libinternal_la_SOURCES = mount.c mount-util.c

EXTRA_DIST = mount-util.c mount-util.h
