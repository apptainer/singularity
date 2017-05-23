SUBDIRS = image dir squashfs

MAINTAINERCLEANFILES = Makefile.in 
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ *.la

AM_CFLAGS = -Wall -fpie
AM_LDFLAGS = -pie
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)

noinst_LTLIBRARIES = librootfs.la

librootfs_la_LIBADD = image/librootfsimage.la dir/librootfsdir.la squashfs/librootfssquashfs.la
librootfs_la_SOURCES = rootfs.c

EXTRA_DIST = rootfs.h
