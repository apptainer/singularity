MAINTAINERCLEANFILES = Makefile.in config.h config.h.in
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ 
AM_CFLAGS = -Wall
AM_CPPFLAGS = -DLIBEXECDIR=\"$(libexecdir)\"

libexecbindir = $(libexecdir)/singularity
libexecbin_PROGRAMS = sexec ftrace sapplauncher

sexec_SOURCES = sexec.c util.c util.h

EXTRA_DIST = config.h 
