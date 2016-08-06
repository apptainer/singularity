SUBDIRS = ns rootfs

MAINTAINERCLEANFILES = Makefile.in config.h config.h.in
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ 
AM_CFLAGS = -Wall -fpie -shared -fPIC
AM_LDFLAGS = -pie


lib_LTLIBRARIES = libsingularity.la

libsingularity_la_LIBADD = ns/libns.la rootfs/librootfs.la
libsingularity_la_SOURCES = singularity.c message.c util.c file.c privilege.c
libsingularity_la_LDFLAGS = -version-info 1:0:0


bindir = $(libexecdir)/singularity

bin_PROGRAMS = sexec-devel

sexec_devel_SOURCES = sexec-devel.c sessiondir.c
sexec_devel_LDADD = libsingularity.la




#sexec_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)
#sexec_nosuid_CPPFLAGS = -DSINGULARITY_NOSUID=1 -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)
#bootstrap_CPPFLAGS = -DLIBEXECDIR=\"$(libexecdir)\"
#ftrace_CPPFLAGS = -DARCH_$(SINGULARITY_ARCH)

#dist_suidPROGRAM_INSTALL = ${INSTALL} -m 640

install-data-hook: make_suid

make_suid:
	@if test `id -ru` = "0"; then \
		echo " /bin/chown root:root $(DESTDIR)$(libexecdir)/singularity/sexec-devel"; \
		/bin/chown root:root $(DESTDIR)$(libexecdir)/singularity/sexec-devel; \
		echo " /bin/chmod 4755 $(DESTDIR)$(libexecdir)/singularity/sexec-devel"; \
		/bin/chmod 4755 $(DESTDIR)$(libexecdir)/singularity/sexec-devel; \
	fi

#bin_PROGRAMS = sexec image-create image-expand image-mount image-bind sexec_nosuid
#
#ftrace_SOURCES = ftrace.c file.c util.c message.c
#ftype_SOURCES = ftype.c util.c file.c message.c
#sexec_SOURCES = sexec.c util.c loop-control.c mounts.c container_files.c file.c image.c config_parser.c container_actions.c privilege.c message.c namespaces.c
#sexec_nosuid_SOURCES = sexec.c util.c loop-control.c mounts.c container_files.c file.c image.c config_parser.c container_actions.c privilege.c message.c namespaces.c
#image_create_SOURCES = image-create.c file.c util.c image.c message.c privilege.c
#image_expand_SOURCES = image-expand.c file.c util.c image.c message.c privilege.c
#image_mount_SOURCES = image-mount.c util.c loop-control.c mounts.c file.c image.c message.c config_parser.c privilege.c
#image_bind_SOURCES = image-bind.c util.c loop-control.c mounts.c file.c image.c message.c config_parser.c privilege.c

EXTRA_DIST = config.h config_parser.h container_actions.h file.h singularity.h
#image.h loop-control.h mounts.h container_files.h util.h privilege.h message.h namespaces.h

ACLOCAL_AMFLAGS = -I $(top_srcdir)/src/ -I .
