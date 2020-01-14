# This file contains installation rule for starter-suid binary. In order to 
#   include this file, Makefile_runtime.stub MUST first be included.

# starter suid
starter_suid := $(BUILDDIR)/cmd/starter/c/starter-suid
starter_suid_INSTALL := $(DESTDIR)$(LIBEXECDIR)/singularity/bin/starter-suid

$(starter_suid): $(starter)
	@echo " GO" $@; echo "    [+] GO_TAGS" \"$(GO_TAGS_SUID)\"
	$(V)$(GO) build $(GO_MODFLAGS) $(GO_BUILDMODE) -tags "$(GO_TAGS_SUID)" $(GO_LDFLAGS) $(GO_GCFLAGS) $(GO_ASMFLAGS) \
		-o $@ $(SOURCEDIR)/cmd/starter/main_linux.go

$(starter_suid_INSTALL): $(starter_suid)
	@echo " INSTALL SUID" $@
	$(V)umask 0022 && mkdir -p $(@D)
	$(V)install $(starter_suid) $(starter_suid_INSTALL)
	@if [ `id -u` -ne 0 ] ; then \
		echo "$(starter_suid_INSTALL) -- installed with incorrect permissions"; \
	else \
		chmod 4755 $(starter_suid_INSTALL); \
	fi

CLEANFILES += $(starter_suid)
INSTALLFILES += $(starter_suid_INSTALL)
ALL += $(starter_suid)
