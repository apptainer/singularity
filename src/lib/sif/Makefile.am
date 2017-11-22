MAINTAINERCLEANFILES = Makefile.in config.h config.h.in
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ *.la
AM_CFLAGS = -Wall -fpie -fPIC
AM_LDFLAGS = -pie
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)
distlibdir = $(libdir)/singularity
distincludedir = $(includedir)/singularity

noinst_LTLIBRARIES = libsif.la
libsif_la_SOURCES = list.c sif.c sifaccess.c
libsif_la_CFLAGS = $(AM_CFLAGS) # This fixes duplicate sources in library and progs

distinclude_HEADERS = list.h sif.h sifaccess.h
distlib_LTLIBRARIES = libsingularity-sif.la

libsingularity_sif_la_SOURCES =
libsingularity_sif_la_LIBADD = $(noinst_LTLIBRARIES)
libsingularity_sif_la_LDFLAGS = -version-info 1:0:0
libsingularity_sif_la_CFLAGS = $(AM_CFLAGS)
