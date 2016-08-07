MAINTAINERCLEANFILES = Makefile.in 
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~

AM_CFLAGS = -Wall -fpie
AM_LDFLAGS = -pie
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)

noinst_LTLIBRARIES = libactionexec.la
libactionrun_la_SOURCES = exec.c

EXTRA_DIST = exec.h
