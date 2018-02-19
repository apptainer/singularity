SUBDIRS = ext3 dir squashfs sif

MAINTAINERCLEANFILES = Makefile.in config.h config.h.in
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ *.la
AM_CFLAGS = -Wall -fpie -fPIC
AM_LDFLAGS = -pie
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES)
distlibdir = $(libdir)/singularity
distincludedir = $(includedir)/singularity

noinst_LTLIBRARIES = libimage.la
libimage_la_LIBADD = ext3/libinternal.la dir/libinternal.la squashfs/libinternal.la sif/libinternal.la
libimage_la_SOURCES = image.c bind.c ../../util/registry.c ../../util/message.c ../../util/config_parser.c ../../util/privilege.c ../../util/util.c ../../util/file.c ../../util/suid.c ../../util/capability.c ../../util/mount.c
libimage_la_CFLAGS = $(AM_CFLAGS) # This fixes duplicate sources in library and progs

distinclude_HEADERS = image.h bind.h
distlib_LTLIBRARIES = libsingularity-image.la

libsingularity_image_la_SOURCES =
libsingularity_image_la_LIBADD = $(noinst_LTLIBRARIES)
libsingularity_image_la_LDFLAGS = -version-info 1:0:0
libsingularity_image_la_CFLAGS = $(AM_CFLAGS)


# These are kludges so they don't remove the $(DEPDIR) in ../../util/ otherwise
# the clean will fail when other Makefile tries to remove those directories
distclean: distclean-recursive
	-rm ./$(DEPDIR)
	-rm -f Makefile

maintainer-clean: maintainer-clean-recursive
	-rm ./$(DEPDIR)
	-rm -f Makefile

