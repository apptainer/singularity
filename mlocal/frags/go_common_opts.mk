# go tool default build options
GO_TAGS := -tags "containers_image_openpgp apparmor selinux"
GO_LDFLAGS :=
GO_BUILDMODE := -buildmode=default

CGO_CPPFLAGS := -I$(BUILDDIR) -I$(SOURCEDIR)/cmd/starter/c -I$(SOURCEDIR)/internal/pkg/runtime/c/lib
CGO_CPPFLAGS += -include $(BUILDDIR_ABSPATH)/config.h

CGO_LDFLAGS := -L$(BUILDDIR_ABSPATH)/lib -L$(BUILDDIR) -lruntime

export CGO_CPPFLAGS CGO_LDFLAGS
