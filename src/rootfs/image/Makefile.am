AM_CFLAGS = -Wall -fpie
AM_LDFLAGS = -pie
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)

noinst_LTLIBRARIES = librootfsimage.la
#libimage_util_a_SOURCES = image-util.c ../../message.c ../../config_parser.c
librootfsimage_la_SOURCES = image.c image-util.c loop-control.c ../../file.c ../../util.c ../../message.c ../../config_parser.c

EXTRA_DIST = loop-control.h image.h image-util.h
