AM_CFLAGS = -Wall -fpie
AM_LDFLAGS = -pie
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)

noinst_LIBRARIES = libimage-util.a libimage-mount.a
libimage_util_a_SOURCES = image-util.c ../message.c ../config_parser.c
libimage_mount_a_SOURCES = image-mount.c loop-control.c ../message.c ../config_parser.c

EXTRA_DIST = loop-control.h image-mount.h image-util.h
