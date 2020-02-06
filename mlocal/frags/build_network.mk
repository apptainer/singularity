# This file contains rules for building CNI plugins which can enable
#   different networking functions between container(s) and the host

singularity_REPO := github.com/sylabs/singularity

cni_builddir := $(BUILDDIR_ABSPATH)/cni
cni_install_DIR := $(DESTDIR)$(LIBEXECDIR)/singularity/cni
cni_plugins := $(shell grep '^	_' $(SOURCEDIR)/internal/pkg/runtime/engine/singularity/plugins_linux.go | cut -d\" -f2)
cni_plugins_EXECUTABLES := $(addprefix $(cni_builddir)/, $(notdir $(cni_plugins)))
cni_plugins_INSTALL := $(addprefix $(cni_install_DIR)/, $(notdir $(cni_plugins)))
cni_config_LIST := $(SOURCEDIR)/etc/network/00_bridge.conflist \
                   $(SOURCEDIR)/etc/network/10_ptp.conflist \
                   $(SOURCEDIR)/etc/network/20_ipvlan.conflist \
                   $(SOURCEDIR)/etc/network/30_macvlan.conflist \
                   $(SOURCEDIR)/etc/network/40_fakeroot.conflist
cni_config_INSTALL := $(DESTDIR)$(SYSCONFDIR)/singularity/network

.PHONY: cniplugins
cniplugins:
	$(V)umask 0022 && mkdir -p $(cni_builddir)
	$(V)for p in $(cni_plugins); do \
		name=`basename $$p`; \
		cniplugin=$(cni_builddir)/$$name; \
		if [ ! -f $$cniplugin ]; then \
			echo " CNI PLUGIN" $$name; \
		$(GO) build $(GO_MODFLAGS) $(GO_BUILDMODE) -tags "$(GO_TAGS)" $(GO_LDFLAGS) $(GO_GCFLAGS) $(GO_ASMFLAGS) \
			-o $$cniplugin $$p; \
		fi \
	done

$(cni_plugins_INSTALL): $(cni_plugins_EXECUTABLES)
	@echo " INSTALL CNI PLUGIN" $@
	$(V)umask 0022 && mkdir -p $(@D)
	$(V)install -m 0755 $(cni_builddir)/$(@F) $@

$(cni_config_INSTALL): $(cni_config_LIST)
	@echo " INSTALL CNI CONFIGURATION FILES"
	$(V)umask 0022 && mkdir -p $(cni_config_INSTALL)
	$(V)install -m 0644 $? $@

CLEANFILES += $(cni_plugins_EXECUTABLES)
INSTALLFILES += $(cni_plugins_INSTALL) $(cni_config_INSTALL)
ALL += cniplugins
