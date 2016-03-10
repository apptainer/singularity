MAINTAINERCLEANFILES = Makefile.in config.h config.h.in
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ 
AM_CFLAGS = -Wall
AM_CPPFLAGS = -DLIBEXECDIR=\"$(libexecdir)\" $(NAMESPACE_DEFINES)

bindir = $(libexecdir)/singularity
bin_PROGRAMS = sexec ftrace

sexec_SOURCES = sexec.c util.c util.h

EXTRA_DIST = config.h 
