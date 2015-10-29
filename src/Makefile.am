MAINTAINERCLEANFILES = Makefile.in config.h config.h.in
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ 
AM_CFLAGS = -Wall

bindir = $(libexecdir)/singularity
bin_PROGRAMS = sexec

singularity_SOURCES = sexec.c
singularity_LDADD = $(INTI_LIBS)

EXTRA_DIST = config.h 

setuid: sexec
	chown root:root sexec
	chmod +s sexec
