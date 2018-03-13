MAINTAINERCLEANFILES = Makefile.in 
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ *.la *.o

AM_CFLAGS = -Wall -fpie
AM_LDFLAGS = -pie
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES)

11_user_defined_SINGULARITYENV_PATH_SOURCES = 11_user_defined_SINGULARITYENV_PATH.c
11_user_defined_SINGULARITYENV_PATH_LDADD = bootstrap-lib/libinternal.la
11_user_defined_SINGULARITYENV_PATH_CPPFLAGS = $(AM_CPPFLAGS)

noinst_LTLIBRARIES = libinternal.la
libinternal_la_SOURCES = 11_user_defined_SINGULARITYENV_PATH.c

EXTRA_DIST = 11_user_defined_SINGULARITYENV_PATH.h
