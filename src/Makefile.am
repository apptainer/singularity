SUBDIRS = lib action-lib bootstrap-lib util slurm

MAINTAINERCLEANFILES = Makefile.in config.h config.h.in
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ *.la
AM_CFLAGS = -Wall -fpie -fPIC
AM_LDFLAGS = -pie
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)

bindir = $(libexecdir)/singularity/bin

bin_PROGRAMS = action bootstrap copy cleanupd create expand export get-section import mount $(BUILD_SUID)
EXTRA_PROGRAMS = action-suid create-suid expand-suid export-suid import-suid mount-suid

cleanupd_SOURCES = cleanupd.c util/util.c util/file.c util/message.c util/privilege.c util/config_parser.c util/registry.c
cleanupd_CPPFLAGS = $(AM_CPPFLAGS)


action_SOURCES = action.c util/util.c util/file.c util/registry.c util/privilege.c util/sessiondir.c util/suid.c util/cleanupd.c
action_LDADD = lib/image/libsingularity-image.la lib/runtime/libsingularity-runtime.la action-lib/libinternal.la
action_CPPFLAGS = $(AM_CPPFLAGS)

bootstrap_SOURCES = bootstrap.c util/util.c util/file.c util/registry.c util/sessiondir.c
bootstrap_LDADD = lib/image/libsingularity-image.la lib/runtime/libsingularity-runtime.la bootstrap-lib/libinternal.la
bootstrap_CPPFLAGS = $(AM_CPPFLAGS)
bootstrap_LDFLAGS = -static

create_SOURCES = create.c util/util.c util/file.c util/registry.c util/privilege.c util/suid.c util/fork.c
create_LDADD = lib/image/libsingularity-image.la lib/runtime/libsingularity-runtime.la bootstrap-lib/libinternal.la
create_CPPFLAGS = $(AM_CPPFLAGS)

expand_SOURCES = expand.c util/util.c util/file.c util/registry.c util/privilege.c util/suid.c util/fork.c
expand_LDADD = lib/image/libsingularity-image.la lib/runtime/libsingularity-runtime.la bootstrap-lib/libinternal.la
expand_CPPFLAGS = $(AM_CPPFLAGS)

export_SOURCES = export.c util/util.c util/file.c util/registry.c util/privilege.c util/suid.c util/fork.c util/sessiondir.c
export_LDADD = lib/image/libsingularity-image.la lib/runtime/libsingularity-runtime.la
export_CPPFLAGS = $(AM_CPPFLAGS)

copy_SOURCES = copy.c util/util.c util/file.c util/registry.c util/privilege.c util/suid.c util/fork.c util/sessiondir.c
copy_LDADD = lib/image/libsingularity-image.la lib/runtime/libsingularity-runtime.la
copy_CPPFLAGS = $(AM_CPPFLAGS)

import_SOURCES = import.c util/util.c util/file.c util/registry.c util/privilege.c util/suid.c util/fork.c util/sessiondir.c
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

import_suid_SOURCES = $(import_SOURCES)
import_suid_LDADD = $(import_LDADD)
import_suid_LDFLAGS = -static
import_suid_CPPFLAGS = -DSINGULARITY_SUID $(AM_CPPFLAGS)

create_suid_SOURCES = $(create_SOURCES)
create_suid_LDADD = $(create_LDADD)
create_suid_LDFLAGS = -static
create_suid_CPPFLAGS = -DSINGULARITY_SUID $(AM_CPPFLAGS)

expand_suid_SOURCES = $(expand_SOURCES)
expand_suid_LDADD = $(expand_LDADD)
expand_suid_LDFLAGS = -static
expand_suid_CPPFLAGS = -DSINGULARITY_SUID $(AM_CPPFLAGS)

export_suid_SOURCES = $(export_SOURCES)
export_suid_LDADD = $(export_LDADD)
export_suid_LDFLAGS = -static
export_suid_CPPFLAGS = -DSINGULARITY_SUID $(AM_CPPFLAGS)

mount_suid_SOURCES = $(mount_SOURCES)
mount_suid_LDADD = $(mount_LDADD)
mount_suid_LDFLAGS = -static
mount_suid_CPPFLAGS = -DSINGULARITY_SUID $(AM_CPPFLAGS)


install-exec-hook: make_suid

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
