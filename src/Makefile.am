SUBDIRS = lib action-lib bootstrap-lib util

MAINTAINERCLEANFILES = Makefile.in config.h config.h.in
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ *.la
AM_CFLAGS = -Wall -fpie -fPIC
AM_LDFLAGS = -pie
AM_CPPFLAGS = -DBINDIR=\"$(bindir)\" -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES)

lexecdir = $(libexecdir)/singularity/bin

lexec_PROGRAMS = action builddef cleanupd get-section image-type mount start sif wrapper $(BUILD_SUID)
EXTRA_PROGRAMS = wrapper-suid

cleanupd_SOURCES = cleanupd.c util/util.c util/file.c util/message.c util/privilege.c util/config_parser.c util/registry.c util/capability.c util/suid.c
cleanupd_CPPFLAGS = $(AM_CPPFLAGS)

action_SOURCES = action.c util/util.c util/file.c util/registry.c util/privilege.c util/sessiondir.c util/suid.c util/cleanupd.c util/daemon.c util/capability.c util/mount.c util/binary.c
action_LDADD = lib/image/libsingularity-image.la lib/runtime/libsingularity-runtime.la lib/sif/libsingularity-sif.la action-lib/libinternal.la -luuid -lelf
action_CPPFLAGS = $(AM_CPPFLAGS)

builddef_SOURCES = builddef.c util/util.c util/file.c util/registry.c util/sessiondir.c util/capability.c util/binary.c
builddef_LDADD = lib/image/libsingularity-image.la lib/runtime/libsingularity-runtime.la lib/sif/libsingularity-sif.la bootstrap-lib/libinternal.la -luuid -lelf
builddef_CPPFLAGS = $(AM_CPPFLAGS)
builddef_LDFLAGS = -static

start_SOURCES = start.c util/util.c util/file.c util/registry.c util/privilege.c util/sessiondir.c util/suid.c util/cleanupd.c util/fork.c util/daemon.c util/signal.c util/capability.c util/mount.c util/binary.c
start_LDADD = lib/image/libsingularity-image.la lib/runtime/libsingularity-runtime.la lib/sif/libsingularity-sif.la action-lib/libinternal.la -luuid -lelf
start_CPPFLAGS = $(AM_CPPFLAGS)

mount_SOURCES = mount.c util/util.c util/file.c util/registry.c util/suid.c util/privilege.c util/capability.c util/mount.c util/binary.c
mount_LDADD = lib/image/libsingularity-image.la lib/runtime/libsingularity-runtime.la lib/sif/libsingularity-sif.la -luuid -lelf
mount_CPPFLAGS = $(AM_CPPFLAGS)

sif_SOURCES = sif.c util/util.c util/binary.c
sif_LDADD = lib/image/libsingularity-image.la lib/signing/libsingularity-signing.la lib/sif/libsingularity-sif.la -luuid -lcrypto -lelf
sif_CPPFLAGS = $(AM_CPPFLAGS)

get_section_SOURCES = get-section.c util/util.c util/file.c util/message.c util/privilege.c util/config_parser.c util/registry.c util/capability.c util/suid.c
get_section_CPPFLAGS = $(AM_CPPFLAGS)

image_type_SOURCES = image-type.c util/util.c util/message.c util/config_parser.c util/file.c
image_type_LDADD = lib/image/libsingularity-image.la lib/sif/libsingularity-sif.la -luuid
image_type_CPPFLAGS = $(AM_CPPFLAGS)

wrapper_SOURCES = wrapper.c util/util.c util/file.c util/registry.c util/suid.c util/privilege.c util/capability.c util/binary.c
wrapper_LDADD = lib/image/libsingularity-image.la lib/runtime/libsingularity-runtime.la lib/sif/libsingularity-sif.la -luuid -lelf
wrapper_CPPFLAGS = $(AM_CPPFLAGS)

wrapper_suid_SOURCES = $(wrapper_SOURCES)
wrapper_suid_LDADD = $(wrapper_LDADD)
wrapper_suid_LDFLAGS = -static
wrapper_suid_CPPFLAGS = -DSINGULARITY_SUID $(AM_CPPFLAGS)

install-data-hook: make_suid

make_suid:
	@if test `id -ru` = "0"; then \
		for i in $(BUILD_SUID); do \
			echo " /bin/chown root:root $(DESTDIR)$(libexecdir)/singularity/bin/$$i"; \
			/bin/chown root:root $(DESTDIR)$(libexecdir)/singularity/bin/$$i ; \
			echo " /bin/chmod 4755 $(DESTDIR)$(libexecdir)/singularity/bin/$$i"; \
			/bin/chmod 4755 $(DESTDIR)$(libexecdir)/singularity/bin/$$i; \
		done; \
	fi


EXTRA_DIST = config.h
