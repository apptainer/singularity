MAINTAINERCLEANFILES = Makefile.in config.h config.h.in
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ 
AM_CFLAGS = -DLIBEXECPATH=\"$(libexecdir)/singularity\" -g


bindir = $(libexecdir)/singularity
bin_PROGRAMS = singularity

AM_CXXFLAGS = $(INTI_CFLAGS)

singularity_SOURCES = singularity.c
singularity_LDADD = $(INTI_LIBS)

EXTRA_DIST = config.h constants.h singularity.h


setuid: singularity
	chown root:root singularity
	chmod +s singularity
