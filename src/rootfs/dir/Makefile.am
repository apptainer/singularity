MAINTAINERCLEANFILES = Makefile.in 
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~

AM_CFLAGS = -Wall -fpie
AM_LDFLAGS = -pie
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)

noinst_LTLIBRARIES = librootfsdir.la
librootfsdir_la_SOURCES = dir.c

EXTRA_DIST = dir.h
