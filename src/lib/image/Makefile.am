SUBDIRS = attach bind create check expand offset
#SUBDIRS = attach create check mount bind expand offset

MAINTAINERCLEANFILES = Makefile.in config.h config.h.in
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ *.la
AM_CFLAGS = -Wall -fpie -fPIC
AM_LDFLAGS = -pie
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)
distlibdir = $(libdir)/singularity
distincludedir = $(includedir)/singularity

noinst_LTLIBRARIES = libinternal.la
libinternal_la_LIBADD = attach/libinternal.la bind/libinternal.la create/libinternal.la check/libinternal.la expand/libinternal.la offset/libinternal.la
libinternal_la_SOURCES = image.c
libinternal_la_CFLAGS = $(AM_CFLAGS) # This fixes duplicate sources in library and progs

distinclude_HEADERS = image.h
distlib_LTLIBRARIES = libsingularity-image.la

libsingularity_image_la_SOURCES =
libsingularity_image_la_LIBADD = $(noinst_LTLIBRARIES)
libsingularity_image_la_LDFLAGS = -version-info 1:0:0
libsingularity_image_la_CFLAGS = $(AM_CFLAGS)

