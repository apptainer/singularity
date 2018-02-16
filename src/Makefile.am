SUBDIRS = lib action-lib bootstrap-lib util

MAINTAINERCLEANFILES = Makefile.in config.h config.h.in
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ *.la
AM_CFLAGS = -Wall -fpie -fPIC
AM_LDFLAGS = -pie
AM_CPPFLAGS = -DBINDIR=\"$(bindir)\" -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES)

lexecdir = $(libexecdir)/singularity/bin

lexec_PROGRAMS = action builddef cleanupd get-section image-type mount prepheader start $(BUILD_SUID)
EXTRA_PROGRAMS = action-suid mount-suid start-suid

cleanupd_SOURCES = cleanupd.c util/util.c util/file.c util/message.c util/privilege.c util/config_parser.c util/registry.c
cleanupd_CPPFLAGS = $(AM_CPPFLAGS)

action_SOURCES = action.c util/util.c util/file.c util/registry.c util/privilege.c util/sessiondir.c util/suid.c util/cleanupd.c util/daemon.c util/mount.c
action_LDADD = lib/image/libsingularity-image.la lib/runtime/libsingularity-runtime.la action-lib/libinternal.la
action_CPPFLAGS = $(AM_CPPFLAGS)

builddef_SOURCES = builddef.c util/util.c util/file.c util/registry.c util/sessiondir.c
builddef_LDADD = lib/image/libsingularity-image.la lib/runtime/libsingularity-runtime.la bootstrap-lib/libinternal.la
builddef_CPPFLAGS = $(AM_CPPFLAGS)
builddef_LDFLAGS = -static

start_SOURCES = start.c util/util.c util/file.c util/registry.c util/privilege.c util/sessiondir.c util/suid.c util/cleanupd.c util/fork.c util/daemon.c util/signal.c util/mount.c
start_LDADD = lib/image/libsingularity-image.la lib/runtime/libsingularity-runtime.la action-lib/libinternal.la
start_CPPFLAGS = $(AM_CPPFLAGS)

mount_SOURCES = mount.c util/util.c util/file.c util/registry.c util/suid.c util/privilege.c util/mount.c
mount_LDADD = lib/image/libsingularity-image.la lib/runtime/libsingularity-runtime.la
mount_CPPFLAGS = $(AM_CPPFLAGS)

prepheader_SOURCES = prepheader.c
prepheader_LDADD =
prepheader_CPPFLAGS = $(AM_CPPFLAGS)

get_section_SOURCES = get-section.c util/util.c util/file.c util/message.c util/privilege.c util/config_parser.c util/registry.c
get_section_CPPFLAGS = $(AM_CPPFLAGS)

image_type_SOURCES = image-type.c util/util.c util/message.c util/config_parser.c util/file.c
image_type_LDADD = lib/image/libsingularity-image.la
image_type_CPPFLAGS = $(AM_CPPFLAGS)

action_suid_SOURCES = $(action_SOURCES)
action_suid_LDADD = $(action_LDADD)
action_suid_LDFLAGS = -static
action_suid_CPPFLAGS = -DSINGULARITY_SUID $(AM_CPPFLAGS)

start_suid_SOURCES = $(start_SOURCES)
start_suid_LDADD = $(start_LDADD)
start_suid_LDFLAGS = -static
start_suid_CPPFLAGS = -DSINGULARITY_SUID $(AM_CPPFLAGS)

mount_suid_SOURCES = $(mount_SOURCES)
mount_suid_LDADD = $(mount_LDADD)
mount_suid_LDFLAGS = -static
mount_suid_CPPFLAGS = -DSINGULARITY_SUID $(AM_CPPFLAGS)


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
