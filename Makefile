topdir  = $(shell pwd)
coredir = $(topdir)/core
buildtree = $(coredir)/buildtree

CONFIG_PKG = github.com/singularityware/singularity/pkg/configs
CONFIG_LDFLAGS = -X $(CONFIG_PKG).BUILDTREE=$(buildtree) -X $(CONFIG_PKG).LIBEXECDIR=/tmp/testing

CGO_CPPFLAGS = -I$(buildtree) -I$(coredir) -I$(coredir)/lib
CGO_LDFLAGS = -L$(buildtree)/lib

GO_TAGS = "containers_image_openpgp"
GO_BINS = $(buildtree)/singularity $(buildtree)/sbuild $(buildtree)/scontainer $(buildtree)/smaster

.PHONEY: all dep c clean
all: $(GO_BINS) c

dep:
	dep ensure -vendor-only

$(buildtree)/singularity: c
	@echo "go build $(buildtree)/singularity"
	@export CGO_CPPFLAGS="$(CGO_CPPFLAGS)" CGO_LDFLAGS="$(CGO_LDFLAGS)" && \
	go build -ldflags "$(CONFIG_LDFLAGS)" --tags "$(GO_TAGS)" -o $(buildtree)/singularity $(topdir)/cmd/cli/cli.go

$(buildtree)/sbuild: c
	@echo "go build $(buildtree)/sbuild"
	@export CGO_CPPFLAGS="$(CGO_CPPFLAGS)" CGO_LDFLAGS="$(CGO_LDFLAGS)" && \
	go build -ldflags "$(CONFIG_LDFLAGS)" -o $(buildtree)/sbuild $(topdir)/cmd/sbuild/sbuild.go

$(buildtree)/scontainer: c
	@echo "go build $(buildtree)/scontainer"
	@export CGO_CPPFLAGS="$(CGO_CPPFLAGS)" CGO_LDFLAGS="$(CGO_LDFLAGS)" && \
	go build -ldflags "$(CONFIG_LDFLAGS)" -o $(buildtree)/scontainer $(coredir)/runtime/go/scontainer.go

$(buildtree)/smaster: c
	@echo "go build $(buildtree)/smaster"
	@export CGO_CPPFLAGS="$(CGO_CPPFLAGS)" CGO_LDFLAGS="$(CGO_LDFLAGS)" && \
	go build -ldflags "$(CONFIG_LDFLAGS)" -o $(buildtree)/smaster $(coredir)/runtime/go/smaster.go

$(buildtree)/librpc.so:
	@echo "go build $(buildtree)/librpc.so"
	@go build -ldflags="-s -w" -buildmode=c-shared -o $(buildtree)/librpc.so $(coredir)/runtime/go/rpc.go

c: $(buildtree)/Makefile $(buildtree)/librpc.so
	@make -j $(shell nproc 2>/dev/null || echo 1) -C $(buildtree)

$(buildtree)/Makefile:
	@cd $(coredir) && ./mconfig -b $(buildtree) && cd $(topdir)

clean:
	sudo rm -rf $(buildtree)
