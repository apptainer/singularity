SUBDIRS = lib action-lib bootstrap-lib import-lib
#SUBDIRS = lib util slurm

MAINTAINERCLEANFILES = Makefile.in config.h config.h.in
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ *.la
AM_CFLAGS = -Wall -fpie -fPIC
AM_LDFLAGS = -pie
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)

bindir = $(libexecdir)/singularity

bin_PROGRAMS = action action-suid bootstrap import import-suid mount
EXTRA_PROGRAMS = action-suid import-suid

action_SOURCES = action.c util/util.c util/file.c lib/registry.c lib/privilege.c
action_LDADD = lib/image/libsingularity-image.la lib/runtime/libsingularity-runtime.la action-lib/libinternal.la

bootstrap_SOURCES = bootstrap.c util/util.c util/file.c lib/registry.c
bootstrap_LDADD = lib/image/libsingularity-image.la lib/runtime/libsingularity-runtime.la bootstrap-lib/libinternal.la

import_SOURCES = import.c util/util.c util/file.c lib/registry.c
import_LDADD = lib/image/libsingularity-image.la lib/runtime/libsingularity-runtime.la import-lib/libinternal.la

mount_SOURCES = mount.c util/util.c util/file.c lib/registry.c
mount_LDADD = lib/image/libsingularity-image.la


action_suid_SOURCES = $(action_SOURCES)
action_suid_LDADD = $(action_LDADD)
action_suid_LDFLAGS = -static
action_suid_CPPFLAGS = -DSINGULARITY_SUID $(AM_CPPFLAGS)

import_suid_SOURCES = $(import_SOURCES)
import_suid_LDADD = $(import_LDADD)
import_suid_LDFLAGS = -static
import_suid_CPPFLAGS = -DSINGULARITY_SUID $(AM_CPPFLAGS)



#bin_PROGRAMS = $(BUILD_SUID) sexec simage get-section
#EXTRA_PROGRAMS = sexec-suid
#
#sexec_SOURCES = sexec.c util/util.c util/file.c
#sexec_LDADD = lib/libsingularity.la
#
#sexec_suid_SOURCES = sexec.c util/util.c util/file.c
#sexec_suid_LDADD = lib/libsingularity.la
#sexec_suid_LDFLAGS = -static
#sexec_suid_CPPFLAGS = -DSINGULARITY_SUID $(AM_CPPFLAGS)
#
#simage_SOURCES = simage.c util/util.c util/file.c
#simage_LDADD = lib/libsingularity.la
#
##image_create_SOURCES = image-create.c util/file.c util/util.c
##image_create_LDADD = lib/libsingularity.la
#
##image_expand_SOURCES = image-expand.c util/file.c util/util.c
##image_expand_LDADD = lib/libsingularity.la
#
##image_mount_SOURCES = image-mount.c util/util.c util/file.c
##image_mount_LDADD = lib/libsingularity.la
#
##image_bind_SOURCES = image-bind.c util/util.c util/file.c
##image_bind_LDADD = lib/libsingularity.la
#
#get_section_SOURCES = get-section.c util/util.c util/file.c
#get_section_LDADD = lib/libsingularity.la
#
##sexec_nosuid_CPPFLAGS = -DSINGULARITY_NOSUID=1 -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)
##bootstrap_CPPFLAGS = -DLIBEXECDIR=\"$(libexecdir)\"
##ftrace_CPPFLAGS = -DARCH_$(SINGULARITY_ARCH)
#
##dist_suidPROGRAM_INSTALL = ${INSTALL} -m 640
#
#install-exec-hook: make_suid
#
#make_suid:
#	@if test `id -ru` = "0" -a "x$(BUILD_SUID)" = "xsexec-suid"; then \
#		echo " /bin/chown root:root $(DESTDIR)$(libexecdir)/singularity/sexec-suid"; \
#		/bin/chown root:root $(DESTDIR)$(libexecdir)/singularity/sexec-suid; \
#		echo " /bin/chmod 4755 $(DESTDIR)$(libexecdir)/singularity/sexec-suid"; \
#		/bin/chmod 4755 $(DESTDIR)$(libexecdir)/singularity/sexec-suid; \
#	fi
#
EXTRA_DIST = config.h
