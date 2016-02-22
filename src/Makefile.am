MAINTAINERCLEANFILES = Makefile.in config.h config.h.in
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ 
AM_CFLAGS = -Wall
AM_CPPFLAGS = -DLIBEXECDIR=\"$(libexecdir)\"

libexecbindir = $(libexecdir)/singularity
libexecbin_PROGRAMS = sexec ftrace sapplauncher sappbuild sappdump sappmeta

sexec_SOURCES = sexec.c util.c util.h
sappbuild_SOURCES = sappbuild.c sappheader.c sappheader.h
sappdump_SOURCES = sappdump.c sappheader.c sappheader.h
sappmeta_SOURCES = sappdump.c sappheader.c sappheader.h

EXTRA_DIST = config.h 
