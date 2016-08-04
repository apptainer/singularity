SUBDIRS = image dir

AM_CFLAGS = -Wall -fpie
AM_LDFLAGS = -pie
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)

noinst_LIBRARIES = librootfs.a

librootfs_a_LIBADD = image/librootfsimage.a dir/librootfsdir.a
librootfs_a_SOURCES = rootfs.c ../message.c ../config_parser.c

EXTRA_DIST = rootfs.h
