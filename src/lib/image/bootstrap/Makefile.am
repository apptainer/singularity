SUBDIRS = docker yum debootstrap busybox arch

MAINTAINERCLEANFILES = Makefile.in 
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ *.la

AM_CFLAGS = -Wall -fpie
AM_LDFLAGS = -pie
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)

noinst_LTLIBRARIES = libimagebootstrap.la

libimagebootstrap_la_LIBADD = docker/libbootstrapdocker.la debootstrap/libbootstrapdebootstrap.la yum/libbootstrapyum.la busybox/libbootstrapbusybox.la arch/libbootstraparch.la
libimagebootstrap_la_SOURCES = bootstrap.c bootdef_parser.c

EXTRA_DIST = bootstrap.h bootdef_parser.h
