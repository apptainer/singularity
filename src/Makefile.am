SUBDIRS = lib action-lib bootstrap-lib

MAINTAINERCLEANFILES = Makefile.in config.h config.h.in
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ *.la
AM_CFLAGS = -Wall -fpie -fPIC
AM_LDFLAGS = -pie
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)

bindir = $(libexecdir)/singularity

bin_PROGRAMS = action bootstrap create expand get-section import mount $(BUILD_SUID)
EXTRA_PROGRAMS = action-suid create-suid expand-suid import-suid mount-suid

action_SOURCES = action.c util/util.c util/file.c util/registry.c util/privilege.c util/suid.c
action_LDADD = lib/image/libsingularity-image.la lib/runtime/libsingularity-runtime.la action-lib/libinternal.la
action_CPPFLAGS = $(AM_CPPFLAGS)

bootstrap_SOURCES = bootstrap.c util/util.c util/file.c util/registry.c
bootstrap_LDADD = lib/image/libsingularity-image.la lib/runtime/libsingularity-runtime.la bootstrap-lib/libinternal.la
bootstrap_CPPFLAGS = $(AM_CPPFLAGS)

create_SOURCES = create.c util/util.c util/file.c util/registry.c util/privilege.c util/suid.c util/fork.c
create_LDADD = lib/image/libsingularity-image.la lib/runtime/libsingularity-runtime.la bootstrap-lib/libinternal.la
create_CPPFLAGS = $(AM_CPPFLAGS)

expand_SOURCES = expand.c util/util.c util/file.c util/registry.c util/privilege.c util/suid.c util/fork.c
expand_LDADD = lib/image/libsingularity-image.la lib/runtime/libsingularity-runtime.la bootstrap-lib/libinternal.la
expand_CPPFLAGS = $(AM_CPPFLAGS)

import_SOURCES = import.c util/util.c util/file.c util/registry.c util/privilege.c util/suid.c util/fork.c
import_LDADD = lib/image/libsingularity-image.la lib/runtime/libsingularity-runtime.la
import_CPPFLAGS = $(AM_CPPFLAGS)

mount_SOURCES = mount.c util/util.c util/file.c util/registry.c util/suid.c util/privilege.c
mount_LDADD = lib/image/libsingularity-image.la lib/runtime/libsingularity-runtime.la
mount_CPPFLAGS = $(AM_CPPFLAGS)

get_section_SOURCES = get-section.c util/util.c util/file.c util/message.c util/privilege.c util/config_parser.c util/registry.c
get_section_CPPFLAGS = $(AM_CPPFLAGS)


action_suid_SOURCES = $(action_SOURCES)
action_suid_LDADD = $(action_LDADD)
action_suid_LDFLAGS = -static
action_suid_CPPFLAGS = -DSINGULARITY_SUID $(AM_CPPFLAGS)

create_suid_SOURCES = $(create_SOURCES)
create_suid_LDADD = $(create_LDADD)
create_suid_LDFLAGS = -static
create_suid_CPPFLAGS = -DSINGULARITY_SUID $(AM_CPPFLAGS)

expand_suid_SOURCES = $(expand_SOURCES)
expand_suid_LDADD = $(expand_LDADD)
expand_suid_LDFLAGS = -static
expand_suid_CPPFLAGS = -DSINGULARITY_SUID $(AM_CPPFLAGS)

import_suid_SOURCES = $(import_SOURCES)
import_suid_LDADD = $(import_LDADD)
import_suid_LDFLAGS = -static
import_suid_CPPFLAGS = -DSINGULARITY_SUID $(AM_CPPFLAGS)

mount_suid_SOURCES = $(mount_SOURCES)
mount_suid_LDADD = $(mount_LDADD)
mount_suid_LDFLAGS = -static
mount_suid_CPPFLAGS = -DSINGULARITY_SUID $(AM_CPPFLAGS)



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
##sexec_nosuid_CPPFLAGS = -DSINGULARITY_NOSUID=1 -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)
##bootstrap_CPPFLAGS = -DLIBEXECDIR=\"$(libexecdir)\"
##ftrace_CPPFLAGS = -DARCH_$(SINGULARITY_ARCH)
#
##dist_suidPROGRAM_INSTALL = ${INSTALL} -m 640
#
install-exec-hook: make_suid

make_suid:
	@if test `id -ru` = "0"; then \
		for i in $(BUILD_SUID); do \
			echo " /bin/chown root:root $(DESTDIR)$(libexecdir)/singularity/$$i"; \
			/bin/chown root:root $(DESTDIR)$(libexecdir)/singularity/$$i ; \
			echo " /bin/chmod 4755 $(DESTDIR)$(libexecdir)/singularity/$$i"; \
			/bin/chmod 4755 $(DESTDIR)$(libexecdir)/singularity/$$i; \
		done; \
	fi


EXTRA_DIST = config.h
