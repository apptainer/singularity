# This file contains installation rule for starter-suid binary. In order to 
#   include this file, Makefile_runtime.stub MUST first be included.

# starter suid
starter_suid := $(BUILDDIR)/cmd/starter/c/starter-suid
starter_suid_INSTALL := $(DESTDIR)$(LIBEXECDIR)/singularity/bin/starter-suid

$(starter_suid): $(BUILDDIR)/.clean-starter $(singularity_build_config) $(starter_SOURCE)
	@echo " GO" $@; echo "    [+] GO_TAGS" \"$(GO_TAGS_SUID)\"
	$(V)$(GO) build $(GO_MODFLAGS) $(GO_BUILDMODE) -tags "$(GO_TAGS_SUID)" $(GO_LDFLAGS) $(GO_GCFLAGS) $(GO_ASMFLAGS) \
		-o $@ $(SOURCEDIR)/cmd/starter/main_linux.go

$(starter_suid_INSTALL): $(starter_suid)
	@if [ `id -u` -ne 0 -a -z "${RPM_BUILD_ROOT}" ] ; then \
		echo "SUID binary requires to execute make install as root, use sudo make install to finish installation"; \
		exit 1 ; \
	fi
	@echo " INSTALL SUID" $@
	$(V)install -d $(@D)
	$(V)install -m 4755 $(starter_suid) $(starter_suid_INSTALL)

CLEANFILES += $(starter_suid)
INSTALLFILES += $(starter_suid_INSTALL)
ALL += $(starter_suid)
