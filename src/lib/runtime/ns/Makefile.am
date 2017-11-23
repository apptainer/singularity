SUBDIRS = mnt pid ipc net uts user

MAINTAINERCLEANFILES = Makefile.in 
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ *.la

AM_CFLAGS = -Wall -fpie
AM_LDFLAGS = -pie
libns_pid_a_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES)
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES)

noinst_LTLIBRARIES = libinternal.la

libinternal_la_LIBADD = mnt/libinternal.la pid/libinternal.la ipc/libinternal.la net/libinternal.la uts/libinternal.la user/libinternal.la
libinternal_la_SOURCES = ns.c ../../../util/daemon.c

EXTRA_DIST = ns.h
