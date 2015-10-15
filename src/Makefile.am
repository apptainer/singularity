MAINTAINERCLEANFILES = Makefile.in aclocal.m4 configure compile config.* ltmain.sh depcomp install-sh missing stamp-*
DISTCLEANFILES =
CLEANFILES = Makefile core.* *~ 


bin_PROGRAMS = singularity

AM_CXXFLAGS = $(INTI_CFLAGS)

singularity_SOURCES = singularity.c
singularity_LDADD = $(INTI_LIBS)

EXTRA_DIST = 


setuid:
	chown root:root singularity
	chmod +s singularity
