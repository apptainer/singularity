SUBDIRS = mount create bind expand

MAINTAINERCLEANFILES = Makefile.in 
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ *.la

AM_CFLAGS = -Wall -fpie
AM_LDFLAGS = -pie
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)

noinst_LTLIBRARIES = libimage.la

libimage_la_LIBADD = mount/libimagemount.la create/libimagecreate.la bind/libimagebind.la expand/libimageexpand.la
libimage_la_SOURCES = image.c

EXTRA_DIST = image.h
