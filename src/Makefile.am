MAINTAINERCLEANFILES = Makefile.in config.h config.h.in
DISTCLEANFILES =
CLEANFILES = Makefile core.* *~ 


bin_PROGRAMS = singularity

AM_CXXFLAGS = $(INTI_CFLAGS)

singularity_SOURCES = singularity.c
singularity_LDADD = $(INTI_LIBS)

EXTRA_DIST = config.h constants.h


setuid: singularity
	chown root:root singularity
	chmod +s singularity
