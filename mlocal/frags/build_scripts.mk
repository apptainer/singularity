# This file contains generation rules for scripts

# go-test script
$(SOURCEDIR)/scripts/go-test: export GO := $(GO)
$(SOURCEDIR)/scripts/go-test: export GO111MODULE := $(GO111MODULE)
$(SOURCEDIR)/scripts/go-test: export GOFLAGS := $(GOFLAGS)
$(SOURCEDIR)/scripts/go-test: export GO_TAGS := $(GO_TAGS)
$(SOURCEDIR)/scripts/go-test: $(SOURCEDIR)/scripts/go-test.in $(SOURCEDIR)/scripts/expand-env.go
	@echo ' GEN $@'
	$(V) $(GO) $(GO_MODFLAGS) run $(SOURCEDIR)/scripts/expand-env.go < $< > $@
	$(V) chmod +x $@

ALL += $(SOURCEDIR)/scripts/go-test
