MAINTAINERCLEANFILES = Makefile.in 
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ *.la *.o

AM_CFLAGS = -Wall -fpie
AM_LDFLAGS = -pie
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES)

noinst_LTLIBRARIES = libinternal.la
libinternal_la_SOURCES = 11_udsep.c 

EXTRA_DIST = 11_udsep.c 
