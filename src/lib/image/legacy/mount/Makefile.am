SUBDIRS = dir image squashfs

MAINTAINERCLEANFILES = Makefile.in 
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ *.la

AM_CFLAGS = -Wall -fpie
AM_LDFLAGS = -pie
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)

libinternal_la_LIBADD = dir/libinternal.la image/libinternal.la squashfs/libinternal.la
noinst_LTLIBRARIES = libinternal.la
libinternal_la_SOURCES = mount.c

EXTRA_DIST = mount.h
