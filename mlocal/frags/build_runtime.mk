# This file contains all of the rules for building the singularity runtime
#   and installing the necessary config files.

# contain starter_SOURCE variable list
starter_deps := $(BUILDDIR_ABSPATH)/starter.d

-include $(starter_deps)

$(starter_deps): $(GO_MODFILES)
	@echo " GEN GO DEP" $@
	$(V)$(SOURCEDIR)/makeit/gengodep -v3 "$(GO)" "starter_SOURCE" "$(GO_TAGS)" "$@" "$(SOURCEDIR)/cmd/starter"

starter_CSOURCE := $(wildcard $(SOURCEDIR)/cmd/starter/c/*.c)
starter_CSOURCE += $(wildcard $(SOURCEDIR)/cmd/starter/c/include/*.h)

$(BUILDDIR)/.clean-starter: $(starter_CSOURCE)
	@echo " GO clean -cache"
	-$(V)$(GO) clean -cache 2>/dev/null
	$(V)touch $@


# starter
# Look at dependencies file changes via starter_deps
# because it means that a module was updated.
starter := $(BUILDDIR)/cmd/starter/c/starter
$(starter): $(BUILDDIR)/.clean-starter $(singularity_build_config) $(starter_deps) $(starter_SOURCE)
	@echo " GO" $@
	$(V)$(GO) build $(GO_MODFLAGS) $(GO_BUILDMODE) -tags "$(GO_TAGS)" $(GO_LDFLAGS) $(GO_GCFLAGS) $(GO_ASMFLAGS) \
		-o $@ $(SOURCEDIR)/cmd/starter/main_linux.go

starter_INSTALL := $(DESTDIR)$(LIBEXECDIR)/singularity/bin/starter
$(starter_INSTALL): $(starter)
	@echo " INSTALL" $@
	$(V)umask 0022 && mkdir -p $(@D)
	$(V)install -m 0755 $(starter) $@

CLEANFILES += $(starter)
INSTALLFILES += $(starter_INSTALL)
ALL += $(starter)


# sessiondir
sessiondir_INSTALL := $(DESTDIR)$(LOCALSTATEDIR)/singularity/mnt/session
$(sessiondir_INSTALL):
	@echo " INSTALL" $@
	$(V)umask 0022 && mkdir -p $@

INSTALLFILES += $(sessiondir_INSTALL)


# run-singularity script
run_singularity := $(SOURCEDIR)/scripts/run-singularity

run_singularity_INSTALL := $(DESTDIR)$(BINDIR)/run-singularity
$(run_singularity_INSTALL): $(run_singularity)
	@echo " INSTALL" $@
	$(V)umask 0022 && mkdir -p $(@D)
	$(V)install -m 0755 $< $@

INSTALLFILES += $(run_singularity_INSTALL)


# capability config file
capability_config_INSTALL := $(DESTDIR)$(SYSCONFDIR)/singularity/capability.json
$(capability_config_INSTALL):
	@echo " INSTALL" $@
	$(V)umask 0022 && mkdir -p $(@D)
	$(V)touch $@

INSTALLFILES += $(capability_config_INSTALL)


# syecl config file
syecl_config := $(SOURCEDIR)/internal/pkg/syecl/syecl.toml.example

syecl_config_INSTALL := $(DESTDIR)$(SYSCONFDIR)/singularity/ecl.toml
$(syecl_config_INSTALL): $(syecl_config)
	@echo " INSTALL" $@
	$(V)umask 0022 && mkdir -p $(@D)
	$(V)install -m 0644 $< $@

INSTALLFILES += $(syecl_config_INSTALL)


# seccomp profile
seccomp_profile := $(SOURCEDIR)/etc/seccomp-profiles/default.json

seccomp_profile_INSTALL := $(DESTDIR)$(SYSCONFDIR)/singularity/seccomp-profiles/default.json
$(seccomp_profile_INSTALL): $(seccomp_profile)
	@echo " INSTALL" $@
	$(V)umask 0022 && mkdir -p $(@D)
	$(V)install -m 0644 $< $@

INSTALLFILES += $(seccomp_profile_INSTALL)


# nvidia liblist config file
nvidia_liblist := $(SOURCEDIR)/etc/nvliblist.conf

nvidia_liblist_INSTALL := $(DESTDIR)$(SYSCONFDIR)/singularity/nvliblist.conf
$(nvidia_liblist_INSTALL): $(nvidia_liblist)
	@echo " INSTALL" $@
	$(V)umask 0022 && mkdir -p $(@D)
	$(V)install -m 0644 $< $@

INSTALLFILES += $(nvidia_liblist_INSTALL)


# rocm liblist config file
rocm_liblist := $(SOURCEDIR)/etc/rocmliblist.conf

 rocm_liblist_INSTALL := $(DESTDIR)$(SYSCONFDIR)/singularity/rocmliblist.conf
$(rocm_liblist_INSTALL): $(rocm_liblist)
	@echo " INSTALL" $@
	$(V)umask 0022 && mkdir -p $(@D)
	$(V)install -m 0644 $< $@

INSTALLFILES += $(rocm_liblist_INSTALL)


# cgroups config file
cgroups_config := $(SOURCEDIR)/internal/pkg/cgroups/example/cgroups.toml

cgroups_config_INSTALL := $(DESTDIR)$(SYSCONFDIR)/singularity/cgroups/cgroups.toml
$(cgroups_config_INSTALL): $(cgroups_config)
	@echo " INSTALL" $@
	$(V)umask 0022 && mkdir -p $(@D)
	$(V)install -m 0644 $< $@

INSTALLFILES += $(cgroups_config_INSTALL)

# global keyring
global_keyring := $(SOURCEDIR)/etc/global-pgp-public

global_keyring_INSTALL := $(DESTDIR)$(SYSCONFDIR)/singularity/global-pgp-public
$(global_keyring_INSTALL): $(global_keyring)
	@echo " INSTALL" $@
	$(V)umask 0022 && mkdir -p $(@D)
	$(V)install -m 0644 $< $@

INSTALLFILES += $(global_keyring_INSTALL)

