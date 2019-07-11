
BUILD := gomtree
BUILDPATH := github.com/vbatts/go-mtree/cmd/gomtree
CWD := $(shell pwd)
SOURCE_FILES := $(shell find . -type f -name "*.go")
CLEAN_FILES := *~
TAGS :=
ARCHES := linux,386 linux,amd64 linux,arm linux,arm64 openbsd,amd64 windows,amd64 darwin,amd64

default: build validation

.PHONY: validation
validation: .test .lint .vet .cli.test

.PHONY: validation.tags
validation.tags: .test.tags .vet.tags .cli.test

.PHONY: test
test: .test

CLEAN_FILES += .test .test.tags

.test: $(SOURCE_FILES)
	go test -v $$(glide novendor) && touch $@

.test.tags: $(SOURCE_FILES)
	set -e ; for tag in $(TAGS) ; do go test -tags $$tag -v $$(glide novendor) ; done && touch $@

.PHONY: lint
lint: .lint

CLEAN_FILES += .lint

.lint: $(SOURCE_FILES)
	set -e ; for dir in $$(glide novendor) ; do golint -set_exit_status $$dir ; done && touch $@

.PHONY: vet
vet: .vet .vet.tags

CLEAN_FILES += .vet .vet.tags

.vet: $(SOURCE_FILES)
	go vet $$(glide novendor) && touch $@

.vet.tags: $(SOURCE_FILES)
	set -e ; for tag in $(TAGS) ; do go vet -tags $$tag -v $$(glide novendor) ; done && touch $@

.PHONY: cli.test
cli.test: .cli.test

CLEAN_FILES += .cli.test .cli.test.tags

.cli.test: $(BUILD) $(wildcard ./test/cli/*.sh)
	@go run ./test/cli.go ./test/cli/*.sh && touch $@

.cli.test.tags: $(BUILD) $(wildcard ./test/cli/*.sh)
	@set -e ; for tag in $(TAGS) ; do go run -tags $$tag ./test/cli.go ./test/cli/*.sh ; done && touch $@

.PHONY: build
build: $(BUILD)

$(BUILD): $(SOURCE_FILES)
	go build -o $(BUILD) $(BUILDPATH)

install.tools:
	go get -u -v github.com/Masterminds/glide
	go get -u -v golang.org/x/lint/golint

./bin:
	mkdir -p $@

CLEAN_FILES += bin

build.arches: ./bin
	@set -e ;\
	for pair in $(ARCHES); do \
	p=$$(echo $$pair | cut -d , -f 1);\
	a=$$(echo $$pair | cut -d , -f 2);\
	echo "Building $$p/$$a ...";\
	GOOS=$$p GOARCH=$$a go build -o ./bin/gomtree.$$p.$$a $(BUILDPATH) ;\
	done

clean:
	rm -rf $(BUILD) $(CLEAN_FILES)

