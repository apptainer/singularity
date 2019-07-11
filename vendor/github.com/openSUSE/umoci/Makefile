# umoci: Umoci Modifies Open Containers' Images
# Copyright (C) 2016, 2017, 2018 SUSE LLC.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Use bash, so that we can do process substitution.
SHELL = /bin/bash

# Go tools.
GO ?= go
GO_MD2MAN ?= go-md2man
export GO111MODULE=off

# Set up the ... lovely ... GOPATH hacks.
PROJECT := github.com/openSUSE/umoci
CMD := ${PROJECT}/cmd/umoci

# We use Docker because Go is just horrific to deal with.
UMOCI_IMAGE := umoci_dev
DOCKER_RUN := docker run --rm -it --security-opt apparmor:unconfined --security-opt label:disable -v ${PWD}:/go/src/${PROJECT}

# Output directory.
BUILD_DIR ?= .

# Release information.
GPG_KEYID ?=

# Version information.
VERSION := $(shell cat ./VERSION)
COMMIT_NO := $(shell git rev-parse HEAD 2> /dev/null || true)
COMMIT := $(if $(shell git status --porcelain --untracked-files=no),"${COMMIT_NO}-dirty","${COMMIT_NO}")

# Basic build flags.
BUILD_FLAGS ?=
BASE_FLAGS := ${BUILD_FLAGS} -tags "${BUILDTAGS}"
BASE_LDFLAGS := -s -w -X main.gitCommit=${COMMIT} -X main.version=${VERSION}

# Specific build flags for build type.
DYN_BUILD_FLAGS := ${BASE_FLAGS} -buildmode=pie -ldflags "${BASE_LDFLAGS}"
TEST_BUILD_FLAGS := ${BASE_FLAGS} -buildmode=pie -ldflags "${BASE_LDFLAGS} -X ${PROJECT}/pkg/testutils.binaryType=test"
STATIC_BUILD_FLAGS := ${BASE_FLAGS} -ldflags "${BASE_LDFLAGS} -extldflags '-static'"

# Installation directories.
DESTDIR ?=
PREFIX ?=/usr
BINDIR ?=$(PREFIX)/bin
MANDIR ?=$(PREFIX)/share/man

.DEFAULT: umoci

GO_SRC = $(shell find . -name \*.go)

# NOTE: If you change these make sure you also update local-validate-build.

umoci: $(GO_SRC)
	$(GO) build ${DYN_BUILD_FLAGS} -o $(BUILD_DIR)/$@ ${CMD}

umoci.static: $(GO_SRC)
	env CGO_ENABLED=0 $(GO) build ${STATIC_BUILD_FLAGS} -o $(BUILD_DIR)/$@ ${CMD}

umoci.cover: $(GO_SRC)
	$(GO) test -c -cover -covermode=count -coverpkg=./... ${TEST_BUILD_FLAGS} -o $(BUILD_DIR)/$@ ${CMD}

.PHONY: release
release:
	hack/release.sh -S "$(GPG_KEYID)" -r release/$(VERSION) -v $(VERSION)

.PHONY: install
install: umoci doc
	install -D -m0755 umoci $(DESTDIR)/$(BINDIR)/umoci
	-for man in $(MANPAGES); do \
		filename="$$(basename -- "$$man")"; \
		target="$(DESTDIR)/$(MANDIR)/man$${filename##*.}/$$filename"; \
		install -D -m0644 "$$man" "$$target"; \
		gzip -9f "$$target"; \
	 done

.PHONY: uninstall
uninstall:
	rm -f $(DESTDIR)/$(BINDIR)/umoci
	-rm -f $(DESTDIR)/$(MANDIR)/man*/umoci*

.PHONY: clean
clean:
	rm -f umoci umoci.static umoci.cov*
	rm -f $(MANPAGES)

.PHONY: validate
validate: umociimage
	$(DOCKER_RUN) $(UMOCI_IMAGE) make local-validate

.PHONY: local-validate
local-validate: local-validate-git local-validate-go local-validate-reproducible local-validate-build

# TODO: Remove the special-case ignored system/* warnings.
.PHONY: local-validate-go
local-validate-go:
	@type gofmt     >/dev/null 2>/dev/null || (echo "ERROR: gofmt not found." && false)
	test -z "$$(gofmt -s -l . | grep -vE '^vendor/|^third_party/' | tee /dev/stderr)"
	@type golint    >/dev/null 2>/dev/null || (echo "ERROR: golint not found." && false)
	test -z "$$(golint $(PROJECT)/... | grep -vE '/vendor/|/third_party/' | tee /dev/stderr)"
	@go doc cmd/vet >/dev/null 2>/dev/null || (echo "ERROR: go vet not found." && false)
	test -z "$$($(GO) vet $$($(GO) list $(PROJECT)/... | grep -vE '/vendor/|/third_party/') 2>&1 | tee /dev/stderr)"
	@type gosec     >/dev/null 2>/dev/null || (echo "ERROR: gosec not found." && false)
	test -z "$$(gosec -quiet -exclude=G301,G302,G304 $$GOPATH/$(PROJECT)/... | tee /dev/stderr)"
	./hack/test-vendor.sh

EPOCH_COMMIT ?= 97ecdbd53dcb72b7a0d62196df281f131dc9eb2f
.PHONY: local-validate-git
local-validate-git:
	@type git-validation > /dev/null 2>/dev/null || (echo "ERROR: git-validation not found." && false)
ifdef TRAVIS_COMMIT_RANGE
	git-validation -q -run DCO,short-subject
else
	git-validation -q -run DCO,short-subject -range $(EPOCH_COMMIT)..HEAD
endif

# Make sure that our builds are reproducible even if you wait between them and
# the modified time of the files is different.
.PHONY: local-validate-reproducible
local-validate-reproducible:
	mkdir -p .tmp-validate
	make -B umoci && cp umoci .tmp-validate/umoci.a
	@echo sleep 10s
	@sleep 10s && touch $(GO_SRC)
	make -B umoci && cp umoci .tmp-validate/umoci.b
	diff -s .tmp-validate/umoci.{a,b}
	sha256sum .tmp-validate/umoci.{a,b}
	rm -r .tmp-validate/umoci.{a,b}

.PHONY: local-validate-build
local-validate-build:
	$(GO) build ${DYN_BUILD_FLAGS} -o /dev/null ${CMD}
	env CGO_ENABLED=0 $(GO) build ${STATIC_BUILD_FLAGS} -o /dev/null ${CMD}
	$(GO) test -run nothing ${DYN_BUILD_FLAGS} $(PROJECT)/...

MANPAGES_MD := $(wildcard doc/man/*.md)
MANPAGES    := $(MANPAGES_MD:%.md=%)

doc/man/%.1: doc/man/%.1.md
	$(GO_MD2MAN) -in $< -out $@

.PHONY: doc
doc: $(MANPAGES)

# Used for tests.
DOCKER_IMAGE :=opensuse/amd64:tumbleweed

.PHONY: umociimage
umociimage:
	docker build -t $(UMOCI_IMAGE) --build-arg DOCKER_IMAGE=$(DOCKER_IMAGE) .

ifndef COVERAGE
COVERAGE := $(shell mktemp --dry-run umoci.cov.XXXXXX)
endif

.PHONY: test-unit
test-unit: umociimage
	touch $(COVERAGE) && chmod a+rw $(COVERAGE)
	$(DOCKER_RUN) -e COVERAGE=$(COVERAGE) --cap-add=SYS_ADMIN $(UMOCI_IMAGE) make local-test-unit
	$(DOCKER_RUN) -e COVERAGE=$(COVERAGE) -u 1000:1000 --cap-drop=all $(UMOCI_IMAGE) make local-test-unit

.PHONY: local-test-unit
local-test-unit:
	GO=$(GO) COVER=1 hack/test-unit.sh

.PHONY: test-integration
test-integration: umociimage
	touch $(COVERAGE) && chmod a+rw $(COVERAGE)
	$(DOCKER_RUN) -e COVERAGE=$(COVERAGE) $(UMOCI_IMAGE) make TESTS="${TESTS}" local-test-integration
	$(DOCKER_RUN) -e COVERAGE=$(COVERAGE) -u 1000:1000 --cap-drop=all $(UMOCI_IMAGE) make TESTS="${TESTS}" local-test-integration

.PHONY: local-test-integration
local-test-integration: umoci.cover
	TESTS="${TESTS}" COVER=1 hack/test-integration.sh

shell: umociimage
	$(DOCKER_RUN) $(UMOCI_IMAGE) bash

.PHONY: ci
ci: umoci umoci.cover doc local-validate test-unit test-integration
	hack/ci-coverage.sh $(COVERAGE)
