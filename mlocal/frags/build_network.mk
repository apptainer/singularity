# This file contains rules for building CNI plugins which can enable
#   different networking functions between container(s) and the host

singularity_REPO := github.com/sylabs/singularity

cni_builddir := $(BUILDDIR_ABSPATH)/cni
cni_install_DIR := $(DESTDIR)$(LIBEXECDIR)/singularity/cni
cni_vendor_GOPATH := $(singularity_REPO)/vendor/github.com/containernetworking/plugins/plugins
cni_plugins_GOPATH := $(cni_vendor_GOPATH)/meta/bandwidth \
                      $(cni_vendor_GOPATH)/main/bridge \
                      $(cni_vendor_GOPATH)/ipam/dhcp \
                      $(cni_vendor_GOPATH)/meta/flannel \
                      $(cni_vendor_GOPATH)/main/host-device \
                      $(cni_vendor_GOPATH)/ipam/host-local \
                      $(cni_vendor_GOPATH)/main/ipvlan \
                      $(cni_vendor_GOPATH)/main/loopback \
                      $(cni_vendor_GOPATH)/main/macvlan \
                      $(cni_vendor_GOPATH)/meta/portmap \
                      $(cni_vendor_GOPATH)/main/ptp \
                      $(cni_vendor_GOPATH)/ipam/static \
                      $(cni_vendor_GOPATH)/meta/tuning \
                      $(cni_vendor_GOPATH)/main/vlan
cni_plugins_EXECUTABLES := $(cni_builddir)/bandwidth \
                           $(cni_builddir)/bridge \
                           $(cni_builddir)/dhcp \
                           $(cni_builddir)/flannel \
                           $(cni_builddir)/host-device \
                           $(cni_builddir)/host-local \
                           $(cni_builddir)/ipvlan \
                           $(cni_builddir)/loopback \
                           $(cni_builddir)/macvlan \
                           $(cni_builddir)/portmap \
                           $(cni_builddir)/ptp \
                           $(cni_builddir)/static \
                           $(cni_builddir)/tuning \
                           $(cni_builddir)/vlan
cni_plugins_INSTALL := $(cni_install_DIR)/bandwidth \
                       $(cni_install_DIR)/bridge \
                       $(cni_install_DIR)/dhcp \
                       $(cni_install_DIR)/flannel \
                       $(cni_install_DIR)/host-device \
                       $(cni_install_DIR)/host-local \
                       $(cni_install_DIR)/ipvlan \
                       $(cni_install_DIR)/loopback \
                       $(cni_install_DIR)/macvlan \
                       $(cni_install_DIR)/portmap \
                       $(cni_install_DIR)/ptp \
                       $(cni_install_DIR)/static \
                       $(cni_install_DIR)/tuning \
                       $(cni_install_DIR)/vlan
cni_config_LIST := $(SOURCEDIR)/etc/network/00_bridge.conflist \
                   $(SOURCEDIR)/etc/network/10_ptp.conflist \
                   $(SOURCEDIR)/etc/network/20_ipvlan.conflist \
                   $(SOURCEDIR)/etc/network/30_macvlan.conflist
cni_config_INSTALL := $(DESTDIR)$(SYSCONFDIR)/singularity/network

.PHONY: cniplugins
cniplugins:
	$(V)install -d $(cni_builddir)
	$(V)for p in $(cni_plugins_GOPATH); do \
		name=`basename $$p`; \
		cniplugin=$(cni_builddir)/$$name; \
		if [ ! -f $$cniplugin ]; then \
			echo " CNI PLUGIN" $$name; \
			go build $(GO_BUILDMODE) -tags "$(GO_TAGS)" $(GO_LDFLAGS) -o $$cniplugin $$p; \
		fi \
	done

$(cni_plugins_INSTALL): $(cni_plugins_EXECUTABLES)
	@echo " INSTALL CNI PLUGIN" $@
	$(V)install -d $(@D)
	$(V)install -m 0755 $(cni_builddir)/$(@F) $@

$(cni_config_INSTALL): $(cni_config_LIST)
	@echo " INSTALL CNI CONFIGURATION FILES"
	$(V)install -d $(cni_config_INSTALL)
	$(V)install -m 0644 $? $@

CLEANFILES += $(cni_plugins_EXECUTABLES)
INSTALLFILES += $(cni_plugins_INSTALL) $(cni_config_INSTALL)
ALL += cniplugins
