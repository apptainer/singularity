MAINTAINERCLEANFILES = Makefile.in config.h config.h.in
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ 
AM_CFLAGS = -Wall
sexec_CPPFLAGS = -DLIBEXECDIR=\"$(libexecdir)\" $(NAMESPACE_DEFINES)
ftrace_CPPFLAGS = -DARCH_$(SINGULARITY_ARCH)

bindir = $(libexecdir)/singularity
bin_PROGRAMS = sexec ftrace ftype

sexec_SOURCES = sexec.c util.c util.h
ftrace_SOURCES = ftrace.c util.c util.h
ftype_SOURCES = ftype.c util.c util.h

EXTRA_DIST = config.h 
